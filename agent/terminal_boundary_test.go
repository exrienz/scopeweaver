package agent

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"strings"
	"testing"
	"time"

	"github.com/Autumn-27/artex/db"

	"github.com/Autumn-27/norma/agentcore"
	"github.com/Autumn-27/norma/compaction"
	"github.com/Autumn-27/norma/harness"
	"github.com/Autumn-27/norma/llm"
	"github.com/Autumn-27/norma/permission"
	actool "github.com/Autumn-27/norma/tool"
)

// Compaction boundary (Gap 1): Compactor.Pre runs BEFORE QueryDeps.CallModel
// (norma query.go 407-410) and the default/noa summarizers call the Provider
// directly, so the model-call gate alone cannot prevent post-proof compaction
// inference. The wrapped compactor skips Pre only when the gate would fire;
// ordinary compaction and reactive overflow recovery are forwarded untouched.

// countingCompactor is a forwarding no-op compactor that counts calls.
type countingCompactor struct {
	preCalls      int
	reactiveCalls int
	viewCalls     int
	overflowErr   error
}

func (c *countingCompactor) Pre(_ context.Context, msgs []llm.Message, _ int) []llm.Message {
	c.preCalls++
	return msgs
}
func (c *countingCompactor) Reactive(_ context.Context, msgs []llm.Message) ([]llm.Message, bool) {
	c.reactiveCalls++
	return msgs, true
}
func (c *countingCompactor) IsOverflow(err error) bool {
	return c.overflowErr != nil && err == c.overflowErr
}

type countingViewCompactor struct{ *countingCompactor }

func (c countingViewCompactor) View(_ context.Context, msgs []llm.Message) []llm.Message {
	c.viewCalls++
	return msgs
}

// Ordinary pre-proof compaction is forwarded: every loop iteration runs Pre
// before its model call; a non-View compactor does not gain ContextView.
func TestOrdinaryPreProofCompactionIsForwarded(t *testing.T) {
	ts := NewToolSet(nil, "")
	cmp := &countingCompactor{}
	noop := actool.Build(actool.Spec{
		Name: "noop_probe",
		Run: func(context.Context, json.RawMessage, *actool.ToolContext) (actool.Result, error) {
			return actool.Text("noop"), nil
		},
	})
	provider := &twoTurnProvider{firstTool: "noop_probe"}
	opts := agentcore.Options{Provider: provider, SystemPrompt: []string{"probe"}, Tools: []actool.CoreTool{noop}, PermissionMode: permission.ModeBypass, Compactor: cmp}
	observeProof := installTerminalProofGateOn(&opts, ts)
	if _, _, err := captureRun(context.Background(), opts, "run", func(db.Activity) {}, observeProof); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 2 {
		t.Fatalf("model calls = %d, want 2", provider.calls)
	}
	if cmp.preCalls != 2 {
		t.Fatalf("Pre calls = %d, want 2 (forwarded before every model call)", cmp.preCalls)
	}
	if _, ok := opts.Compactor.(harness.ContextView); ok {
		t.Fatal("non-View compactor must not gain ContextView through wrapping")
	}
}

// Post-proof: the iteration that would issue the next request skips Pre and the
// gate answers the model call locally — no provider summary, no main request.
func TestPostProofCompactionPreSkippedNoFurtherProviderWork(t *testing.T) {
	ts := NewToolSet(nil, "")
	cmp := &countingCompactor{}
	provider := &scriptedProvider{script: &proofScript{toolName: "terminal_proof_probe", executed: make(chan struct{})}}
	probe := actool.Build(actool.Spec{
		Name: "terminal_proof_probe",
		Run: func(_ context.Context, _ json.RawMessage, tc *actool.ToolContext) (actool.Result, error) {
			ts.GoalMet = true
			ts.markTerminalProof(tc.ToolUseID)
			close(provider.script.executed)
			return actool.Text("goal proven"), nil
		},
	})
	opts := agentcore.Options{Provider: provider, SystemPrompt: []string{"probe"}, Tools: []actool.CoreTool{probe}, PermissionMode: permission.ModeBypass, Compactor: cmp}
	observeProof := installTerminalProofGateOn(&opts, ts)
	if _, _, err := captureRun(context.Background(), opts, "run", func(db.Activity) {}, observeProof); err != nil {
		t.Fatal(err)
	}
	if provider.callsCount() != 1 {
		t.Fatalf("provider calls = %d, want 1 (gate; no post-proof compaction summary either)", provider.callsCount())
	}
	if cmp.preCalls != 1 {
		t.Fatalf("Pre calls = %d, want 1 (pre-proof only; the post-proof iteration must skip Pre)", cmp.preCalls)
	}
}

