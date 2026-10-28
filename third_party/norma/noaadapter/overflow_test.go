// Modified by ScopeWeaver on 2026-10-25: assert overflow learning, monotonic
// ceilings and refusal through real retry views instead of skipping.
// Also verify output errors bypass recovery through the actual harness call chain.
package noaadapter

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"testing"

	"github.com/Autumn-27/norma/harness"
	"github.com/Autumn-27/norma/llm"
	"github.com/Autumn-27/norma/noa"
)

// The three harness-facing methods are the whole public surface between noa and
// the agent loop, and until now only View was exercised.

// Pre must not touch the history. The built-in compaction rewrites messages
// here; noa's message identity is a content hash, so a rewrite would change
// every id and invalidate every ref the model is holding.
func TestPreLeavesTheHistoryUntouched(t *testing.T) {
	sess := simSession(t, 40000)
	c := &Compactor{sess: sess}
	in := []llm.Message{userText("one"), userText("two")}

	out := c.Pre(t.Context(), in, 12345)
	if len(out) != len(in) {
		t.Fatalf("Pre returned %d messages, want the %d it was given", len(out), len(in))
	}
	for i := range in {
		if out[i].Text() != in[i].Text() {
			t.Fatalf("Pre rewrote message %d: %q -> %q", i, in[i].Text(), out[i].Text())
		}
	}
	// The provider's own count is more accurate than any estimate, so it must be
	// taken when offered — resolveTokenCount prefers it over estimateCoreTokens
	// whenever it is fresh and close.
	sess.mu.Lock()
	got := sess.providerTokens
	sess.mu.Unlock()
	if got != 12345 {
		t.Fatalf("Pre recorded providerTokens = %d, want 12345; the pressure ladder keeps "+
			"running on estimates when a measured number was available", got)
	}
}

// A zero count means the provider did not report one — it must not be recorded
// as "the context is empty".
func TestPreIgnoresAnUnreportedTokenCount(t *testing.T) {
	sess := simSession(t, 40000)
	c := &Compactor{sess: sess}
	c.Pre(t.Context(), nil, 7000)
	before := sess.tokensBefore()
	c.Pre(t.Context(), nil, 0)
	if sess.tokensBefore() != before {
		t.Fatalf("a zero token count overwrote the last known size %d with %d",
			before, sess.tokensBefore())
	}
}

// IsOverflow decides whether Reactive runs at all. Getting it wrong in one
// direction wastes a recovery attempt; in the other it turns a recoverable
// prompt-too-long into a failed run.
func TestIsOverflowRecognisesProviderPhrasings(t *testing.T) {
	c := &Compactor{sess: simSession(t, 40000)}
	overflow := []string{
		"This model's maximum context length is 200000 tokens, however you requested 214000",
		"prompt is too long: 215000 tokens > 200000 maximum",
		"error: context_length_exceeded",
		"anthropic: status 413",
	}
	for _, msg := range overflow {
		if !c.IsOverflow(errors.New(msg)) {
			t.Errorf("IsOverflow(%q) = false; the run fails instead of recovering", msg)
		}
	}
	other := []string{"rate limit exceeded", "connection reset by peer", "invalid api key"}
	for _, msg := range other {
		if c.IsOverflow(errors.New(msg)) {
			t.Errorf("IsOverflow(%q) = true; a recovery attempt is spent on an unrelated failure", msg)
		}
	}
	if c.IsOverflow(nil) {
		t.Error("IsOverflow(nil) = true")
	}
}

// overflowFixture sends one view of an uncompressed session on a configured
// 200K window, so Reactive has a real rejected request to measure against.
// outputChars picks whether the valve can cut the tool results (large) or not
// (medium, below both truncation criteria).
func overflowFixture(t *testing.T, outputChars, turns int) (*Compactor, []llm.Message, int) {
	t.Helper()
	sess := simSession(t, 200_000)
	a := newSimAgent(t, sess, 1_000_000)
	for range turns {
		a.work(outputChars)
	}
	c := &Compactor{sess: sess}
	sent := estimateMessages(c.View(t.Context(), a.history))
	if sent <= 121_600 || sent >= 190_000 {
		t.Fatalf("fixture sent %d tokens; it must sit between the learned (121600) and "+
			"configured (190000) ceilings to tell them apart", sent)
	}
	return c, a.history, sent
}

func historyBytes(t *testing.T, msgs []llm.Message) string {
	t.Helper()
	b, err := json.Marshal(msgs)
	if err != nil {
		t.Fatalf("marshal history: %v", err)
	}
	return string(b)
}

