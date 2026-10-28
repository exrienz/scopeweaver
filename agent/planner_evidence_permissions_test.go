package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/internal/dbtest"
	"github.com/Autumn-27/artex/locale"
	"github.com/Autumn-27/norma/llm"
	"github.com/Autumn-27/norma/permission"
	actool "github.com/Autumn-27/norma/tool"
)

// Independent actual Planner.Plan boundary controls. This file is separate
// from the immutable S1 oracle and refers to no candidate-only helper symbols.
// Baseline has no prefetch and should pass these controls; candidate must still
// pass while the positive S1 stock case requires useful full evidence.
func TestPlannerResolvedEvidenceBoundary(t *testing.T) {
	t.Setenv("AGENT_CORE_DISABLE_INTERACTIVE_SHELL", "1")
	priorAugment, priorResolve, priorPrompt, priorTraffic := ToolAugment, ToolResolve, PromptOverride, FindingTrafficBindingEnabled
	ToolAugment, ToolResolve, PromptOverride, FindingTrafficBindingEnabled = nil, nil, nil, nil
	t.Cleanup(func() {
		ToolAugment, ToolResolve, PromptOverride, FindingTrafficBindingEnabled = priorAugment, priorResolve, priorPrompt, priorTraffic
	})
	for _, mode := range []string{"dropped", "redacted", "read_only_false", "permission_deny", "permission_ask", "wrapper_deny", "handler_error", "tool_error", "nil_tool_context"} {
		t.Run(mode, func(t *testing.T) {
			d, err := db.Open(dbtest.DSN(t))
			if err != nil {
				t.Fatal(err)
			}
			defer d.Close()
			var dbname string
			if err := d.QueryRow("SELECT current_database()").Scan(&dbname); err != nil || !strings.HasPrefix(dbname, "scopeweaver_test_") {
				t.Fatal("not an owned disposable database")
			}
			row, err := d.CreateTask("R23 resolved boundary fixture", "Assess only actual permitted evidence; no automatic proof", nil, 0, 600)
			if err != nil {
				t.Fatal(err)
			}
			defer d.DeleteTask(row.ID)
			store := d.Exploration(row.ExplorationID)
			goal, err := store.AddGoal(map[string]any{"text": "a final objective remains unproven in boundary controls"}, "fixture")
			if err != nil {
				t.Fatal(err)
			}
			intent, err := store.AddIntent(map[string]any{"summary": "controlled fixture lineage"}, 5, nil, "fixture")
			if err != nil {
				t.Fatal(err)
			}
			const withheld = "R23_WITHHELD_RAW_FULL_ONLY_SENTINEL"
			fact, err := store.AddNode(db.KindFact, map[string]any{"summary": "compact summary intentionally remains visible", "detail": withheld, "evidence": withheld + " evidence", "confidence": "observed"}, 5, "confirmed", "fixture", nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Link(intent, db.RelYields, fact); err != nil {
				t.Fatal(err)
			}
			if err := store.SetIntentState(intent, "done"); err != nil {
				t.Fatal(err)
			}
			calls := 0
			fabricatedIdentity := false
			nilContext := false
			resolveSeen := false
			allowedResult := fmt.Sprintf("{\"id\":%d,\"kind\":\"fact\",\"payload\":{\"summary\":\"R23_ALLOWED_REDACTED_ONLY\"},\"state\":\"confirmed\"}", fact)
			ToolResolve = func(_ context.Context, key string, tools []actool.CoreTool) []actool.CoreTool {
				if key != "planner" {
					return tools
				}
				resolveSeen = true
				out := make([]actool.CoreTool, 0, len(tools))
				for _, tool := range tools {
					if tool.Name() != "node_detail" {
						out = append(out, tool)
						continue
					}
					if mode == "dropped" {
						continue
					}
					if mode == "wrapper_deny" {
						out = append(out, &testPlannerEvidenceDenyWrapper{CoreTool: tool, calls: &calls})
						continue
					}
					replacement := actool.Build(actool.Spec{
						Name: "node_detail", Description: "R23 controlled resolved replacement",
						Schema:   map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "integer"}}, "required": []string{"id"}},
						ReadOnly: func(json.RawMessage) bool { return mode != "read_only_false" },
						Permissions: func(context.Context, json.RawMessage, permission.Context) permission.Decision {
							if mode == "permission_deny" {
								return permission.Denied("controlled replacement denies automatic read")
							}
							if mode == "permission_ask" {
								return permission.AskUser("controlled replacement needs approval")
							}
							return permission.Allowed()
						},
						Run: func(ctx context.Context, _ json.RawMessage, tc *actool.ToolContext) (actool.Result, error) {
							calls++
							if ctx == nil {
								nilContext = true
								return actool.Result{}, errors.New("nil operation context")
							}
							if tc != nil && tc.ToolUseID != "" {
								fabricatedIdentity = true
							}
							if mode == "handler_error" {
								return actool.Result{}, errors.New("controlled resolved handler error")
							}
							if mode == "tool_error" {
								return actool.Errorf("controlled resolved error result"), nil
							}
							if mode == "nil_tool_context" && tc == nil {
								// Final panic guard may recover this. Neither recovery nor an
								// unavailable optional ToolContext may fall back to raw node data.
								panic("controlled replacement requires optional ToolContext")
							}
							return actool.Text(allowedResult), nil
						},
					})
					out = append(out, replacement)
				}
				return out
			}
			artifactRoot := os.Getenv("R23_ORACLE_OUTPUT")
			if artifactRoot == "" {
				artifactRoot = t.TempDir()
			}
			artifactDir := filepath.Join(artifactRoot, mode)
			if err := os.MkdirAll(artifactDir, 0700); err != nil {
				t.Fatal(err)
			}
			provider := &testPlannerEvidenceBoundaryProvider{withheld: withheld, artifactDir: artifactDir}
			planner := NewPlanner(provider, "controlled-r23-resolved", t.TempDir(), nil, 0, 2)
			planner.SetNonStreaming(func() bool { return true })
			ctx, cancel := context.WithTimeout(locale.WithLang(context.Background(), locale.En), 8*time.Second)
			defer cancel()
			var acts []db.Activity
			met, reason, runErr := planner.Plan(ctx, row.ID, d.Assets(), store, row.Goal, []TriggerEvent{{Kind: "done", IntentID: intent}}, func(a db.Activity) { acts = append(acts, a) })
			receipt := map[string]any{"mode": mode, "database": dbname, "provider_calls": provider.calls, "handler_calls": calls, "resolve_seen": resolveSeen, "raw_sentinel_exposed": provider.leaked, "synthetic_tool_result_in_first_request": provider.syntheticResult, "fabricated_tool_identity": fabricatedIdentity, "nil_operation_context": nilContext, "met": met, "reason": reason, "run_error": fmt.Sprint(runErr), "real_provider_calls": 0}
			defer func() {
				b, _ := json.MarshalIndent(receipt, "", "  ")
				if err := os.WriteFile(filepath.Join(artifactDir, "receipt.json"), b, 0600); err != nil {
					t.Error(err)
				}
			}()
			if !resolveSeen {
				t.Error("actual Planner.Plan did not pass through ToolResolve")
			}
			if runErr != nil {
				t.Errorf("safe omission/restriction failed planner run: %v", runErr)
			}
			if provider.calls != 1 {
				t.Errorf("first-request control provider calls=%d want1", provider.calls)
			}
			if provider.leaked {
				t.Error("RESOLVED_NODE_DETAIL_BYPASS: withheld raw full-only DB sentinel leaked in actual first request")
			}
			if provider.syntheticResult || fabricatedIdentity {
				t.Error("automatic read fabricated a model tool-use/result identity")
			}
			if nilContext {
				t.Error("resolved handler received nil operation context")
			}
			if mode == "dropped" || mode == "read_only_false" || mode == "permission_deny" || mode == "permission_ask" || mode == "wrapper_deny" {
				if calls != 0 {
					t.Errorf("unavailable/unsafe/unpermitted final handler invoked: mode=%s calls=%d", mode, calls)
				}
			}
			if met {
				t.Error("automatic evidence handling marked an unproven goal met")
			}
			g, err := store.GetNode(goal)
			if err != nil || g == nil || g.State != "open" {
				t.Errorf("unproven goal changed: node=%+v err=%v", g, err)
			}
			for _, a := range acts {
				if a.Tool == "prove_goal" {
					t.Error("proof tool receipt invented despite controlled provider returning only text")
				}
			}
			t.Logf("RESOLVED_BOUNDARY: mode=%s handlers=%d provider=%d leaked=%v met=%v", mode, calls, provider.calls, provider.leaked, met)
		})
	}
}