// Selection contract: explicit Compactor wins (duplicate config cleared); a
// Compaction config builds precisely the SDK default and is wrapped (config
// cleared); neither stays absent; optional ContextView survives wrapping.
func TestCompactorWrapSelectionAndContextView(t *testing.T) {
	ts := NewToolSet(nil, "")
	provider := &scriptedProvider{script: &proofScript{toolName: "terminal_proof_probe", executed: make(chan struct{})}}

	withConfig := agentcore.Options{Provider: provider, Compaction: &compaction.Config{}}
	installTerminalProofGateOn(&withConfig, ts)
	if withConfig.Compactor == nil {
		t.Fatal("config-only options must gain a wrapped SDK-default compactor")
	}
	if withConfig.Compaction != nil {
		t.Fatal("duplicate Compaction config must be cleared")
	}
	if _, ok := withConfig.Compactor.(*terminalProofCompactor); !ok {
		t.Fatalf("config-only options must wrap the SDK default compactor, got %T", withConfig.Compactor)
	}

	explicit := &countingCompactor{}
	withExplicit := agentcore.Options{Provider: provider, Compactor: explicit, Compaction: &compaction.Config{}}
	installTerminalProofGateOn(&withExplicit, ts)
	if withExplicit.Compaction != nil {
		t.Fatal("explicit Compactor must clear the duplicate config")
	}
	if base, ok := withExplicit.Compactor.(*terminalProofCompactor); !ok || base.inner != explicit {
		t.Fatalf("explicit compactor must be wrapped in place, got %T", withExplicit.Compactor)
	}

	view := &countingCompactor{}
	withView := agentcore.Options{Provider: provider, Compactor: countingViewCompactor{view}}
	installTerminalProofGateOn(&withView, ts)
	wrapped, ok := withView.Compactor.(harness.ContextView)
	if !ok {
		t.Fatal("ContextView presence must survive wrapping")
	}
	_ = wrapped.View(context.Background(), nil)
	if view.viewCalls != 1 {
		t.Fatal("View must forward to the wrapped compactor")
	}

	absent := agentcore.Options{Provider: provider}
	installTerminalProofGateOn(&absent, ts)
	if absent.Compactor != nil || absent.Compaction != nil {
		t.Fatal("compaction must stay absent when neither form is set")
	}
}

// Recoverable overflow boundary (Gap 2): a proof tool can run and arm during a
// response that then fails with a recoverable prompt-too-long error. The SDK
// retries WITHOUT recording that response's assistant/tool-result messages, so
// the arming must not produce a local success turn: the retry goes upstream with
// real recovery semantics, and the invalidation survives a later clean response.

var errProofOverflow = errors.New("simulated prompt-too-long after proof")

// overflowProvider: turn 1 emits the proof tool call, waits for the tool to run
// (arming), then fails with errProofOverflow. Later turns follow extra.
type overflowProvider struct {
	calls    int
	executed chan struct{}
	toolName string
	extra    func(call int, yield func(llm.StreamEvent, error) bool)
}

