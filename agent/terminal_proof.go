package agent

import (
	"context"
	"iter"
	"sync"
	"sync/atomic"

	"github.com/Autumn-27/norma/agentcore"
	"github.com/Autumn-27/norma/compaction"
	"github.com/Autumn-27/norma/harness"
	"github.com/Autumn-27/norma/llm"
)

// Terminal goal proof: stop the planner session BEFORE any further upstream
// inference — but ONLY at a fully successful response boundary.
//
// Boundary contract (each clause pinned by a fixture in terminal_proof_test.go):
//
//   - The harness loop records a completed turn (assistant message, per-turn
//     usage event, tool-result messages) at the END of its iteration, then starts
//     the next model call. Gate + compaction wrappers are consulted only from
//     that point on, so a gated run never cuts a live response, tool,
//     capOutput/PostToolUse or executor drain (the rejected R12 cancel races).
//   - Compaction: Compactor.Pre runs BEFORE QueryDeps.CallModel (norma query.go
//     407-410) and the default/noa summarizers call the Provider directly, so the
//     model-call gate alone would not prevent post-proof compaction inference.
//     The selected Compactor is therefore wrapped so Pre is skipped exactly when
//     the gate would fire; ordinary pre-proof compaction and reactive overflow
//     recovery are forwarded untouched.
//   - Recoverable overflow: a response whose proof tool ran can still fail
//     afterwards (prompt-too-long). The SDK's reactive path retries WITHOUT
//     recording the failed response's assistant/tool-result messages, so an
//     arming from that response must NOT later produce a local success turn.
//     Readiness is therefore qualified by model-call sequence: the arming's
//     response must have completed cleanly (okCalls), its marked terminal tool
//     must have an authentic successful result (receiptedIn), and any later
//     model/tool-result error invalidates that call and earlier (failedThrough). Each tool
//     keeps its originating call sequence through its tool-use ID, so a handler
//     completing after a stream error cannot borrow a later successful response
//     to qualify the failed proof turn.
//   - Parent cancellation wins (loop checks ctx before the gate; the gate also
//     refuses a cancelled context). Real provider/stream errors keep their native
//     semantics: the gate is consulted only for a NEXT call, which a failed run
//     never issues. The gated turn is content-free with zero usage and flows
//     through the native no-tool completion path (Stop hooks, ReasonCompleted).
//   - Existing dependency wrappers (e.g. sidequestion capture) are composed, not
//     overwritten; the concrete Provider and its optional interfaces are untouched.

// terminalProof tracks the terminal-proof state shared by tool handlers (writer
// side) and the gate/compaction wrappers (reader side). All scalar fields are
// atomic; origin is a sync.Map written by the deps wrappers before a tool_use
// starts executing and read by the handler when it finishes.
type terminalProof struct {
	armed         atomic.Bool  // a terminal branch ran (prove_goal all-met / goal_met)
	current       atomic.Int64 // model-call sequence generator (also the in-flight seq)
	armedIn       atomic.Int64 // ORIGINATING call of the arming tool_use (0 = unattributed)
	okCalls       atomic.Int64 // highest sequence that completed without error
	failedThrough atomic.Int64 // highest sequence with a model or authentic tool-result error
	receiptedIn   atomic.Int64 // highest originating terminal call with a successful emitted result
	markedUses    sync.Map     // terminal tool-use ID -> originating model-call sequence
	origin        sync.Map     // tool_use ID -> originating model-call sequence
}

// markTerminalProof arms the pre-next-call gate from a tool handler, bound to
// the ORIGINATING model call of its tool_use block (toolUseID from the SDK
// ToolContext). Binding the origin — never the call in flight when the handler
// happens to finish — is what makes a proof launched by a FAILED response
// unattributable to a later successful one: the reviewer-verified race is a
// call-1 handler still running while call 2 (e.g. reactive recovery) streams;
// stamping "current" there would borrow call 2's identity and suppress call 3
// without the proof turn ever being recorded. An unknown ID (direct invocation
// outside a session) leaves the arming unattributed and never gates inference.
// Call only after the terminal condition is durably recorded (same point as
// ToolSet.GoalMet).
func (t *ToolSet) markTerminalProof(toolUseID string) {
	seq, ok := t.terminalProof.origin.Load(toolUseID)
	origin, _ := seq.(int64)
	if !ok || origin == 0 {
		// Unattributed arming (direct invocation / unknown identity): never
		// terminal by itself, and must not erase an already-recorded valid proof.
		t.terminalProof.armed.Store(true)
		return
	}
	t.terminalProof.markedUses.Store(toolUseID, origin)
	// Retain the NEWEST originating proof monotonically. Last-completer-wins
	// would let a late handler from an OLDER (possibly failed) response erase a
	// newer valid proof and resurrect unnecessary post-proof inference — the
	// independently reproduced converse-order defect (review-terminal-order).
	for {
		current := t.terminalProof.armedIn.Load()
		if origin <= current && current != 0 {
			break // an equal or newer originating proof is already retained
		}
		if t.terminalProof.armedIn.CompareAndSwap(current, origin) {
			break
		}
	}
	t.terminalProof.armed.Store(true)
}

