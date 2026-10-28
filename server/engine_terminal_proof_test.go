package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Autumn-27/artex/agent"
	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/internal/dbtest"
	"github.com/Autumn-27/artex/locale"
	"github.com/Autumn-27/norma/llm"
	"github.com/Autumn-27/norma/transcript"
)

const s18ProofToolID = "s18-authentic-proof"
const s18ProviderFailure = "S18 controlled provider failure after proof"

// The first response stays open AFTER the real prove_goal has run. A context-aware
// provider emits usage only afterward; cancel-inside-tool cannot pass this oracle.
type s18Provider struct {
	mode               string
	goalID, evidenceID int64
	calls              atomic.Int64
	started            chan struct{}
	unexpected         chan struct{}
	releaseTail        chan struct{}
	release            sync.Once
}

func (p *s18Provider) Stream(ctx context.Context, _ llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
	return func(yield func(llm.StreamEvent, error) bool) {
		number := p.calls.Add(1)
		if number > 1 {
			if number == 2 {
				close(p.unexpected)
			}
			<-ctx.Done()
			yield(llm.StreamEvent{}, ctx.Err())
			return
		}
		push := func(event llm.StreamEvent) bool {
			if err := ctx.Err(); err != nil {
				yield(llm.StreamEvent{}, err)
				return false
			}
			return yield(event, nil)
		}
		if !push(llm.StreamEvent{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 11, CacheReadTokens: 3}}) {
			return
		}
		close(p.started)
		input, _ := json.Marshal(map[string]any{"goal_id": p.goalID, "evidence_id": p.evidenceID, "reason": "owned S18 fact proves the sole synthetic objective"})
		if !push(llm.StreamEvent{Type: llm.SEToolUseStart, ToolID: s18ProofToolID, ToolName: "prove_goal"}) {
			return
		}
		if !push(llm.StreamEvent{Type: llm.SEToolInputJSON, Text: string(input)}) {
			return
		}
		// A non-JSON event lets Norma finalize/start the tool before the tail arrives.
		if !push(llm.StreamEvent{Type: llm.SEMessageDelta, StopReason: "tool_use"}) {
			return
		}
		select {
		case <-ctx.Done():
			yield(llm.StreamEvent{}, ctx.Err())
			return
		case <-p.releaseTail:
		}
		if !push(llm.StreamEvent{Type: llm.SEMessageDelta, StopReason: "tool_use", Usage: llm.Usage{OutputTokens: 7, CacheWriteTokens: 2}}) {
			return
		}
		if p.mode == "provider_error" {
			yield(llm.StreamEvent{}, errors.New(s18ProviderFailure))
			return
		}
		push(llm.StreamEvent{Type: llm.SEMessageStop})
	}
}

func (p *s18Provider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
	return accumulateStreamForTest(ctx, p.Stream, req)
}