// overflowState documents itself as mining the error for the provider's real
// window — "rather than guessing again next turn, the error is mined for the
// actual number and the mechanical valve is armed so the next view is built to
// fit."
//
// ScopeWeaver: upstream left ParseOverflowWindow and overflowState.learn without
// callers, so arm() fell back to the CONFIGURED limit the provider had just
// rejected (upstream v0.4.3 armed 190000 here, not 121600). IsOverflow now
// learns from errors that unmistakably state the context window. The previous
// version of this test skipped when Reactive disarmed on an empty view; this one
// uses a real rejected request and asserts the retry view itself.
func TestOverflowLearningIsWiredIn(t *testing.T) {
	c, history, sent := overflowFixture(t, 40_000, 14)
	raw := historyBytes(t, history)

	// The provider says its real window is 128000, not the configured 200000.
	err := errors.New("This model's maximum context length is 128000 tokens, however you requested 214000")
	if n := ParseOverflowWindow(err.Error()); n != 128000 {
		t.Fatalf("ParseOverflowWindow = %d, want 128000 — the parser itself is broken", n)
	}

	// The loop's actual sequence: detect, then recover.
	if !c.IsOverflow(err) {
		t.Fatal("IsOverflow rejected a context-length error")
	}
	out, ok := c.Reactive(t.Context(), history)
	if !ok {
		t.Fatalf("Reactive refused a %d-token request the valve can cut below the learned ceiling", sent)
	}
	want := int(float64(128000) * 0.95)
	if floor := c.overflow.armedFloor(); floor != want {
		t.Fatalf("the armed floor is %d, not %d (95%% of the %d the provider reported); "+
			"the next view is rebuilt to a limit the provider has already rejected",
			floor, want, 128000)
	}
	retry := estimateMessages(c.View(t.Context(), out))
	if retry == 0 || retry > want || retry >= sent {
		t.Fatalf("retry view is %d tokens; want 0 < view <= %d and below the rejected %d", retry, want, sent)
	}
	if historyBytes(t, out) != raw || historyBytes(t, history) != raw {
		t.Fatal("Reactive or the retry view rewrote the raw history")
	}
}

// A learned ceiling only ever tightens. A larger later claim, a claim above the
// configured window, and a genuine overflow that names no window all keep the
// 121600 ceiling; an unrelated error is not an overflow at all.
func TestLearnedWindowIsMonotonic(t *testing.T) {
	c, history, _ := overflowFixture(t, 40_000, 14)
	c.IsOverflow(errors.New("maximum context length is 128000 tokens"))
	if _, ok := c.Reactive(t.Context(), history); !ok {
		t.Fatal("Reactive refused the first recovery")
	}
	const want = 121_600

	later := []string{
		"This model's maximum context length is 150000 tokens",
		"This model's maximum context length is 300000 tokens",
		"prompt is too long",
	}
	for _, msg := range later {
		if !c.IsOverflow(errors.New(msg)) {
			t.Fatalf("IsOverflow(%q) = false; a genuine overflow lost its recovery", msg)
		}
		if c.overflow.learnedWindow != 128_000 || c.overflow.armedFloor() != want {
			t.Fatalf("after %q: learned %d, ceiling %d; want 128000 and %d",
				msg, c.overflow.learnedWindow, c.overflow.armedFloor(), want)
		}
		if v := estimateMessages(c.View(t.Context(), history)); v > want {
			t.Fatalf("after %q: retry view %d exceeds the learned ceiling %d", msg, v, want)
		}
	}

	// The bounded view cannot shrink again, so the next recovery refuses and
	// disarms; the next request is built to the configured window again.
	if _, ok := c.Reactive(t.Context(), history); ok {
		t.Fatal("Reactive claimed it shrank an already bounded view")
	}
	sent := estimateMessages(c.View(t.Context(), history))
	if sent <= want {
		t.Fatalf("disarmed view is %d tokens; the fixture no longer exceeds the learned ceiling", sent)
	}
	// A later, larger claim re-arms at the learned ceiling, not at 95%% of 150000.
	c.IsOverflow(errors.New("This model's maximum context length is 150000 tokens"))
	if _, ok := c.Reactive(t.Context(), history); !ok {
		t.Fatalf("Reactive refused a %d-token request it can cut below %d", sent, want)
	}
	if floor := c.overflow.armedFloor(); floor != want {
		t.Fatalf("re-armed ceiling %d, want the learned %d", floor, want)
	}
	if v := estimateMessages(c.View(t.Context(), history)); v > want || v >= sent {
		t.Fatalf("re-armed retry view %d; want <= %d and below the rejected %d", v, want, sent)
	}

	if c.IsOverflow(errors.New("rate limit exceeded (max 64000 tokens per minute)")) {
		t.Fatal("a rate limit was classified as an overflow; recovery would run on it")
	}
	if c.overflow.learnedWindow != 128_000 {
		t.Fatalf("an unrelated error changed the learned window to %d", c.overflow.learnedWindow)
	}
}

