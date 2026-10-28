package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/internal/dbtest"
	actool "github.com/Autumn-27/norma/tool"

	"github.com/Autumn-27/norma/agentcore"
	"github.com/Autumn-27/norma/harness"
	"github.com/Autumn-27/norma/llm"
	"github.com/Autumn-27/norma/permission"
	"github.com/Autumn-27/norma/transcript"
)

// Terminal goal proof: the pre-next-call gate (terminal_proof.go). The R11
// Flash/high receipt proved the completion gap (successful final prove_goal at
// 18:36:24, then another model call streamed until the 420 s pause); the R12
// immediate-cancel design was REJECTED by review because cancelling inside the
// proof tool races the still-arriving response, capOutput/PostToolUse and
// executor drain — losing assistant/tool-result/late-usage records or relabeling
// the successful tool "interrupted". These fixtures exercise that vulnerable
// boundary with context-aware scripted providers:
//
//   - proof stream held OPEN after the tool executed, usage arriving AFTER
//   - delayed PostToolUse hook
//   - parent cancellation racing after the proof (keeps abort semantics)
//   - a real provider/stream error after the armed proof (never becomes success)
//   - partial/rejected proof (no gate)
//   - streaming AND non-streaming entry points (Deps.CallModel/CallModelSync)
//   - the real prove_goal storage path (DB-gated) and the real goal_met tool
//
// The RED receipt reintroduces the R12-style in-tool cancel as a mutation and
// shows these fixtures catch exactly the reviewed races.

// proofScript drives scriptedProvider. Turn 1 emits the proof tool call, then a
// text delta (which finalizes the tool block and starts execution while the
// response is still open), then waits for the tool to run, then applies
// afterExecuted (default: late usage + tool_use stop). While waiting, the
// provider selects on ctx.Done — a context-aware stream cancelled mid-response
// ends without the late events.
type proofScript struct {
	toolName string
	executed chan struct{}
	// input overrides the turn-1 tool arguments (default "{}").
	input string
	// afterExecuted, when set, replaces the default late-usage completion and may
	// yield further events or return early (context-aware streams end without the
	// late events once ctx is done).
	afterExecuted func(yield func(llm.StreamEvent, error) bool, ctx context.Context)
}

type scriptedProvider struct {
	mu     sync.Mutex
	calls  int
	script *proofScript
}

func (p *scriptedProvider) callsCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

func (p *scriptedProvider) Stream(ctx context.Context, _ llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
	return func(yield func(llm.StreamEvent, error) bool) { p.stream(ctx, yield) }
}

func (p *scriptedProvider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
	return streamViaComplete(p.Stream, ctx, req)
}

func (p *scriptedProvider) stream(ctx context.Context, yield func(llm.StreamEvent, error) bool) {
	p.mu.Lock()
	p.calls++
	call := p.calls
	p.mu.Unlock()
	if call == 1 {
		input := p.script.input
		if input == "" {
			input = "{}"
		}
		for _, ev := range []llm.StreamEvent{
			{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 10}},
			{Type: llm.SEToolUseStart, ToolID: "call-1", ToolName: p.script.toolName},
			{Type: llm.SEToolInputJSON, Text: input},
			{Type: llm.SETextDelta, Text: "verifying the objective"},
		} {
			if !yield(ev, nil) {
				return
			}
		}
		if p.script.afterExecuted != nil {
			p.script.afterExecuted(yield, ctx)
			return
		}
		p.hold(ctx)
		if ctx.Err() != nil {
			return // context-aware: no late events once cancelled mid-response
		}
		yield(llm.StreamEvent{Type: llm.SEMessageDelta, StopReason: "tool_use", Usage: llm.Usage{OutputTokens: 7}}, nil)
		return
	}
	for _, ev := range []llm.StreamEvent{
		{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 20}},
		{Type: llm.SETextDelta, Text: "extra inference after terminal proof"},
		{Type: llm.SEMessageDelta, StopReason: "end_turn", Usage: llm.Usage{OutputTokens: 4}},
	} {
		if !yield(ev, nil) {
			return
		}
	}
}