// Reuses the stock fixture schema: no parent fixtures or real providers are modified.
func TestEngineTerminalProofPersistsAuthenticReceipt(t *testing.T) {
	t.Setenv("AGENT_CORE_DISABLE_INTERACTIVE_SHELL", "1")
	priorAugment, priorResolve, priorPrompt, priorTraffic := agent.ToolAugment, agent.ToolResolve, agent.PromptOverride, agent.FindingTrafficBindingEnabled
	agent.ToolAugment, agent.ToolResolve, agent.PromptOverride, agent.FindingTrafficBindingEnabled = nil, nil, nil, nil
	t.Cleanup(func() {
		agent.ToolAugment, agent.ToolResolve, agent.PromptOverride, agent.FindingTrafficBindingEnabled = priorAugment, priorResolve, priorPrompt, priorTraffic
	})
	for _, mode := range []string{"success", "parent_pause", "provider_error"} {
		t.Run(mode, func(t *testing.T) {
			parentDSN := dbtest.DSN(t)
			fixtureDSN, fixtureSchema := proofCreateStockSchema(t, parentDSN)
			pg, err := db.Open(fixtureDSN)
			if err != nil {
				t.Fatal("open owned S18 fixture schema failed")
			}
			defer pg.Close()
			var actual string
			if err := pg.QueryRow("SELECT current_schema()").Scan(&actual); err != nil || actual != fixtureSchema {
				t.Fatal("owned fixture schema identity mismatch")
			}
			row, err := pg.CreateTask("S18 owned synthetic proof", "prove the single stored fact", nil, 0, 600)
			if err != nil {
				t.Fatal(err)
			}
			if err := pg.SetStatus(row.ID, "running"); err != nil {
				t.Fatal(err)
			}
			store := pg.Exploration(row.ExplorationID)
			goal, err := store.AddGoal(map[string]any{"text": "prove the single stored fact"}, "fixture")
			if err != nil {
				t.Fatal(err)
			}
			fact, err := store.AddNode(db.KindFact, map[string]any{"summary": "S18 controlled observation", "detail": "fixed synthetic fact; no external target or inference", "confidence": "observed"}, 0, "confirmed", "fixture", nil)
			if err != nil {
				t.Fatal(err)
			}
			task := &Task{ID: strconv.FormatInt(row.ID, 10), ExpID: row.ExplorationID, Description: row.Description, Goal: row.Goal, Status: "running", PlanHeartbeatSeconds: 600, Store: store, notify: make(chan struct{}, 1)}
			workDir := t.TempDir()
			manager := &Manager{pg: pg, dir: workDir, assets: pg.Assets(), tasks: map[string]*Task{task.ID: task}}
			if err := manager.SetWorkers(1); err != nil {
				t.Fatal(err)
			}
			provider := &s18Provider{mode: mode, goalID: goal, evidenceID: fact, started: make(chan struct{}), unexpected: make(chan struct{}), releaseTail: make(chan struct{})}
			tx := transcript.NewStore(filepath.Join(workDir, "transcripts"))
			planner := agent.NewPlanner(provider, "controlled-s18", workDir, tx, 0, 4)
			worker := agent.NewWorker(provider, "controlled-s18", workDir, nil, 0, 4)
			engine := NewEngine(manager)
			engine.debounce = time.Millisecond
			engine.SetAuthoritativeAgentResolver(func(*Task) (*agent.Planner, *agent.Worker) { return planner, worker })
			parent, cancel := context.WithCancel(locale.WithLang(context.Background(), locale.En))
			control := &Server{m: manager, engine: engine, ctx: parent}
			defer func() {
				cancel()
				stopped := make(chan struct{})
				go func() { engine.StopTask(task.ID); close(stopped) }()
				select {
				case <-stopped:
				case <-time.After(5 * time.Second):
					t.Error("S18 engine cleanup exceeded bound")
				}
			}()
			// This child uses the same execution context as actual worker runs. It must
			// receive the existing GoalMet cancellation only after successful completion.
			workerContext, workerCancel := context.WithCancel(engine.execContextFor(parent, task.ID))
			defer workerCancel()
			engine.Run(parent, task)
			timer := time.NewTimer(4 * time.Second)
			defer timer.Stop()
			tick := time.NewTicker(5 * time.Millisecond)
			defer tick.Stop()
			observedProof, acted, unexpected, timedOut := false, false, false, false
		observe:
			for {
				select {
				case <-provider.unexpected:
					unexpected = true
					break observe
				case <-timer.C:
					timedOut = true
					break observe
				case <-tick.C:
					node, err := store.GetNode(goal)
					if err != nil {
						t.Fatal(err)
					}
					if node != nil && node.State == "met" {
						observedProof = true
						if !acted {
							acted = true
							if mode == "parent_pause" {
								// Actual single/batch user-control path: named engine cancel, then
								// persistent paused flag, as in task_control.go:93-137.
								result, err := control.applyTaskControl(task, "pause")
								if err != nil {
									t.Fatalf("actual user pause failed: %v", err)
								}
								paused, err := pg.GetTask(row.ID)
								if err != nil || paused == nil || !paused.Paused || !result.Paused || result.Status != "paused" {
									t.Fatal("actual user pause did not persist its paused flag/control result")
								}
								t.Logf("actual user pause committed: raw_status=%s paused=%v control_status=%s", paused.Status, paused.Paused, result.Status)
							} else {
								provider.release.Do(func() { close(provider.releaseTail) })
							}
						}
					}
					select {
					case <-provider.started:
						if engine.ActiveLLMCalls(task.ID) == 0 && engine.inflightCount(task.ID) == 0 {
							break observe
						}
					default:
					}
				}
			}
			persisted, err := pg.GetTask(row.ID)
			if err != nil || persisted == nil {
				t.Fatal("read final persisted task failed")
			}
			goalNode, err := store.GetNode(goal)
			if err != nil || goalNode == nil {
				t.Fatal("read persisted goal failed")
			}
			rows, _, err := store.ActivityList(nil, 0, 100)
			if err != nil {
				t.Fatal(err)
			}
			uses, successfulResults, terminalResults := 0, 0, []db.Activity{}
			details := map[int64]string{}
			for _, activity := range rows {
				detail, err := store.ActivityDetail(activity.ID)
				if err != nil {
					t.Fatal(err)
				}
				details[activity.ID] = detail
				usageJSON, _ := json.Marshal(map[string]any{"input": activity.InputTokens, "output": activity.OutputTokens, "cache_read": activity.CacheReadTokens, "cache_write": activity.CacheWriteTokens})
				t.Logf("persisted activity kind=%s tool=%s tool_use_id=%s is_error=%v usage=%s detail=%s", activity.Kind, activity.Tool, activity.ToolUseID, activity.IsError, usageJSON, detail)
				if activity.Kind == "tool_use" && activity.Tool == "prove_goal" && activity.ToolUseID == s18ProofToolID {
					uses++
				}
				if activity.Kind == "tool_result" && activity.Tool == "prove_goal" && activity.ToolUseID == s18ProofToolID && !activity.IsError && strings.Contains(detail, "all task objectives are achieved") && !strings.Contains(detail, "interrupted") {
					successfulResults++
				}
				if activity.Kind == "result" && activity.Worker == "planner" {
					terminalResults = append(terminalResults, activity)
				}
			}
			if !observedProof || goalNode.State != "met" {
				t.Errorf("real final proof never committed: observed=%v state=%s", observedProof, goalNode.State)
			}
			proofEdges, err := store.Edges(100)
			if err != nil {
				t.Fatal(err)
			}
			proves := 0
			for _, edge := range proofEdges {
				if edge.From == fact && edge.To == goal && edge.Rel == db.RelProves {
					proves++
				}
			}
			if proves != 1 {
				t.Errorf("proof edge count=%d want1", proves)
			}
			t.Logf("S18 mode=%s persisted_status=%s goal=%s real_provider_calls=%d unexpected=%v timed_out=%v proof_uses=%d authentic_results=%d terminal_results=%d", mode, persisted.Status, goalNode.State, provider.calls.Load(), unexpected, timedOut, uses, successfulResults, len(terminalResults))
			if mode == "success" {
				if uses != 1 || successfulResults != 1 {
					t.Errorf("successful proof receipt not authentic/paired exactly once: uses=%d success_results=%d", uses, successfulResults)
				}
				if len(terminalResults) != 1 {
					t.Errorf("terminal result count=%d want1", len(terminalResults))
				} else {
					result := terminalResults[0]
					if result.IsError || strings.Contains(details[result.ID], "incomplete") || strings.Contains(details[result.ID], "interrupted") {
						t.Error("terminal success was normalized from an interruption/error")
					}
					if result.InputTokens == nil || *result.InputTokens != 11 || result.OutputTokens == nil || *result.OutputTokens != 7 || result.CacheReadTokens == nil || *result.CacheReadTokens != 3 || result.CacheWriteTokens == nil || *result.CacheWriteTokens != 2 {
						t.Errorf("terminal usage lost late tail: %+v", result)
					}
				}
				total, err := store.TokenTotal()
				if err != nil {
					t.Fatal(err)
				}
				if total.InputTokens != 11 || total.OutputTokens != 7 || total.CacheReadTokens != 3 || total.CacheWriteTokens != 2 {
					t.Errorf("persisted token total lost exact real usage: %+v", total)
				}
				records, err := tx.Load(tx.MainPath(fmt.Sprintf("exp%d-planner", store.ID())))
				if err != nil {
					t.Fatal(err)
				}
				messages, usage := transcript.Messages(records)
				t.Logf("S18 raw transcript usage input=%d output=%d cache_read=%d cache_write=%d", usage.InputTokens, usage.OutputTokens, usage.CacheReadTokens, usage.CacheWriteTokens)
				transcriptUse, transcriptResult := 0, 0
				for _, message := range messages {
					for _, block := range message.Content {
						if block.Type == llm.BlockToolUse && block.ID == s18ProofToolID {
							transcriptUse++
						}
						if block.Type == llm.BlockToolResult && block.ToolUseID == s18ProofToolID && !block.IsError {
							transcriptResult++
						}
					}
				}
				if transcriptUse != 1 || transcriptResult != 1 {
					t.Errorf("raw transcript proof pair missing/duplicated: use=%d result=%d", transcriptUse, transcriptResult)
				}
				if usage.InputTokens != 11 || usage.OutputTokens != 7 || usage.CacheReadTokens != 3 || usage.CacheWriteTokens != 2 {
					t.Errorf("raw transcript usage lost late tail: %+v", usage)
				}
				if unexpected || provider.calls.Load() != 1 {
					t.Error("post-proof upstream inference was attempted instead of local completion")
				}
				if timedOut || persisted.Status != "done" {
					t.Errorf("engine did not promptly persist done: status=%s timeout=%v", persisted.Status, timedOut)
				}
				if code, _, _, ok := agent.AbortReason(workerContext); !ok || code != "goal_met" {
					t.Errorf("existing worker completion cause changed: code=%s ok=%v", code, ok)
				}
			} else {
				if timedOut || unexpected || provider.calls.Load() != 1 {
					t.Errorf("failure/cancellation path leaked or hung: calls=%d", provider.calls.Load())
				}
				// A genuinely proven goal predates this pause. Post-proof task-status
				// precedence is not established here; assert honest run cancellation,
				// not a new engine status policy. Keep the raw/derived status observable.
				if mode == "parent_pause" {
					t.Logf("post-proof pause status observation (precedence unspecified): raw_status=%s paused=%v resolved=%s", persisted.Status, persisted.Paused, control.resolvedTaskStatus(task))
				} else if persisted.Status == "done" || task.lifecycleSnapshot().Status == "done" {
					t.Error("real provider failure was converted to task success")
				}
				if len(terminalResults) != 1 || !terminalResults[0].IsError {
					t.Error("real cancellation/provider failure lost its error terminal receipt")
				} else {
					result := terminalResults[0]
					wantOutput, wantWrite := 7, 2
					if mode == "parent_pause" {
						wantOutput, wantWrite = 0, 0 // tail was never emitted
					}
					if result.InputTokens == nil || *result.InputTokens != 11 || result.OutputTokens == nil || *result.OutputTokens != wantOutput || result.CacheReadTokens == nil || *result.CacheReadTokens != 3 || result.CacheWriteTokens == nil || *result.CacheWriteTokens != wantWrite {
						t.Errorf("failure receipt changed actual emitted usage: %+v", result)
					}
					detail := details[result.ID]
					if mode == "parent_pause" && !strings.Contains(detail, "paused_by_user") {
						t.Error("real parent pause cause lost")
					}
					if mode == "provider_error" && !strings.Contains(detail, s18ProviderFailure) {
						t.Error("real provider error was masked by final goal proof")
					}
				}
			}
		})
	}
}