// The inherited classifier treats any "too long" as an overflow, and the parser
// reads max_tokens values that usually name the OUTPUT cap. An explicit output
// error must neither teach a context window nor invoke input-context recovery.
func TestOutputCapErrorDoesNotTeachAWindow(t *testing.T) {
	c, _, _ := overflowFixture(t, 40_000, 14)
	err := errors.New("output is too long: max_tokens 8192 exceeded for this model")
	if ParseOverflowWindow(err.Error()) != 8192 {
		t.Fatal("fixture no longer exercises the max_tokens pattern")
	}
	if c.IsOverflow(err) {
		t.Fatal("an explicit output-cap error must not invoke input-context recovery")
	}
	if c.overflow.learnedWindow != 0 {
		t.Fatalf("learned a %d-token context window from an output-cap error", c.overflow.learnedWindow)
	}
	if c.overflow.armedFloor() != 0 || c.sess.emergencyFloor != 0 {
		t.Fatal("an output-cap error changed context-limit state")
	}
}

type countedOverflowCompactor struct {
	*Compactor
	recoveries int
}

func (c *countedOverflowCompactor) Reactive(ctx context.Context, msgs []llm.Message) ([]llm.Message, bool) {
	c.recoveries++
	return c.Compactor.Reactive(ctx, msgs)
}

// Exercise Query's real IsOverflow -> Reactive -> retry seam, including the
// request views sent to the controlled provider and the terminal error.
func TestHarnessRecoversContextErrorsButPreservesOutputErrors(t *testing.T) {
	for _, tc := range []struct {
		name       string
		message    string
		contextErr bool
		floor      int
		fullWindow bool
		refuses    bool
	}{
		{name: "output cap", message: "output is too long: max_tokens 8192 exceeded for this model"},
		{name: "output length", message: "generated output length is too long"},
		{name: "parsed context", message: "maximum context length is 128000 tokens", contextErr: true, floor: 121600},
		{name: "unparseable context", message: "prompt is too long", contextErr: true, floor: 190000, fullWindow: true},
		{name: "context mentions output", message: "prompt is too long for the context window with requested output", contextErr: true, floor: 190000, fullWindow: true},
		{name: "output endpoint context error", message: "output endpoint: request is too long", contextErr: true, floor: 190000, fullWindow: true},
		{name: "unparseable context cannot shrink", message: "prompt is too long", contextErr: true, refuses: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var inner *Compactor
			var history []llm.Message
			if tc.fullWindow {
				// Allow the first view up to the configured window. A generic
				// overflow can then genuinely shrink it to the 95% recovery ceiling.
				cfg := noa.DefaultConfig(200000)
				cfg.Truncate.Threshold = 1
				sess, err := newSession(Options{ArchiveBaseDir: t.TempDir(), SessionID: "query-chain", Config: &cfg})
				if err != nil {
					t.Fatal(err)
				}
				a := newSimAgent(t, sess, 1_000_000)
				for range 19 {
					a.work(40000)
				}
				inner, history = &Compactor{sess: sess}, a.history
				sent := estimateMessages(inner.View(t.Context(), history))
				if sent <= 190000 || sent >= 200000 {
					t.Fatalf("generic-overflow fixture sent %d tokens, want strictly between 190000 and 200000", sent)
				}
			} else {
				inner, history, _ = overflowFixture(t, 40000, 14)
			}
			c := &countedOverflowCompactor{Compactor: inner}
			original := historyBytes(t, history)
			providerErr := errors.New(tc.message)
			var sent []int
			call := func(_ context.Context, req llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
				return func(yield func(llm.StreamEvent, error) bool) {
					sent = append(sent, estimateMessages(req.Messages))
					if len(sent) == 1 {
						yield(llm.StreamEvent{}, providerErr)
						return
					}
					if !yield(llm.StreamEvent{Type: llm.SETextDelta, Text: "done"}, nil) {
						return
					}
					yield(llm.StreamEvent{Type: llm.SEMessageDelta, StopReason: "end_turn"}, nil)
				}
			}
			var terminal *harness.Terminal
			for event, err := range harness.Query(t.Context(), harness.QueryInput{
				Messages: history, Compactor: c,
			}, harness.QueryDeps{CallModel: call}) {
				if err != nil {
					t.Fatal(err)
				}
				if event.Kind == harness.KindResult {
					terminal = event.Terminal
				}
			}
			if terminal == nil {
				t.Fatal("no terminal result")
			}
			if tc.contextErr {
				if tc.refuses {
					if c.recoveries != 1 || len(sent) != 1 || terminal.Reason != harness.ReasonPromptTooLong || terminal.Err != providerErr {
						t.Fatalf("unshrinkable context error must fail honestly: recoveries=%d sent=%v reason=%s err=%v", c.recoveries, sent, terminal.Reason, terminal.Err)
					}
					if c.overflow.armedFloor() != 0 || c.sess.emergencyFloor != 0 {
						t.Fatal("refused recovery left the mechanical valve armed")
					}
				} else {
					if c.recoveries != 1 || len(sent) != 2 || terminal.Reason != harness.ReasonCompleted || terminal.Err != nil {
						t.Fatalf("genuine context error did not recover: recoveries=%d sent=%v reason=%s err=%v", c.recoveries, sent, terminal.Reason, terminal.Err)
					}
					if sent[1] >= sent[0] || sent[1] > tc.floor {
						t.Fatalf("retry view %d must shrink from %d and fit %d", sent[1], sent[0], tc.floor)
					}
				}
			} else {
				if c.recoveries != 0 || len(sent) != 1 || terminal.Reason != harness.ReasonModelError || terminal.Err != providerErr {
					t.Fatalf("output error invoked context recovery or changed the error: recoveries=%d sent=%v reason=%s err=%v", c.recoveries, sent, terminal.Reason, terminal.Err)
				}
				if c.overflow.learnedWindow != 0 || c.overflow.armedFloor() != 0 || c.sess.emergencyFloor != 0 {
					t.Fatal("output error changed context-limit state")
				}
			}
			if historyBytes(t, history) != original {
				t.Fatal("error recovery rewrote raw history")
			}
			t.Logf("recoveries=%d sent_tokens=%v terminal=%s", c.recoveries, sent, terminal.Reason)
		})
	}
}