func (p *scriptedProvider) hold(ctx context.Context) {
	select {
	case <-p.script.executed:
	case <-ctx.Done():
	case <-time.After(5 * time.Second):
	}
}

// streamViaComplete adapts a streaming provider for the non-streaming entry
// point (mirrors captureUsageProvider.Complete).
func streamViaComplete(stream func(ctx context.Context, req llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error], ctx context.Context, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
	acc := llm.NewAccumulator()
	for ev, err := range stream(ctx, req) {
		if err != nil {
			return llm.Message{}, "", llm.Usage{}, err
		}
		acc.Add(ev)
	}
	return acc.Message(), acc.StopReason, acc.Usage, nil
}

// delayedPostToolUse models the reviewed PostToolUse race: the hook is still
// busy after the tool handler returned (capOutput/PostToolUse run before the
// executor marks the tool completed).
type delayedPostToolUse struct{ delay time.Duration }

func (h delayedPostToolUse) PreToolUse(context.Context, string, []byte) (bool, string, []byte) {
	return false, "", nil
}
func (h delayedPostToolUse) PostToolUse(context.Context, string, []byte, []byte, bool) {
	time.Sleep(h.delay)
}
func (h delayedPostToolUse) Stop(context.Context, []llm.Message) (bool, []string, string) {
	return false, nil, ""
}

// runProofSession drives captureRun with the scripted provider and a probe tool
// that mirrors prove_goal's terminal branch (GoalMet + Reason + gate armed),
// installing the gate exactly like planner.Plan. r12Cancel, when non-nil,
// reintroduces the REJECTED R12 in-tool cancel (RED-receipt mutation): the probe
// cancels the session context from inside its handler.
func runProofSession(t *testing.T, ts *ToolSet, script *proofScript, hooks harness.HookRunner, nonStreaming bool, ctx context.Context, r12Cancel context.CancelCauseFunc) (int, []db.Activity, error) {
	t.Helper()
	provider := &scriptedProvider{script: script}
	probe := actool.Build(actool.Spec{
		Name: script.toolName,
		Run: func(_ context.Context, _ json.RawMessage, tc *actool.ToolContext) (actool.Result, error) {
			ts.GoalMet = true
			ts.Reason = "all goals met (probe)"
			ts.markTerminalProof(tc.ToolUseID)
			if r12Cancel != nil { // mutation: the rejected R12 behavior
				r12Cancel(context.Canceled)
			}
			close(script.executed)
			return actool.Text("goal proven; task completes automatically"), nil
		},
	})
	opts := agentcore.Options{
		Provider:       provider,
		SystemPrompt:   []string{"probe"},
		Tools:          []actool.CoreTool{probe},
		PermissionMode: permission.ModeBypass,
		Hooks:          hooks,
		NonStreaming:   nonStreaming,
	}
	observeProof := installTerminalProofGateOn(&opts, ts) // what planner.Plan installs
	var activities []db.Activity
	_, _, err := captureRun(ctx, opts, "prove the goal", func(r db.Activity) { activities = append(activities, r) }, observeProof)
	return provider.callsCount(), activities, err
}

func activityByKind(acts []db.Activity, kind string) *db.Activity {
	for i := range acts {
		if acts[i].Kind == kind {
			return &acts[i]
		}
	}
	return nil
}

func toolResultOf(acts []db.Activity) *db.Activity {
	for i := range acts {
		if acts[i].Kind == "tool_result" {
			return &acts[i]
		}
	}
	return nil
}