// terminalProofReady reports whether a terminal proof belongs to a model
// response with an authentic successful terminal tool result, making the next upstream inference
// (main request or compaction summary) unnecessary.
func (t *ToolSet) terminalProofReady() bool {
	p := &t.terminalProof
	if !p.armed.Load() {
		return false
	}
	in := p.armedIn.Load()
	return in > 0 && in <= p.okCalls.Load() && in > p.failedThrough.Load() && in == p.receiptedIn.Load()
}

// observeTerminalProofEvent receives the executor's authentic result, after it
// decides completed versus interrupted. A handler or PostToolUse return alone
// cannot qualify: timeout may still win before the executor emits its result.
func (t *ToolSet) observeTerminalProofEvent(ev harness.Event) {
	if t == nil || ev.Kind != harness.KindToolResult || ev.ToolResult == nil {
		return
	}
	p := &t.terminalProof
	seq, ok := p.origin.Load(ev.ToolResult.ToolUseID)
	origin, _ := seq.(int64)
	if !ok || origin == 0 {
		return
	}
	if ev.ToolResult.IsError {
		// Also reject an interrupted sibling tool in the same response: its
		// timeout settlement must not be swallowed by an earlier proof result.
		t.terminalProofCallFailed(origin)
		return
	}
	marked, ok := p.markedUses.Load(ev.ToolResult.ToolUseID)
	if !ok || marked != origin {
		return
	}
	for {
		old := p.receiptedIn.Load()
		if origin <= old || p.receiptedIn.CompareAndSwap(old, origin) {
			return
		}
	}
}

// beginTerminalProofCall stamps a new model-call sequence and returns it.
func (t *ToolSet) beginTerminalProofCall() int64 { return t.terminalProof.current.Add(1) }

// recordTerminalProofOrigin binds a tool_use ID to its originating model call.
// Called by the deps wrappers when the tool_use identity first appears (stream
// event or assembled sync message), strictly before the executor starts it.
func (t *ToolSet) recordTerminalProofOrigin(toolUseID string, seq int64) {
	if toolUseID != "" {
		t.terminalProof.origin.Store(toolUseID, seq)
	}
}

// terminalProofCallSucceeded records that call seq completed without error.
func (t *ToolSet) terminalProofCallSucceeded(seq int64) {
	for {
		old := t.terminalProof.okCalls.Load()
		if seq <= old || t.terminalProof.okCalls.CompareAndSwap(old, seq) {
			return
		}
	}
}

// terminalProofCallFailed records a model or authentic tool-result error, invalidating
// armings from that call and earlier.
func (t *ToolSet) terminalProofCallFailed(seq int64) {
	for {
		old := t.terminalProof.failedThrough.Load()
		if seq <= old || t.terminalProof.failedThrough.CompareAndSwap(old, seq) {
			return
		}
	}
}

// terminalProofLocalTurn is the content-free local completion turn: a single
// end_turn delta with no text, no thinking, no tool calls and zero usage. The
// accumulator assembles an empty assistant message, the loop takes the native
// no-tool completion path, and nothing is attributed to the provider.
func terminalProofLocalTurn() iter.Seq2[llm.StreamEvent, error] {
	return func(yield func(llm.StreamEvent, error) bool) {
		yield(llm.StreamEvent{Type: llm.SEMessageDelta, StopReason: "end_turn"}, nil)
	}
}