func (p *overflowProvider) Stream(ctx context.Context, _ llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
	return func(yield func(llm.StreamEvent, error) bool) { p.stream(ctx, yield) }
}
func (p *overflowProvider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
	return streamViaComplete(p.Stream, ctx, req)
}
func (p *overflowProvider) stream(_ context.Context, yield func(llm.StreamEvent, error) bool) {
	p.calls++
	if p.calls == 1 {
		for _, ev := range []llm.StreamEvent{
			{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 10}},
			{Type: llm.SEToolUseStart, ToolID: "call-1", ToolName: p.toolName},
			{Type: llm.SEToolInputJSON, Text: "{}"},
			{Type: llm.SETextDelta, Text: "proving"},
		} {
			if !yield(ev, nil) {
				return
			}
		}
		<-p.executed // tool ran and armed while the response is still open
		yield(llm.StreamEvent{}, errProofOverflow)
		return
	}
	if p.extra != nil {
		p.extra(p.calls, yield)
		return
	}
	for _, ev := range []llm.StreamEvent{
		{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 20}},
		{Type: llm.SETextDelta, Text: "recovered"},
		{Type: llm.SEMessageDelta, StopReason: "end_turn", Usage: llm.Usage{OutputTokens: 3}},
	} {
		if !yield(ev, nil) {
			return
		}
	}
}

// twoTurnProvider: turn 1 is a plain tool call (no arming); turn 2 ends.
type twoTurnProvider struct {
	calls     int
	firstTool string
}

func (p *twoTurnProvider) Stream(ctx context.Context, _ llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
	return func(yield func(llm.StreamEvent, error) bool) {
		p.calls++
		if p.calls == 1 {
			for _, ev := range []llm.StreamEvent{
				{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 10}},
				{Type: llm.SEToolUseStart, ToolID: "call-1", ToolName: p.firstTool},
				{Type: llm.SEToolInputJSON, Text: "{}"},
				{Type: llm.SEMessageDelta, StopReason: "tool_use", Usage: llm.Usage{OutputTokens: 2}},
			} {
				if !yield(ev, nil) {
					return
				}
			}
			return
		}
		for _, ev := range []llm.StreamEvent{
			{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 20}},
			{Type: llm.SETextDelta, Text: "done"},
			{Type: llm.SEMessageDelta, StopReason: "end_turn", Usage: llm.Usage{OutputTokens: 1}},
		} {
			if !yield(ev, nil) {
				return
			}
		}
	}
}
func (p *twoTurnProvider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
	return streamViaComplete(p.Stream, ctx, req)
}

// The recoverable-overflow retry must be a REAL upstream call (a naive armed
// flag would answer it locally and mask the failed proof response), with
// recovery compaction forwarded and native completion afterwards.
func TestRecoverableOverflowAfterProofDoesNotLocalComplete(t *testing.T) {
	ts := NewToolSet(nil, "")
	cmp := &countingCompactor{overflowErr: errProofOverflow}
	provider := &overflowProvider{executed: make(chan struct{}), toolName: "terminal_proof_probe"}
	probe := actool.Build(actool.Spec{
		Name: "terminal_proof_probe",
		Run: func(_ context.Context, _ json.RawMessage, tc *actool.ToolContext) (actool.Result, error) {
			ts.GoalMet = true
			ts.markTerminalProof(tc.ToolUseID)
			close(provider.executed)
			return actool.Text("goal proven"), nil
		},
	})
	opts := agentcore.Options{Provider: provider, SystemPrompt: []string{"probe"}, Tools: []actool.CoreTool{probe}, PermissionMode: permission.ModeBypass, Compactor: cmp}
	observeProof := installTerminalProofGateOn(&opts, ts)
	var acts []db.Activity
	_, _, err := captureRun(context.Background(), opts, "prove", func(r db.Activity) { acts = append(acts, r) }, observeProof)
	if err != nil {
		t.Fatalf("overflow recovery must complete natively, got %v", err)
	}
	if provider.calls != 2 {
		t.Fatalf("upstream calls = %d, want 2 (the overflow retry must NOT become a local success turn)", provider.calls)
	}
	if cmp.reactiveCalls != 1 {
		t.Fatalf("reactive recovery calls = %d, want 1 (forwarded)", cmp.reactiveCalls)
	}
	if cmp.preCalls != 2 {
		t.Fatalf("Pre calls = %d, want 2 (pre-proof + recovery iteration, both forwarded)", cmp.preCalls)
	}
	if result := activityByKind(acts, "result"); result == nil || result.IsError {
		t.Fatalf("native completion after recovery must be non-error: %+v", result)
	}
	if ts.terminalProofReady() {
		t.Fatal("arming from the overflowed response must stay invalidated")
	}
}