// F1 (streaming): the proof response stays open while the tool executes; usage
// arrives AFTER execution. The gate lets the turn complete fully — successful
// tool result, late usage recorded, native non-error completion — and answers
// the NEXT model call locally: exactly one upstream request.
func TestProofStreamHeldOpenUsageAfterTool_GateCompletesWithoutNextCall(t *testing.T) {
	ts := NewToolSet(nil, "")
	calls, acts, err := runProofSession(t, ts, &proofScript{toolName: "terminal_proof_probe", executed: make(chan struct{})}, nil, false, context.Background(), nil)
	if err != nil {
		t.Fatalf("terminal proof run returned an error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("upstream model calls = %d; the gate must answer the next call locally after a fully recorded proof", calls)
	}
	tr := toolResultOf(acts)
	if tr == nil || tr.IsError || !strings.Contains(tr.Detail, "goal proven") {
		t.Fatalf("successful proof tool result missing/changed: %+v", tr)
	}
	if result := activityByKind(acts, "result"); result == nil || result.IsError {
		t.Fatalf("completion recorded as error: %+v", result)
	}
	if result := activityByKind(acts, "result"); result.OutputTokens == nil || *result.OutputTokens != 7 {
		t.Fatalf("late-arriving usage lost from the result row: %+v", result)
	}
	if !ts.GoalMet {
		t.Fatal("GoalMet not set")
	}
}

// F2 (streaming, delayed PostToolUse): the successful proof tool result must
// survive a busy PostToolUse hook; nothing may interrupt or relabel it.
func TestDelayedPostToolUseDoesNotInterruptProofResult(t *testing.T) {
	ts := NewToolSet(nil, "")
	calls, acts, err := runProofSession(t, ts,
		&proofScript{toolName: "terminal_proof_probe", executed: make(chan struct{})},
		delayedPostToolUse{delay: 75 * time.Millisecond}, false, context.Background(), nil)
	if err != nil {
		t.Fatalf("run returned an error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("upstream model calls = %d, want 1 (gate)", calls)
	}
	tr := toolResultOf(acts)
	if tr == nil || tr.IsError || !strings.Contains(tr.Detail, "goal proven") {
		t.Fatalf("delayed PostToolUse corrupted the successful proof result: %+v", tr)
	}
	if result := activityByKind(acts, "result"); result == nil || result.IsError {
		t.Fatalf("completion recorded as error: %+v", result)
	}
}

// A clean model response is insufficient when the real terminal tool overruns
// MaxDuration inside PostToolUse: its interrupted receipt must allow settlement.
func TestInterruptedTerminalProofDoesNotGateSettlement(t *testing.T) {
	ts := NewToolSet(nil, "")
	provider := &scriptedProvider{script: &proofScript{
		toolName: "goal_met", input: `{"reason":"synthetic objectives achieved"}`, executed: make(chan struct{}),
		afterExecuted: func(yield func(llm.StreamEvent, error) bool, _ context.Context) {
			yield(llm.StreamEvent{Type: llm.SEMessageDelta, StopReason: "tool_use", Usage: llm.Usage{OutputTokens: 7}}, nil)
		},
	}}
	cmp := &countingCompactor{}
	opts := agentcore.Options{Provider: provider, SystemPrompt: []string{"probe"}, Tools: []actool.CoreTool{ts.goalMet()}, PermissionMode: permission.ModeBypass,
		MaxDuration: 30 * time.Millisecond, Hooks: delayedPostToolUse{delay: 120 * time.Millisecond}, Compactor: cmp,
		Settlement: &harness.Settlement{Prompt: "settle interrupted tool honestly", MaxTurns: 1}}
	observeProof := installTerminalProofGateOn(&opts, ts)
	var acts []db.Activity
	_, reason, err := captureRun(context.Background(), opts, "prove", func(a db.Activity) { acts = append(acts, a) }, observeProof)
	// Retain the original review probe's 30/120 ms timing and late-hook check.
	time.Sleep(150 * time.Millisecond)
	tr := toolResultOf(acts)
	met, detail := ts.goalResult()
	t.Logf("calls=%d reason=%s err=%v ready=%v tool_result=%+v", provider.callsCount(), reason, err, ts.terminalProofReady(), tr)
	if !met || detail != "synthetic objectives achieved" {
		t.Fatalf("real terminal handler did not mark its proof: met=%v reason=%q", met, detail)
	}
	if tr == nil || !tr.IsError || !strings.Contains(tr.Detail, "tool execution interrupted") {
		t.Fatalf("fixture did not interrupt terminal tool: %+v", tr)
	}
	if err != nil || reason != harness.ReasonTimeout {
		t.Fatalf("native timeout semantics changed: reason=%s err=%v", reason, err)
	}
	if provider.callsCount() != 2 {
		t.Fatalf("interrupted proof gated real settlement request: calls=%d want 2", provider.callsCount())
	}
	if cmp.preCalls != 2 || ts.terminalProofReady() {
		t.Fatalf("interrupted proof skipped settlement compaction or became ready: Pre=%d ready=%v", cmp.preCalls, ts.terminalProofReady())
	}
}

type blockedTerminalPostHook struct {
	entered, release, exited chan struct{}
}

func (*blockedTerminalPostHook) PreToolUse(context.Context, string, []byte) (bool, string, []byte) {
	return false, "", nil
}
func (h *blockedTerminalPostHook) PostToolUse(context.Context, string, []byte, []byte, bool) {
	close(h.entered)
	<-h.release
	close(h.exited)
}
func (*blockedTerminalPostHook) Stop(context.Context, []llm.Message) (bool, []string, string) {
	return false, nil, ""
}

// The provider waits for the real handler and hook before ending its response;
// the hook stays blocked until capture observes the interrupted SDK receipt.
// A slow host cannot pass by timing out before terminal tool execution starts.
func TestInterruptedTerminalReceiptAllowsSettlementAfterLateHook(t *testing.T) {
	ts := NewToolSet(nil, "")
	hook := &blockedTerminalPostHook{entered: make(chan struct{}), release: make(chan struct{}), exited: make(chan struct{})}
	var once sync.Once
	release := func() { once.Do(func() { close(hook.release) }) }
	defer func() {
		release()
		select {
		case <-hook.exited:
		case <-time.After(time.Second):
			t.Error("held terminal hook did not exit")
		}
	}()
	provider := &scriptedProvider{script: &proofScript{
		toolName: "goal_met", input: `{"reason":"synchronized terminal proof"}`, executed: make(chan struct{}),
		afterExecuted: func(yield func(llm.StreamEvent, error) bool, ctx context.Context) {
			select {
			case <-hook.entered:
				yield(llm.StreamEvent{Type: llm.SEMessageDelta, StopReason: "tool_use", Usage: llm.Usage{OutputTokens: 7}}, nil)
			case <-ctx.Done():
			}
		},
	}}
	cmp := &countingCompactor{}
	opts := agentcore.Options{Provider: provider, SystemPrompt: []string{"probe"}, Tools: []actool.CoreTool{ts.goalMet()}, PermissionMode: permission.ModeBypass,
		MaxDuration: 30 * time.Millisecond, Hooks: hook, Compactor: cmp,
		Settlement: &harness.Settlement{Prompt: "settle interrupted tool honestly", MaxTurns: 1}}
	observeProof := installTerminalProofGateOn(&opts, ts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var acts []db.Activity
	observed := false
	_, reason, err := captureRun(ctx, opts, "prove", func(a db.Activity) {
		acts = append(acts, a)
		if a.Kind == "tool_result" && a.Tool == "goal_met" {
			select {
			case <-hook.entered:
			default:
				t.Error("interruption occurred before the real post-tool hook entered")
			}
			met, detail := ts.goalResult()
			if !met || detail != "synchronized terminal proof" || !a.IsError {
				t.Errorf("real terminal execution/interruption missing: met=%v reason=%q receipt=%+v", met, detail, a)
			}
			observed = true
			release()
		}
	}, observeProof)
	if !observed || toolResultOf(acts) == nil {
		t.Fatal("no authentic terminal-tool interruption receipt")
	}
	if err != nil || reason != harness.ReasonTimeout || provider.callsCount() != 2 || cmp.preCalls != 2 {
		t.Fatalf("settlement suppressed or native reason changed: calls=%d Pre=%d reason=%s err=%v", provider.callsCount(), cmp.preCalls, reason, err)
	}
	select {
	case <-hook.exited:
	case <-time.After(time.Second):
		t.Fatal("released terminal hook did not finish")
	}
	if ts.terminalProofReady() {
		t.Fatal("late hook completion qualified an already interrupted receipt")
	}
}

// F3a: parent cancellation wins over an armed gate — with the context already
// cancelled the loop aborts before any model call; the gate itself also refuses
// to act on a cancelled context and delegates to the provider.
func TestParentCancelWinsOverArmedGate(t *testing.T) {
	ts := NewToolSet(nil, "")
	priorCalls, priorActs, priorErr := runProofSession(t, ts, &proofScript{toolName: "prior_terminal_proof", executed: make(chan struct{})}, nil, false, context.Background(), nil)
	priorResult := toolResultOf(priorActs)
	if priorErr != nil || priorCalls != 1 || priorResult == nil || priorResult.IsError || !ts.terminalProofReady() {
		t.Fatalf("prior proof did not reach an authentic successful boundary: calls=%d result=%+v ready=%v err=%v", priorCalls, priorResult, ts.terminalProofReady(), priorErr)
	}
	provider := &scriptedProvider{script: &proofScript{toolName: "terminal_proof_probe", executed: make(chan struct{})}}
	opts := agentcore.Options{Provider: provider, SystemPrompt: []string{"probe"}, PermissionMode: permission.ModeBypass}
	observeProof := installTerminalProofGateOn(&opts, ts)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var acts []db.Activity
	_, _, err := captureRun(ctx, opts, "prove", func(r db.Activity) { acts = append(acts, r) }, observeProof)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("parent cancellation must surface context.Canceled, got %v", err)
	}
	if provider.callsCount() != 0 {
		t.Fatalf("upstream calls = %d; a cancelled run must not reach the provider", provider.callsCount())
	}
	if result := activityByKind(acts, "result"); result == nil || !result.IsError {
		t.Fatalf("cancelled run must record an error result: %+v", result)
	}
}

// F3b: parent cancellation landing while the proof response is still open keeps
// honest abort semantics — records made before the boundary stay, the run
// finishes cancelled, and nothing masks it as success.
func TestParentCancelDuringOpenProofStreamKeepsAbortSemantics(t *testing.T) {
	ts := NewToolSet(nil, "")
	var parentCancel context.CancelFunc
	executed := make(chan struct{})
	script := &proofScript{
		toolName: "terminal_proof_probe",
		executed: executed,
		afterExecuted: func(_ func(llm.StreamEvent, error) bool, ctx context.Context) {
			<-executed // tool ran, gate armed, response still open
			parentCancel()
			<-ctx.Done()
			// context-aware stream: ends without the late usage
		},
	}
	provider := &scriptedProvider{script: script}
	probe := actool.Build(actool.Spec{
		Name: script.toolName,
		Run: func(_ context.Context, _ json.RawMessage, tc *actool.ToolContext) (actool.Result, error) {
			ts.GoalMet = true
			ts.Reason = "all goals met (probe)"
			ts.markTerminalProof(tc.ToolUseID)
			close(executed)
			return actool.Text("goal proven; task completes automatically"), nil
		},
	})
	opts := agentcore.Options{Provider: provider, SystemPrompt: []string{"probe"}, Tools: []actool.CoreTool{probe}, PermissionMode: permission.ModeBypass}
	observeProof := installTerminalProofGateOn(&opts, ts)
	ctx, cancel := context.WithCancel(context.Background())
	parentCancel = cancel
	defer cancel()
	var acts []db.Activity
	_, _, err := captureRun(ctx, opts, "prove", func(r db.Activity) { acts = append(acts, r) }, observeProof)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("parent cancellation during the proof response must surface context.Canceled, got %v", err)
	}
	if tr := toolResultOf(acts); tr == nil || tr.IsError {
		t.Fatalf("records made before cancellation must be preserved: %+v", tr)
	}
	if result := activityByKind(acts, "result"); result == nil || !result.IsError {
		t.Fatalf("cancelled run must record an error result (no success masking): %+v", result)
	}
}

// F4: a REAL provider/stream error after the armed proof (same open response)
// finishes the run with that error — a recorded goal state alone never becomes
// success, because the gate is only consulted for a next call that never happens.
func TestRealStreamErrorAfterProofIsNotSuccess(t *testing.T) {
	ts := NewToolSet(nil, "")
	executed := make(chan struct{})
	upstreamErr := errors.New("upstream 500 after proof")
	script := &proofScript{
		toolName: "terminal_proof_probe",
		executed: executed,
		afterExecuted: func(yield func(llm.StreamEvent, error) bool, ctx context.Context) {
			<-executed // tool ran, gate armed
			yield(llm.StreamEvent{}, upstreamErr)
		},
	}
	provider := &scriptedProvider{script: script}
	probe := actool.Build(actool.Spec{
		Name: script.toolName,
		Run: func(_ context.Context, _ json.RawMessage, tc *actool.ToolContext) (actool.Result, error) {
			ts.GoalMet = true
			ts.Reason = "all goals met (probe)"
			ts.markTerminalProof(tc.ToolUseID)
			close(executed)
			return actool.Text("goal proven; task completes automatically"), nil
		},
	})
	opts := agentcore.Options{Provider: provider, SystemPrompt: []string{"probe"}, Tools: []actool.CoreTool{probe}, PermissionMode: permission.ModeBypass}
	observeProof := installTerminalProofGateOn(&opts, ts)
	var acts []db.Activity
	_, _, err := captureRun(context.Background(), opts, "prove", func(r db.Activity) { acts = append(acts, r) }, observeProof)
	if !errors.Is(err, upstreamErr) {
		t.Fatalf("real provider error after proof must surface verbatim, got %v", err)
	}
	if provider.callsCount() != 1 {
		t.Fatalf("upstream calls = %d, want 1 (the error ended the run before any next call)", provider.callsCount())
	}
	if result := activityByKind(acts, "result"); result == nil || !result.IsError {
		t.Fatalf("provider error must record an error result (never success): %+v", result)
	}
}

// F5 (negative): a rejected proof never arms the gate — the session continues
// with further inference and completes normally.
func TestRejectedProofDoesNotArmGate(t *testing.T) {
	ts := NewToolSet(nil, "")
	executed := make(chan struct{})
	provider := &scriptedProvider{script: &proofScript{toolName: "terminal_proof_probe", executed: executed}}
	probe := actool.Build(actool.Spec{
		Name: "terminal_proof_probe",
		Run: func(context.Context, json.RawMessage, *actool.ToolContext) (actool.Result, error) {
			close(executed)
			return actool.Errorf("proof rejected: evidence insufficient"), nil
		},
	})
	opts := agentcore.Options{Provider: provider, SystemPrompt: []string{"probe"}, Tools: []actool.CoreTool{probe}, PermissionMode: permission.ModeBypass}
	observeProof := installTerminalProofGateOn(&opts, ts)
	var acts []db.Activity
	_, _, err := captureRun(context.Background(), opts, "attempt the proof", func(r db.Activity) { acts = append(acts, r) }, observeProof)
	if err != nil {
		t.Fatal(err)
	}
	if provider.callsCount() < 2 {
		t.Fatalf("a rejected proof must let the session continue (calls=%d)", provider.callsCount())
	}
	if ts.GoalMet {
		t.Fatal("rejected proof must not set GoalMet")
	}
	if result := activityByKind(acts, "result"); result == nil || result.IsError {
		t.Fatalf("normal continuation produced an error result: %+v", result)
	}
}

// F6 (non-streaming): the CallModelSync gate behaves identically — one upstream
// request, full records, native completion. (The hold-open race does not exist
// here: the complete response is assembled before any tool runs, so the script
// finishes the turn directly.)
func TestNonStreamingEntryPointGatesNextCall(t *testing.T) {
	ts := NewToolSet(nil, "")
	script := &proofScript{
		toolName: "terminal_proof_probe",
		executed: make(chan struct{}),
		afterExecuted: func(yield func(llm.StreamEvent, error) bool, ctx context.Context) {
			yield(llm.StreamEvent{Type: llm.SEMessageDelta, StopReason: "tool_use", Usage: llm.Usage{OutputTokens: 7}}, nil)
		},
	}
	calls, acts, err := runProofSession(t, ts, script, nil, true, context.Background(), nil)
	if err != nil {
		t.Fatalf("non-streaming terminal proof run returned an error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("upstream model calls = %d, want 1 (CallModelSync gate)", calls)
	}
	if tr := toolResultOf(acts); tr == nil || tr.IsError || !strings.Contains(tr.Detail, "goal proven") {
		t.Fatalf("successful proof tool result missing: %+v", tr)
	}
	if result := activityByKind(acts, "result"); result == nil || result.IsError {
		t.Fatalf("completion recorded as error: %+v", result)
	}
}

// F8: the real goal_met tool (no store needed) keeps its opt-in meaning — it
// still sets the engine-facing GoalMet/Reason AND arms the gate, so the session
// ends after one upstream call without further inference.
func TestGoalMetToolMeaningSurvivesGate(t *testing.T) {
	ts := NewToolSet(nil, "")
	var goalMetTool actool.CoreTool
	for _, candidate := range ts.PlannerTools() {
		if candidate.Name() == "goal_met" {
			goalMetTool = candidate
			break
		}
	}
	if goalMetTool == nil {
		t.Fatal("goal_met missing from planner tools")
	}
	provider := &scriptedProvider{script: &proofScript{
		toolName: "goal_met",
		executed: make(chan struct{}),
		input:    `{"reason":"objectives achieved"}`,
		afterExecuted: func(yield func(llm.StreamEvent, error) bool, ctx context.Context) {
			// goal_met is the real tool (not the probe); nothing signals executed,
			// so finish the turn directly — the real race fixture is F1/F2.
			yield(llm.StreamEvent{Type: llm.SEMessageDelta, StopReason: "tool_use", Usage: llm.Usage{OutputTokens: 7}}, nil)
		},
	}}
	opts := agentcore.Options{Provider: provider, SystemPrompt: []string{"probe"}, Tools: []actool.CoreTool{goalMetTool}, PermissionMode: permission.ModeBypass}
	observeProof := installTerminalProofGateOn(&opts, ts)
	var acts []db.Activity
	_, _, err := captureRun(context.Background(), opts, "close the engagement", func(r db.Activity) { acts = append(acts, r) }, observeProof)
	if err != nil {
		t.Fatal(err)
	}
	if provider.callsCount() != 1 {
		t.Fatalf("upstream calls = %d, want 1 (goal_met arms the gate)", provider.callsCount())
	}
	if !ts.GoalMet || ts.Reason != "objectives achieved" {
		t.Fatalf("goal_met meaning changed: GoalMet=%v Reason=%q", ts.GoalMet, ts.Reason)
	}
	if result := activityByKind(acts, "result"); result == nil || result.IsError {
		t.Fatalf("completion recorded as error: %+v", result)
	}
}

// F7 (DB-gated): the REAL prove_goal handler in a real session with a
// transcript: both goals proven in one turn (the second arms the gate), exactly
// one upstream call, both successful tool results recorded, goals durably met,
// and tool-use/result pairing persisted in the transcript.
func TestRealProveGoalSessionGatesAfterFinalProof(t *testing.T) {
	dsn := dbtest.DSN(t)
	d, err := db.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	expID, err := d.CreateExploration("terminal proof gate probe", "verify the real prove_goal boundary")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = d.Exec(`DELETE FROM explorations WHERE id=$1`, expID) })
	ts := d.Exploration(expID)
	tools := NewToolSet(ts, "planner")
	var proveGoal actool.CoreTool
	for _, candidate := range tools.PlannerTools() {
		if candidate.Name() == "prove_goal" {
			proveGoal = candidate
			break
		}
	}
	if proveGoal == nil {
		t.Fatal("prove_goal missing from planner tools")
	}
	evidence, err := ts.AddNode(db.KindFact, map[string]any{
		"summary": "controlled observation for both synthetic objectives",
		"detail":  "owned fixture evidence; no external target or inference",
	}, 0, "confirmed", "fixture", nil)
	if err != nil {
		t.Fatal(err)
	}
	goalOne, err := ts.AddGoal(map[string]any{"text": "objective one"}, "test")
	if err != nil {
		t.Fatal(err)
	}
	goalTwo, err := ts.AddGoal(map[string]any{"text": "objective two"}, "test")
	if err != nil {
		t.Fatal(err)
	}
	store := transcript.NewStore(t.TempDir())
	const sessionID = "terminal-proof-real-provegoal"
	provider := &twoGoalProvider{goalOne: goalOne, goalTwo: goalTwo, evidence: evidence}
	opts := agentcore.Options{
		Provider:       provider,
		SystemPrompt:   []string{"probe"},
		Tools:          []actool.CoreTool{proveGoal},
		PermissionMode: permission.ModeBypass,
		Transcript:     store,
		SessionID:      sessionID,
	}
	observeProof := installTerminalProofGateOn(&opts, tools)
	var acts []db.Activity
	_, _, err = captureRun(context.Background(), opts, "prove both objectives", func(r db.Activity) { acts = append(acts, r) }, observeProof)
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 1 {
		t.Fatalf("upstream calls = %d, want 1 (final proof armed the gate)", provider.calls)
	}
	successes := 0
	for i := range acts {
		if acts[i].Kind == "tool_result" && !acts[i].IsError {
			successes++
		}
	}
	if successes != 2 {
		t.Fatalf("successful prove_goal results = %d, want 2", successes)
	}
	if !tools.GoalMet {
		t.Fatal("final proof did not set GoalMet")
	}
	for _, id := range []int64{goalOne, goalTwo} {
		node, err := ts.GetNode(id)
		if err != nil || node == nil || node.State != "met" {
			t.Fatalf("goal %d not durably met: %+v err=%v", id, node, err)
		}
	}
	if result := activityByKind(acts, "result"); result == nil || result.IsError {
		t.Fatalf("completion recorded as error: %+v", result)
	}
	body, err := os.ReadFile(store.MainPath(sessionID))
	if err != nil {
		t.Fatalf("transcript unreadable: %v", err)
	}
	text := string(body)
	for _, marker := range []string{"prove-goal-1", "prove-goal-2", "task completes automatically"} {
		if !strings.Contains(text, marker) {
			t.Fatalf("transcript lost tool-use/result pairing marker %q", marker)
		}
	}
}