// installTerminalProofGate composes the terminal-proof gate into deps and wraps
// the session's compactor (see terminal_proof.go). Non-nil existing CallModel/
// CallModelSync stay wrapped beneath the gate; nil entries are seeded from the
// provider (matching harness.QueryDeps.withDefaults).
func installTerminalProofGate(deps harness.QueryDeps, tsx *ToolSet, provider llm.Provider) harness.QueryDeps {
	stream, complete := deps.CallModel, deps.CallModelSync
	if stream == nil && provider != nil {
		stream = provider.Stream
	}
	if complete == nil && provider != nil {
		complete = provider.Complete
	}
	deps.CallModel = func(ctx context.Context, req llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
		if tsx != nil && tsx.terminalProofReady() && ctx.Err() == nil {
			return terminalProofLocalTurn() // successful boundary reached: no upstream request
		}
		if tsx == nil {
			return stream(ctx, req)
		}
		seq := tsx.beginTerminalProofCall()
		inner := stream(ctx, req)
		return func(yield func(llm.StreamEvent, error) bool) {
			failed := false
			for ev, err := range inner {
				if err != nil {
					// Record immediately: the harness consumer breaks on error events,
					// so post-loop bookkeeping would never run.
					failed = true
					tsx.terminalProofCallFailed(seq)
				}
				if ev.Type == llm.SEToolUseStart {
					// Bind the tool_use identity to this call BEFORE the executor can
					// start it (finalizeTool runs on the next event).
					tsx.recordTerminalProofOrigin(ev.ToolID, seq)
				}
				if !yield(ev, err) {
					return // consumer stopped; run is over either way, no success verdict
				}
			}
			if !failed {
				tsx.terminalProofCallSucceeded(seq)
			}
		}
	}
	deps.CallModelSync = func(ctx context.Context, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
		if tsx != nil && tsx.terminalProofReady() && ctx.Err() == nil {
			return llm.Message{Role: llm.RoleAssistant}, "end_turn", llm.Usage{}, nil
		}
		if tsx == nil {
			return complete(ctx, req)
		}
		seq := tsx.beginTerminalProofCall()
		msg, reason, usage, err := complete(ctx, req)
		if err != nil {
			tsx.terminalProofCallFailed(seq)
			return msg, reason, usage, err
		}
		// Bind every tool_use identity in the assembled message to this call
		// before the replay path can start executing them.
		for _, block := range msg.ToolUses() {
			tsx.recordTerminalProofOrigin(block.ID, seq)
		}
		tsx.terminalProofCallSucceeded(seq)
		return msg, reason, usage, err
	}
	return deps
}

// terminalProofCompactor skips only unnecessary post-proof Pre; Reactive and
// IsOverflow (overflow recovery) are always forwarded.
type terminalProofCompactor struct {
	inner harness.Compactor
	tsx   *ToolSet
}

func (c *terminalProofCompactor) Pre(ctx context.Context, msgs []llm.Message, lastInputTokens int) []llm.Message {
	if c.tsx.terminalProofReady() && ctx.Err() == nil {
		return msgs // successful proof boundary: no post-proof summarization inference
	}
	return c.inner.Pre(ctx, msgs, lastInputTokens)
}
func (c *terminalProofCompactor) Reactive(ctx context.Context, msgs []llm.Message) ([]llm.Message, bool) {
	return c.inner.Reactive(ctx, msgs)
}
func (c *terminalProofCompactor) IsOverflow(err error) bool { return c.inner.IsOverflow(err) }

// terminalProofViewCompactor preserves the optional ContextView of the wrapped
// compactor (requestMessages type-asserts the top-level Compactor).
type terminalProofViewCompactor struct{ *terminalProofCompactor }

func (c terminalProofViewCompactor) View(ctx context.Context, msgs []llm.Message) []llm.Message {
	return c.inner.(harness.ContextView).View(ctx, msgs)
}

func newTerminalProofCompactor(inner harness.Compactor, tsx *ToolSet) harness.Compactor {
	base := &terminalProofCompactor{inner: inner, tsx: tsx}
	if _, ok := inner.(harness.ContextView); ok {
		return terminalProofViewCompactor{base}
	}
	return base
}

// wrapTerminalProofCompactor wraps the session's selected compactor so proactive
// Pre is skipped exactly when the terminal-proof gate would fire. Selection
// follows the SDK contract: an explicit Compactor wins (Compaction cleared);
// otherwise a Compaction config builds precisely the SDK default
// (compaction.New with ProviderSummarizer) which is then wrapped; when neither
// is set, compaction stays absent. Native/noa concrete types are already chosen
// by the caller before this runs — only the final value is wrapped.
func wrapTerminalProofCompactor(opts *agentcore.Options, tsx *ToolSet) {
	switch {
	case opts.Compactor != nil:
		opts.Compactor = newTerminalProofCompactor(opts.Compactor, tsx)
		opts.Compaction = nil // mirror the SDK's mutual exclusion
	case opts.Compaction != nil:
		cfg := *opts.Compaction
		opts.Compactor = newTerminalProofCompactor(
			compaction.New(cfg, compaction.ProviderSummarizer(opts.Provider, opts.MaxTokens, opts.NonStreaming)), tsx)
		opts.Compaction = nil
	}
}

// installTerminalProofGateOn composes the gate and compactor wrap into an
// agentcore.Options value (used by planner.Plan after all compactor selection).
// Its observer must be passed to captureRun so receipts qualify before any next
// compaction/model boundary; it never manufactures a successful tool result.
func installTerminalProofGateOn(opts *agentcore.Options, tsx *ToolSet) func(harness.Event) {
	opts.Deps = installTerminalProofGate(opts.Deps, tsx, opts.Provider)
	wrapTerminalProofCompactor(opts, tsx)
	return tsx.observeTerminalProofEvent
}