// The parser itself, over the phrasings it claims to cover.
func TestParseOverflowWindowCoversItsPatterns(t *testing.T) {
	cases := map[string]int{
		"This model's maximum context length is 200000 tokens": 200000,
		"context window of 131072 exceeded":                    131072,
		"max_tokens must be less than 8192":                    8192,
		"requests are limited to 32000 tokens":                 32000,
		"rate limit exceeded":                                  0,
		"maximum context length is zero":                       0,
	}
	for msg, want := range cases {
		if got := ParseOverflowWindow(msg); got != want {
			t.Errorf("ParseOverflowWindow(%q) = %d, want %d", msg, got, want)
		}
	}
}

// Reactive must never claim success it cannot deliver: returning true promises
// the harness a smaller request, and if the request is not smaller the retry
// hits the same 413 and the loop spins.
//
// ScopeWeaver: the previous version used a one-message view that Reactive
// accepts (nothing was sent yet), so it permitted either outcome. This one sends
// a real request of medium messages the valve cannot cut, learns a smaller
// window, and requires an explicit refusal.
func TestReactiveRefusesWhenItCannotShrink(t *testing.T) {
	c, history, sent := overflowFixture(t, 4_000, 140)
	raw := historyBytes(t, history)

	c.IsOverflow(errors.New("This model's maximum context length is 128000 tokens"))
	out, ok := c.Reactive(context.Background(), history)
	if ok {
		t.Fatalf("Reactive accepted a %d-token request it cannot shrink below 121600; "+
			"that produces a 413 retry loop", sent)
	}
	// A refusal that left the valve armed would shrink every later view for no
	// reason.
	if c.overflow.armed || c.overflow.armedFloor() != 0 {
		t.Fatal("refused but left the valve armed")
	}
	c.sess.mu.Lock()
	floor := c.sess.emergencyFloor
	c.sess.mu.Unlock()
	if floor != 0 {
		t.Fatalf("refused but left the session's emergency floor at %d", floor)
	}
	if historyBytes(t, out) != raw || historyBytes(t, history) != raw {
		t.Fatal("a refused recovery rewrote the raw history")
	}
}