// A later CLEAN response must not revive an arming that belonged to the failed
// response: the third model call is still issued upstream.
func TestOverflowInvalidatedArmingSurvivesCleanRetry(t *testing.T) {
	ts := NewToolSet(nil, "")
	cmp := &countingCompactor{overflowErr: errProofOverflow}
	provider := &overflowProvider{
		executed: make(chan struct{}),
		toolName: "terminal_proof_probe",
		extra: func(call int, yield func(llm.StreamEvent, error) bool) {
			if call == 2 { // clean retry with a NON-terminal tool call
				for _, ev := range []llm.StreamEvent{
					{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 20}},
					{Type: llm.SEToolUseStart, ToolID: "call-2", ToolName: "noop_probe"},
					{Type: llm.SEToolInputJSON, Text: "{}"},
					{Type: llm.SEMessageDelta, StopReason: "tool_use", Usage: llm.Usage{OutputTokens: 3}},
				} {
					if !yield(ev, nil) {
						return
					}
				}
				return
			}
			for _, ev := range []llm.StreamEvent{ // call 3
				{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 20}},
				{Type: llm.SETextDelta, Text: "still working"},
				{Type: llm.SEMessageDelta, StopReason: "end_turn", Usage: llm.Usage{OutputTokens: 2}},
			} {
				if !yield(ev, nil) {
					return
				}
			}
		},
	}
	probe := actool.Build(actool.Spec{
		Name: "terminal_proof_probe",
		Run: func(_ context.Context, _ json.RawMessage, tc *actool.ToolContext) (actool.Result, error) {
			ts.GoalMet = true
			ts.markTerminalProof(tc.ToolUseID)
			close(provider.executed)
			return actool.Text("goal proven"), nil
		},
	})
	noop := actool.Build(actool.Spec{
		Name: "noop_probe",
		Run: func(context.Context, json.RawMessage, *actool.ToolContext) (actool.Result, error) {
			return actool.Text("noop"), nil
		},
	})
	opts := agentcore.Options{Provider: provider, SystemPrompt: []string{"probe"}, Tools: []actool.CoreTool{probe, noop}, PermissionMode: permission.ModeBypass, Compactor: cmp}
	observeProof := installTerminalProofGateOn(&opts, ts)
	if _, _, err := captureRun(context.Background(), opts, "prove", func(db.Activity) {}, observeProof); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 3 {
		t.Fatalf("upstream calls = %d, want 3 (a clean retry must not revive the failed response's arming)", provider.calls)
	}
	if ts.terminalProofReady() {
		t.Fatal("arming from the overflowed response must remain invalidated after a clean retry")
	}
}

// Delayed tool completion: the proof handler is still running when the response
// fails; its late arming stamps the CURRENT (retry) call sequence, so it can
// only be honored if that later response completes cleanly — it never
// retroactively converts the failed turn, and the retry stays a real call.
func TestLateToolCompletionAfterStreamErrorArmsSafely(t *testing.T) {
	ts := NewToolSet(nil, "")
	cmp := &countingCompactor{overflowErr: errProofOverflow}
	provider := &overflowProvider{executed: make(chan struct{}), toolName: "terminal_proof_probe"}
	lateProbe := actool.Build(actool.Spec{
		Name: "terminal_proof_probe",
		Run: func(_ context.Context, _ json.RawMessage, tc *actool.ToolContext) (actool.Result, error) {
			time.Sleep(60 * time.Millisecond) // still completing when the stream errors
			ts.GoalMet = true
			ts.markTerminalProof(tc.ToolUseID)
			close(provider.executed)
			return actool.Text("goal proven (late)"), nil
		},
	})
	opts := agentcore.Options{Provider: provider, SystemPrompt: []string{"probe"}, Tools: []actool.CoreTool{lateProbe}, PermissionMode: permission.ModeBypass, Compactor: cmp}
	observeProof := installTerminalProofGateOn(&opts, ts)
	if _, _, err := captureRun(context.Background(), opts, "prove", func(db.Activity) {}, observeProof); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 2 {
		t.Fatalf("upstream calls = %d, want 2 (retry stays real; late arming never converts it)", provider.calls)
	}
}

