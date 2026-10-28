package agent

import (
	"context"
	"encoding/json"
	"iter"
	"strings"
	"testing"
	"time"

	"github.com/Autumn-27/norma/agentcore"
	"github.com/Autumn-27/norma/harness"
	"github.com/Autumn-27/norma/hook"
	"github.com/Autumn-27/norma/llm"
	"github.com/Autumn-27/norma/permission"
	"github.com/Autumn-27/norma/tool"
)

func TestCaptureRunCancellationDoesNotJoinSDKCallbacks(t *testing.T) {
	for _, stage := range []string{"pre_hook", "tool", "post_hook"} {
		t.Run(stage, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			entered, release, callbackDone := make(chan struct{}), make(chan struct{}), make(chan struct{})
			finished := make(chan struct{})
			released := false
			defer func() {
				cancel()
				if !released {
					close(release)
				}
				select {
				case <-callbackDone:
				case <-time.After(5 * time.Second):
					t.Error("callback did not finish")
				}
				select {
				case <-finished:
				case <-time.After(5 * time.Second):
					t.Error("capture did not finish")
				}
			}()
			hold := func() {
				close(entered)
				<-release
				close(callbackDone)
			}
			hooks := hook.NewRegistry()
			hooks.On(hook.PreToolUse, func(context.Context, hook.Event) hook.Result {
				if stage == "pre_hook" {
					hold()
				}
				return hook.Result{}
			})
			hooks.On(hook.PostToolUse, func(context.Context, hook.Event) hook.Result {
				if stage == "post_hook" {
					hold()
				}
				return hook.Result{}
			})
			probe := tool.Build(tool.Spec{Name: "OwnershipProbe", Schema: map[string]any{"type": "object"},
				Run: func(context.Context, json.RawMessage, *tool.ToolContext) (tool.Result, error) {
					if stage == "tool" {
						hold()
					}
					return tool.Text("done"), nil
				},
			})
			provider := captureUsageProvider{stream: func(_ context.Context, yield func(llm.StreamEvent, error) bool) {
				for _, event := range []llm.StreamEvent{
					{Type: llm.SEToolUseStart, ToolID: "ownership-call", ToolName: probe.Name()},
					{Type: llm.SEToolInputJSON, Text: `{}`},
					{Type: llm.SEMessageDelta, StopReason: "tool_use"},
				} {
					if !yield(event, nil) {
						return
					}
				}
			}}
			go func() {
				defer close(finished)
				_, _, _ = captureRun(ctx, agentcore.Options{Provider: provider, Tools: []tool.CoreTool{probe}, Hooks: hooks,
					PermissionMode: permission.ModeBypass, WorkingDir: t.TempDir(), MaxTurns: 1}, "hold callback", nil)
			}()
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("SDK callback did not start")
			}
			cancel()
			select {
			case <-finished:
			case <-time.After(5 * time.Second):
				t.Error("ordinary cancellation waited for an SDK callback")
			}
			select {
			case <-callbackDone:
				t.Error("blocked callback completed before release")
			default:
			}
			close(release)
			released = true
		})
	}
}

func TestCaptureRunPreservesHookLifecycle(t *testing.T) {
	var events []string
	hooks := hook.NewRegistry()
	hooks.On(hook.SessionStart, func(context.Context, hook.Event) hook.Result {
		events = append(events, "session")
		return hook.Result{}
	})
	hooks.On(hook.UserPromptSubmit, func(context.Context, hook.Event) hook.Result {
		events = append(events, "prompt")
		return hook.Result{AdditionalContext: "injected prompt context"}
	})
	hooks.On(hook.PreToolUse, func(_ context.Context, event hook.Event) hook.Result {
		events = append(events, "pre")
		if string(event.Input) != "malformed input" {
			t.Errorf("raw input changed: %s", event.Input)
		}
		return hook.Result{UpdatedInput: json.RawMessage(`{"rewritten":true}`)}
	})
	hooks.On(hook.PostToolUse, func(_ context.Context, event hook.Event) hook.Result {
		events = append(events, "post")
		if string(event.Input) != `{"rewritten":true}` || event.IsError || !strings.Contains(string(event.Result), "probe result") {
			t.Errorf("post-hook arguments changed: %+v", event)
		}
		return hook.Result{}
	})
	hooks.On(hook.Stop, func(context.Context, hook.Event) hook.Result {
		events = append(events, "stop")
		return hook.Result{}
	})
	probe := tool.Build(tool.Spec{Name: "LifecycleProbe", Schema: map[string]any{"type": "object"}, RawInput: true,
		Run: func(_ context.Context, input json.RawMessage, _ *tool.ToolContext) (tool.Result, error) {
			events = append(events, "call")
			if string(input) != `{"rewritten":true}` {
				t.Errorf("pre-hook rewrite lost: %s", input)
			}
			return tool.Text("probe result"), nil
		},
	})
	turn := 0
	provider := captureUsageProvider{stream: func(_ context.Context, yield func(llm.StreamEvent, error) bool) {
		turn++
		events := []llm.StreamEvent{{Type: llm.SETextDelta, Text: "done"}, {Type: llm.SEMessageDelta, StopReason: "end_turn"}}
		if turn == 1 {
			events = []llm.StreamEvent{{Type: llm.SEToolUseStart, ToolID: "lifecycle-call", ToolName: probe.Name()},
				{Type: llm.SEToolInputJSON, Text: "malformed input"}, {Type: llm.SEMessageDelta, StopReason: "tool_use"}}
		}
		for _, event := range events {
			if !yield(event, nil) {
				return
			}
		}
	}}
	opts := agentcore.Options{Tools: []tool.CoreTool{probe}, Hooks: hooks, PermissionMode: permission.ModeBypass,
		WorkingDir: t.TempDir(), MaxTurns: 2, Deps: harness.QueryDeps{
			CallModel: func(ctx context.Context, req llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
				if turn == 0 && !strings.Contains(req.Messages[len(req.Messages)-1].Text(), "injected prompt context") {
					t.Error("prompt-submission context was not forwarded")
				}
				return provider.Stream(ctx, req)
			},
		}}
	if _, _, err := captureRun(t.Context(), opts, "run the probe", nil); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(events, ","); got != "session,prompt,pre,call,post,stop" {
		t.Fatalf("hook lifecycle=%s", got)
	}
}
