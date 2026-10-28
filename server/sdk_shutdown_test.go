package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Autumn-27/artex/agent"
	"github.com/Autumn-27/artex/guard"
	"github.com/Autumn-27/norma/hook"
	"github.com/Autumn-27/norma/llm"
	"github.com/Autumn-27/norma/tool"
)

func newSDKShutdownServer(t *testing.T) (*Server, context.CancelFunc) {
	t.Helper()
	m := backportTestManager(t)
	oldAugment, oldResolve, oldPrompt := agent.ToolAugment, agent.ToolResolve, agent.PromptOverride
	oldWrapup, oldTurns := agent.WrapupOverride, agent.WrapupMaxTurnsOverride
	oldTimeout, oldTimeoutTurns := agent.WrapupTaskTimeoutOverride, agent.WrapupTaskTimeoutTurnsOverride
	oldBinding := agent.FindingTrafficBindingEnabled
	ctx, cancel := context.WithCancel(context.Background())
	dir := t.TempDir()
	// The production constructor must supply callback ownership through s.ctx.
	// This fixture also compiles on the baseline, which has no ownership API.
	s := New(ctx, m, dir, dir, dir)
	agent.ToolResolve, agent.PromptOverride = nil, nil
	t.Cleanup(func() {
		cancel()
		drainCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := s.DrainChats(drainCtx); err != nil {
			t.Errorf("cleanup drain: %v", err)
		}
		agent.ToolAugment, agent.ToolResolve, agent.PromptOverride = oldAugment, oldResolve, oldPrompt
		agent.WrapupOverride, agent.WrapupMaxTurnsOverride = oldWrapup, oldTurns
		agent.WrapupTaskTimeoutOverride, agent.WrapupTaskTimeoutTurnsOverride = oldTimeout, oldTimeoutTurns
		agent.FindingTrafficBindingEnabled = oldBinding
	})
	return s, cancel
}

func TestShutdownWaitsForSDKToolWrite(t *testing.T) {
	for _, tc := range []struct {
		name, stage string
		deferred    bool
	}{
		{"direct", "tool", false},
		{"deferred", "tool", true},
		{"pre_hook", "pre_hook", false},
		{"post_hook", "post_hook", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testSDKShutdownWrite(t, tc.stage, tc.deferred, false)
		})
	}
}

func TestSDKCallbackSurvivesOrdinaryCancellationUntilShutdown(t *testing.T) {
	testSDKShutdownWrite(t, "tool", false, true)
}

func testSDKShutdownWrite(t *testing.T, stage string, deferred, ordinaryCancel bool) {
	t.Helper()
	s, cancel := newSDKShutdownServer(t)
	lock, err := s.m.pg.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	finished := make(chan error, 1)
	started := false
	var conversationID int64
	defer func() {
		cancel()
		_ = lock.Rollback()
		// Join the baseline's detached callback only in cleanup. Assertions below
		// rely on DrainChats, without polling or joining before reading the row.
		if started {
			select {
			case err := <-finished:
				if err != nil {
					t.Errorf("callback write: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Error("SDK callback write did not finish during cleanup")
			}
		}
		drainCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := s.DrainChats(drainCtx); err != nil {
			t.Errorf("cleanup drain: %v", err)
		}
		_, _ = s.m.pg.Exec(`DELETE FROM task_categories WHERE name=$1`, t.Name())
		_ = s.m.pg.DeleteConversation(conversationID)
	}()
	write := func() error {
		// This is the same non-context store API used by production handlers.
		_, err := s.m.pg.CreateTaskCategory(t.Name())
		finished <- err
		return err
	}
	probe := tool.Build(tool.Spec{Name: "ShutdownWriteProbe", Schema: map[string]any{"type": "object"},
		Run: func(context.Context, json.RawMessage, *tool.ToolContext) (tool.Result, error) {
			if stage == "tool" {
				if err := write(); err != nil {
					return tool.Result{}, err
				}
			}
			return tool.Text("probe complete"), nil
		},
	})
	agent.ToolAugment = func(context.Context, string) ([]tool.CoreTool, agent.DeferredInfo, func()) {
		def := agent.DeferredInfo{}
		if deferred {
			def.Deferred = []string{probe.Name()}
		}
		return []tool.CoreTool{probe}, def, func() {}
	}
	calls := 0
	setRetestProvider(s, retestProvider{complete: func(context.Context, llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
		calls++
		if calls > 1 {
			return llm.Message{Role: llm.RoleAssistant, Content: []llm.ContentBlock{llm.TextBlock("done")}}, "end_turn", llm.Usage{}, nil
		}
		name, input := probe.Name(), `{}`
		if deferred {
			name, input = "ExecuteExtraTool", `{"tool_name":"ShutdownWriteProbe","params":{}}`
		}
		return llm.Message{Role: llm.RoleAssistant, Content: []llm.ContentBlock{{Type: llm.BlockToolUse, ID: "shutdown-write", Name: name, Input: json.RawMessage(input)}}}, "tool_use", llm.Usage{}, nil
	}})
	g := guard.New()
	if stage == "pre_hook" || stage == "post_hook" {
		event := hook.PreToolUse
		if stage == "post_hook" {
			event = hook.PostToolUse
		}
		g.Hooks().On(event, func(context.Context, hook.Event) hook.Result {
			_ = write()
			return hook.Result{}
		})
	}
	s.chatAgent.SetGuard(g)
	conversation, err := s.m.pg.CreateConversation("auto", "SDK shutdown fixture", nil)
	if err != nil {
		t.Fatal(err)
	}
	conversationID = conversation.ID
	if _, err := lock.Exec(`LOCK TABLE task_categories IN SHARE MODE`); err != nil {
		t.Fatal(err)
	}
	send := retestRequest(s.pgSendConversationMessage, http.MethodPost, conversation.ID, `{"message":"save the category"}`)
	if send.Code != http.StatusAccepted {
		t.Fatalf("send: %d %s", send.Code, send.Body)
	}
	started = true
	waitForShutdownTaskWrite(t, s.m, "%INSERT INTO task_categories(name,%")
	if ordinaryCancel {
		s.cancelConversation(conversation.ID)
		waitRetestIdle(t, s)
		if s.ctx.Err() != nil {
			t.Error("ordinary conversation cancellation stopped the server")
		}
		select {
		case <-finished:
			t.Fatal("blocked callback finished before its lock was released")
		default:
		}
	}
	cancel()
	drainCtx, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	err = s.DrainChats(drainCtx)
	stop()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("drain returned while SDK %s held a database write: %v", stage, err)
	}
	if err := lock.Rollback(); err != nil {
		t.Fatal(err)
	}
	drainCtx, stop = context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := s.DrainChats(drainCtx); err != nil {
		t.Fatal(err)
	}
	var exists bool
	if err := s.m.pg.QueryRow(`SELECT EXISTS (SELECT 1 FROM task_categories WHERE name=$1)`, t.Name()).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("successful drain preceded SDK callback commit")
	}
}