// --- Origin-binding regressions (adapted from the independent stale-origin
// review; overlay/terminal_proof_stale_call_review_test.go and
// stale-preflight-20261017-02 receipts: real go exit 1 against the R14 freeze
// — the delayed-origin fixture FAILED with upstream calls=2, want 3, proving
// markTerminalProof must bind the ORIGINATING call, never the call in flight at
// handler completion).

// Adapted review fixture 1: an error consumer that breaks on the error event
// (exactly what the harness stream loop does) must still finalize the failure;
// a later successful call must not make the failed turn's arming ready.
func TestTerminalFailureFinalizedWhenConsumerRejectsError(t *testing.T) {
	ts := NewToolSet(nil, "")
	upstreamErr := errors.New("review overflow")
	calls := 0
	ts.terminalProof.origin.Store("stale-proof", int64(1))
	deps := installTerminalProofGate(harness.QueryDeps{
		CallModel: func(_ context.Context, _ llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
			return func(yield func(llm.StreamEvent, error) bool) {
				calls++
				if calls == 1 {
					ts.markTerminalProof("stale-proof")
					yield(llm.StreamEvent{}, upstreamErr)
					return
				}
				yield(llm.StreamEvent{Type: llm.SEMessageDelta, StopReason: "end_turn"}, nil)
			}
		},
	}, ts, nil)
	for _, err := range deps.CallModel(context.Background(), llm.CompletionRequest{}) {
		if err != nil {
			break // consumer rejects the error event, like the harness does
		}
	}
	for range deps.CallModel(context.Background(), llm.CompletionRequest{}) {
	}
	if got := ts.terminalProof.failedThrough.Load(); got != 1 {
		t.Fatalf("rejected transport error was not finalized before yield(false): failedThrough=%d", got)
	}
	if ts.terminalProofReady() {
		t.Fatal("failed proof response became terminal-ready after a later successful call; error finalization was skipped on yield(false)")
	}
}

// staleOriginProvider: call 1 launches the proof tool and fails with a
// recognized overflow while the handler is still blocked; call 2 (reactive
// recovery) releases the old handler and performs ordinary NON-proof tool work;
// call 3 is ordinary completion.
type staleOriginProvider struct {
	calls    int
	started  chan struct{}
	release  chan struct{}
	marked   chan struct{}
	overflow error
}

func (p *staleOriginProvider) Stream(ctx context.Context, _ llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
	return func(yield func(llm.StreamEvent, error) bool) {
		n := p.calls
		n++
		p.calls = n
		send := func(e llm.StreamEvent) bool { return yield(e, nil) }
		if n == 1 {
			for _, e := range []llm.StreamEvent{
				{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 10}},
				{Type: llm.SEToolUseStart, ToolID: "stale-old-proof", ToolName: "stale_delayed_proof"},
				{Type: llm.SEToolInputJSON, Text: "{}"},
				{Type: llm.SETextDelta, Text: "proof handler is running"},
			} {
				if !send(e) {
					return
				}
			}
			select {
			case <-p.started:
			case <-ctx.Done():
				return
			}
			yield(llm.StreamEvent{}, p.overflow) // handler still in flight
			return
		}
		if n == 2 {
			close(p.release) // failed call 1's handler completes DURING call 2
			select {
			case <-p.marked:
			case <-ctx.Done():
				return
			}
			for _, e := range []llm.StreamEvent{ // ordinary work, NOT a proof
				{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 10}},
				{Type: llm.SEToolUseStart, ToolID: "stale-new-noop", ToolName: "stale_noop"},
				{Type: llm.SEToolInputJSON, Text: "{}"},
				{Type: llm.SEMessageDelta, StopReason: "tool_use", Usage: llm.Usage{OutputTokens: 2}},
			} {
				if !send(e) {
					return
				}
			}
			return
		}
		send(llm.StreamEvent{Type: llm.SETextDelta, Text: "ordinary completion after recovery"})
		send(llm.StreamEvent{Type: llm.SEMessageDelta, StopReason: "end_turn"})
	}
}

func (p *staleOriginProvider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
	return streamViaComplete(p.Stream, ctx, req)
}

// staleReactiveCompactor isolates the SDK's documented Reactive-continue seam; it
// shrinks history and never fabricates proof success.
type staleReactiveCompactor struct{ overflow error }