type testPlannerEvidenceDenyWrapper struct {
	actool.CoreTool
	calls *int
}

func (w *testPlannerEvidenceDenyWrapper) CheckPermissions(context.Context, json.RawMessage, permission.Context) permission.Decision {
	return permission.Denied("controlled wrapper denies raw read")
}
func (w *testPlannerEvidenceDenyWrapper) Call(ctx context.Context, in json.RawMessage, tc *actool.ToolContext) (actool.Result, error) {
	*w.calls++
	return w.CoreTool.Call(ctx, in, tc)
}

type testPlannerEvidenceBoundaryProvider struct {
	withheld, artifactDir   string
	calls                   int
	leaked, syntheticResult bool
}

func (p *testPlannerEvidenceBoundaryProvider) check(ctx context.Context, req llm.CompletionRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.calls++
	b, _ := json.MarshalIndent(req, "", "  ")
	if err := os.WriteFile(filepath.Join(p.artifactDir, "first-request.json"), b, 0600); err != nil {
		return err
	}
	p.leaked = strings.Contains(string(b), p.withheld)
	for _, message := range req.Messages {
		for _, block := range message.Content {
			if block.Type == llm.BlockToolResult || block.Type == llm.BlockToolUse {
				p.syntheticResult = true
			}
		}
	}
	return nil
}
func (p *testPlannerEvidenceBoundaryProvider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
	if err := p.check(ctx, req); err != nil {
		return llm.Message{}, "", llm.Usage{}, err
	}
	return llm.Message{Role: llm.RoleAssistant, Content: []llm.ContentBlock{llm.TextBlock("No proof submitted; controlled boundary fixture ends this round only")}}, "end_turn", llm.Usage{InputTokens: 11, OutputTokens: 7, CacheReadTokens: 3, CacheWriteTokens: 2}, nil
}
func (p *testPlannerEvidenceBoundaryProvider) Stream(ctx context.Context, req llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
	return func(yield func(llm.StreamEvent, error) bool) {
		if err := p.check(ctx, req); err != nil {
			yield(llm.StreamEvent{}, err)
			return
		}
		for _, event := range []llm.StreamEvent{{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 11, CacheReadTokens: 3}}, {Type: llm.SETextDelta, Text: "Controlled boundary round only; no proof"}, {Type: llm.SEMessageDelta, StopReason: "end_turn", Usage: llm.Usage{OutputTokens: 7, CacheWriteTokens: 2}}, {Type: llm.SEMessageStop}} {
			if ctx.Err() != nil {
				yield(llm.StreamEvent{}, ctx.Err())
				return
			}
			if !yield(event, nil) {
				return
			}
		}
	}
}
