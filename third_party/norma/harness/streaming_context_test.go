// Added by ScopeWeaver on 2026-10-25: originating tool-context regression tests.
package harness

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Autumn-27/norma/llm"
	"github.com/Autumn-27/norma/permission"
	"github.com/Autumn-27/norma/tool"
)

func TestStreamExecRetainsOriginatingToolContext(t *testing.T) {
	parent := context.Background()
	origin, cancel := context.WithDeadline(parent, time.Now().Add(-time.Second))
	defer cancel()
	observed := make(chan context.Context, 2)
	ct := tool.Build(tool.Spec{
		Name: "record_context", Schema: map[string]any{"type": "object"},
		Run: func(ctx context.Context, _ json.RawMessage, _ *tool.ToolContext) (tool.Result, error) {
			observed <- ctx
			return tool.Text("recorded"), nil
		},
	})
	l := &loop{
		ctx: parent, toolCtx: origin, sem: make(chan struct{}, 1),
		in:    QueryInput{Tools: tool.NewRegistry(ct), PermissionMode: permission.ModeBypass},
		yield: func(Event, error) bool { return true },
	}
	l.sem <- struct{}{} // Hold the original execution before execOne reaches t.Call.
	original := newStreamExec(l)
	original.add(llm.ContentBlock{Type: llm.BlockToolUse, ID: "original", Name: ct.Name(), Input: json.RawMessage(`{}`)})
	l.toolCtx = parent // The loop enters settlement while the original execution is queued.
	<-l.sem
	assertContext := func(executor *streamExec, want context.Context, wantErr error) {
		t.Helper()
		select {
		case got := <-observed:
			if got != want || got.Err() != wantErr {
				t.Errorf("tool context = %v (err %v), want %v (err %v)", got, got.Err(), want, wantErr)
			}
		case <-time.After(time.Second):
			t.Fatal("tool did not execute")
		}
		select {
		case <-executor.notify:
		case <-time.After(time.Second):
			t.Fatal("executor did not complete")
		}
	}
	assertContext(original, origin, context.DeadlineExceeded)
	if stopped, aborted := original.finish(); stopped || !aborted {
		t.Errorf("original finish = (%v, %v), want (false, true)", stopped, aborted)
	}
	settlement := newStreamExec(l)
	settlement.add(llm.ContentBlock{Type: llm.BlockToolUse, ID: "settlement", Name: ct.Name(), Input: json.RawMessage(`{}`)})
	assertContext(settlement, parent, nil)
	if stopped, aborted := settlement.finish(); stopped || aborted {
		t.Errorf("settlement finish = (%v, %v), want (false, false)", stopped, aborted)
	}
}

type slowContextPostHook struct{}

func (slowContextPostHook) PreToolUse(context.Context, string, []byte) (bool, string, []byte) {
	return false, "", nil
}
func (slowContextPostHook) PostToolUse(context.Context, string, []byte, []byte, bool) {
	time.Sleep(120 * time.Millisecond)
}
func (slowContextPostHook) Stop(context.Context, []llm.Message) (bool, []string, string) {
	return false, nil, ""
}

// No synchronization orders tool execution against settlement's toolCtx rebind.
func TestSettlementAfterSlowPostToolUseKeepsToolContextRaceFree(t *testing.T) {
	m := &scriptedModel{turns: [][]llm.StreamEvent{
		toolTurn("original", "complete", `{}`), textTurn("settled"),
	}}
	ct := tool.Build(tool.Spec{
		Name: "complete", Schema: map[string]any{"type": "object"},
		Run: func(context.Context, json.RawMessage, *tool.ToolContext) (tool.Result, error) {
			return tool.Text("complete"), nil
		},
	})
	term := drainWithCtx(t, context.Background(), QueryInput{
		Messages: []llm.Message{llm.UserText("go")}, Tools: tool.NewRegistry(ct),
		PermissionMode: permission.ModeBypass, MaxDuration: 30 * time.Millisecond,
		Hooks: slowContextPostHook{}, Settlement: &Settlement{Prompt: "settle interrupted tool"},
	}, QueryDeps{CallModel: m.call})
	time.Sleep(150 * time.Millisecond) // Allow the unsynchronized late hook to finish.
	if term.Reason != ReasonTimeout || m.calls != 2 {
		t.Fatalf("reason=%v calls=%d, want timeout and two model calls", term.Reason, m.calls)
	}
	for _, message := range term.Messages {
		for _, block := range message.Content {
			if block.Type == llm.BlockToolResult && block.ToolUseID == "original" {
				if !block.IsError {
					t.Fatal("original tool result must record interruption")
				}
				return
			}
		}
	}
	t.Fatal("missing original tool result")
}