func (c staleReactiveCompactor) Pre(_ context.Context, m []llm.Message, _ int) []llm.Message {
	return m
}
func (c staleReactiveCompactor) IsOverflow(err error) bool { return errors.Is(err, c.overflow) }
func (c staleReactiveCompactor) Reactive(_ context.Context, m []llm.Message) ([]llm.Message, bool) {
	if len(m) == 0 {
		return m, false
	}
	return append([]llm.Message(nil), m[1:]...), true
}

// Adapted review fixture 2 (THE demonstrated defect): a proof handler launched
// by FAILED call 1 finishes while successful non-proof call 2 runs; the gate must
// NOT credit the old proof to call 2 — call 3 must reach upstream.
func TestDelayedFailedTurnProofCannotBorrowLaterCallSequence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewToolSet(nil, "")
	p := &staleOriginProvider{
		started:  make(chan struct{}),
		release:  make(chan struct{}),
		marked:   make(chan struct{}),
		overflow: errors.New("review recognized overflow"),
	}
	proof := actool.Build(actool.Spec{Name: "stale_delayed_proof", Run: func(ctx context.Context, _ json.RawMessage, _ *actool.ToolContext) (actool.Result, error) {
		close(p.started)
		select {
		case <-p.release:
		case <-ctx.Done():
			return actool.Errorf("cancelled fixture"), nil
		}
		ts.GoalMet = true
		ts.Reason = "old failed response's proof"
		ts.markTerminalProof("stale-old-proof") // its ORIGINATING identity (call 1)
		close(p.marked)
		return actool.Text("old proof handler eventually succeeded"), nil
	}})
	noop := actool.Build(actool.Spec{Name: "stale_noop", Run: func(context.Context, json.RawMessage, *actool.ToolContext) (actool.Result, error) {
		return actool.Text("ordinary non-proof work completed"), nil
	}})
	opts := agentcore.Options{Provider: p, SystemPrompt: []string{"review"}, Tools: []actool.CoreTool{proof, noop}, PermissionMode: permission.ModeBypass, Compactor: staleReactiveCompactor{p.overflow}}
	observeProof := installTerminalProofGateOn(&opts, ts)
	var activities []db.Activity
	_, _, err := captureRun(ctx, opts, "controlled delayed proof recovery", func(a db.Activity) { activities = append(activities, a) }, observeProof)
	if err != nil {
		t.Fatalf("fixture failed before attribution assertion: %v", err)
	}
	if p.calls != 3 {
		t.Fatalf("upstream calls=%d, want 3: delayed proof from failed call 1 was attributed to successful non-proof call 2 and prematurely gated call 3", p.calls)
	}
	if ts.terminalProofReady() {
		t.Fatal("a proof from a failed response must never become terminal-ready via a later successful response")
	}
	for _, a := range activities {
		if a.Kind == "result" && strings.Contains(a.Detail, "old proof") {
			t.Fatal("old failed proof was presented as the final successful response")
		}
	}
}

// --- Converse-order regression (adapted from the independent
// review-terminal-order receipt, test SHA 9119c5bc…, real go exit 1 against the
// R16-accepted source; imports normalized original→CLI). Actual Session/SDK
// recovery/executor ordering with default concurrency: call 1 launches an old
// terminal handler and fails with recoverable overflow; call 2 performs a
// successful NEW terminal proof; only AFTER the new marker is installed does the
// old failed-response handler finish. Last-completer-wins used to erase the
// newer valid proof (armed_origin became 1) and leaked a third upstream call.

type supersededProofProvider struct {
	calls                                        int
	oldStarted, releaseOld, oldMarked, newMarked chan struct{}
	overflow                                     error
}