// twoGoalProvider streams one turn proving both goals, then plain text turns.
type twoGoalProvider struct {
	calls    int
	goalOne  int64
	goalTwo  int64
	evidence int64
}

func (p *twoGoalProvider) Stream(ctx context.Context, _ llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
	return func(yield func(llm.StreamEvent, error) bool) {
		p.calls++
		if p.calls == 1 {
			events := []llm.StreamEvent{
				{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 12}},
				{Type: llm.SEToolUseStart, ToolID: "prove-goal-1", ToolName: "prove_goal"},
				{Type: llm.SEToolInputJSON, Text: fmt.Sprintf(`{"goal_id":%d,"evidence_id":%d,"reason":"covers objective one"}`, p.goalOne, p.evidence)},
				{Type: llm.SEToolUseStart, ToolID: "prove-goal-2", ToolName: "prove_goal"},
				{Type: llm.SEToolInputJSON, Text: fmt.Sprintf(`{"goal_id":%d,"evidence_id":%d,"reason":"covers objective two"}`, p.goalTwo, p.evidence)},
				{Type: llm.SEMessageDelta, StopReason: "tool_use", Usage: llm.Usage{OutputTokens: 6}},
			}
			for _, ev := range events {
				if !yield(ev, nil) {
					return
				}
			}
			return
		}
		for _, ev := range []llm.StreamEvent{
			{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 20}},
			{Type: llm.SETextDelta, Text: "extra inference"},
			{Type: llm.SEMessageDelta, StopReason: "end_turn", Usage: llm.Usage{OutputTokens: 2}},
		} {
			if !yield(ev, nil) {
				return
			}
		}
	}
}

func (p *twoGoalProvider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
	return streamViaComplete(p.Stream, ctx, req)
}