func (p *supersededProofProvider) Stream(ctx context.Context, _ llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
	return func(yield func(llm.StreamEvent, error) bool) {
		p.calls++
		call := p.calls
		send := func(e llm.StreamEvent) bool { return yield(e, nil) }
		wait := func(ch <-chan struct{}) bool {
			select {
			case <-ch:
				return true
			case <-ctx.Done():
				yield(llm.StreamEvent{}, ctx.Err())
				return false
			}
		}
		switch call {
		case 1:
			for _, event := range []llm.StreamEvent{
				{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 10}},
				{Type: llm.SEToolUseStart, ToolID: "review-old-proof", ToolName: "review_old_proof"},
				{Type: llm.SEToolInputJSON, Text: "{}"},
				{Type: llm.SETextDelta, Text: "old proof started"},
			} {
				if !send(event) {
					return
				}
			}
			if !wait(p.oldStarted) {
				return
			}
			yield(llm.StreamEvent{}, p.overflow)
		case 2:
			for _, event := range []llm.StreamEvent{
				{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 20}},
				{Type: llm.SEToolUseStart, ToolID: "review-new-proof", ToolName: "review_new_proof"},
				{Type: llm.SEToolInputJSON, Text: "{}"},
				{Type: llm.SETextDelta, Text: "new successful terminal proof"},
			} {
				if !send(event) {
					return
				}
			}
			if !wait(p.newMarked) {
				return
			}
			close(p.releaseOld)
			if !wait(p.oldMarked) {
				return
			}
			send(llm.StreamEvent{Type: llm.SEMessageDelta, StopReason: "tool_use", Usage: llm.Usage{OutputTokens: 7}})
		default:
			send(llm.StreamEvent{Type: llm.SETextDelta, Text: "unnecessary upstream request after valid proof"})
			send(llm.StreamEvent{Type: llm.SEMessageDelta, StopReason: "end_turn"})
		}
	}
}

func (p *supersededProofProvider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
	return streamViaComplete(p.Stream, ctx, req)
}

func TestOlderFailedProofCannotEraseNewSuccessfulProof(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewToolSet(nil, "")
	provider := &supersededProofProvider{
		oldStarted: make(chan struct{}), releaseOld: make(chan struct{}),
		oldMarked: make(chan struct{}), newMarked: make(chan struct{}),
		overflow: errors.New("review recoverable overflow"),
	}
	old := actool.Build(actool.Spec{Name: "review_old_proof", Run: func(ctx context.Context, _ json.RawMessage, tc *actool.ToolContext) (actool.Result, error) {
		close(provider.oldStarted)
		select {
		case <-provider.releaseOld:
		case <-ctx.Done():
			return actool.Errorf("fixture cancelled"), nil
		}
		ts.markTerminalProof(tc.ToolUseID)
		close(provider.oldMarked)
		return actool.Text("old failed-response proof eventually returned"), nil
	}})
	current := actool.Build(actool.Spec{Name: "review_new_proof", Run: func(_ context.Context, _ json.RawMessage, tc *actool.ToolContext) (actool.Result, error) {
		ts.markTerminalProof(tc.ToolUseID)
		close(provider.newMarked)
		return actool.Text("new authentic successful terminal proof"), nil
	}})
	opts := agentcore.Options{Provider: provider, SystemPrompt: []string{"controlled review"}, Tools: []actool.CoreTool{old, current}, PermissionMode: permission.ModeBypass, Compactor: staleReactiveCompactor{provider.overflow}}
	observeProof := installTerminalProofGateOn(&opts, ts)
	var activities []db.Activity
	_, _, err := captureRun(ctx, opts, "review terminal proof supersession", func(a db.Activity) { activities = append(activities, a) }, observeProof)
	if err != nil {
		t.Fatalf("fixture did not complete: %v", err)
	}
	successfulNew := 0
	for _, a := range activities {
		if a.Kind == "tool_result" && a.ToolUseID == "review-new-proof" && !a.IsError {
			successfulNew++
		}
	}
	if successfulNew != 1 {
		t.Fatalf("new proof did not finish authentically: %d results", successfulNew)
	}
	t.Logf("calls=%d armed_origin=%d succeeded=%d failed_through=%d new_successful_receipts=%d", provider.calls, ts.terminalProof.armedIn.Load(), ts.terminalProof.okCalls.Load(), ts.terminalProof.failedThrough.Load(), successfulNew)
	if provider.calls != 2 {
		t.Fatalf("upstream calls=%d, want 2: older failed proof erased newer successful proof and leaked call 3", provider.calls)
	}
	if !ts.terminalProofReady() {
		t.Fatal("newer fully successful proof lost terminal readiness")
	}
}
