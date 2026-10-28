package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/internal/dbtest"
	"github.com/Autumn-27/artex/locale"
	"github.com/Autumn-27/norma/llm"
	actool "github.com/Autumn-27/norma/tool"
)

// Independent oracle: no reference to candidate-only symbols or helper layout.
// The expected full nodes come from the REAL public node_detail tool, while
// observations come from the REAL Planner.Plan first provider request.
func TestPlannerTriggerEvidenceFirstRequest(t *testing.T) {
	t.Setenv("AGENT_CORE_DISABLE_INTERACTIVE_SHELL", "1")
	oldAugment, oldResolve, oldPrompt, oldTraffic := ToolAugment, ToolResolve, PromptOverride, FindingTrafficBindingEnabled
	ToolAugment, ToolResolve, PromptOverride, FindingTrafficBindingEnabled = nil, nil, nil, nil
	t.Cleanup(func() {
		ToolAugment, ToolResolve, PromptOverride, FindingTrafficBindingEnabled = oldAugment, oldResolve, oldPrompt, oldTraffic
	})
	for _, mode := range []string{"stream", "sync"} {
		t.Run(mode, func(t *testing.T) {
			d, err := db.Open(dbtest.DSN(t))
			if err != nil {
				t.Fatal(err)
			}
			defer d.Close()
			var databaseName string
			if err := d.QueryRow("SELECT current_database()").Scan(&databaseName); err != nil || !strings.HasPrefix(databaseName, "scopeweaver_test_") {
				t.Fatalf("disposable database identity rejected: name=%q err=%v", databaseName, err)
			}
			row, err := d.CreateTask("R23 independent controlled evidence oracle", "Prove the single stored synthetic finding using full recorded observations", nil, 0, 600)
			if err != nil {
				t.Fatal(err)
			}
			defer d.DeleteTask(row.ID)
			ts := d.Exploration(row.ExplorationID)
			asset, err := d.Assets().UpsertRootDomain(db.UpsertRootDomainReq{Domain: "r23-owned-fixture.invalid"})
			if err != nil {
				t.Fatal(err)
			}
			origin, err := ts.AddNode(db.KindFact, map[string]any{"summary": "owned controlled fixture origin"}, 0, "origin", "fixture", nil)
			if err != nil {
				t.Fatal(err)
			}
			goal, err := ts.AddGoal(map[string]any{"text": "Record exactly one LOW finding named SYNTHETIC LAB ONLY public debug marker; recorded observations must support the synthetic-only outcome"}, "fixture")
			if err != nil {
				t.Fatal(err)
			}
			intent, err := ts.AddIntent(map[string]any{"summary": "record the owned synthetic observation; no external request", "asset_ids": []int64{asset}}, 5, []int64{asset}, "fixture")
			if err != nil {
				t.Fatal(err)
			}
			if err := ts.Link(origin, db.RelDerivedFrom, intent); err != nil {
				t.Fatal(err)
			}
			fact, err := ts.AddNode(db.KindFact, map[string]any{
				"summary":    "controlled fixture observation saved",
				"detail":     "R23_FULL_FACT_DETAIL_ONLY: actual controlled fixture literal, not a network/prod observation",
				"evidence":   "R23_FULL_FACT_EVIDENCE_ONLY: fixture value=synthetic-r23-marker",
				"confidence": "observed",
			}, 5, "confirmed", "fixture", []int64{asset})
			if err != nil {
				t.Fatal(err)
			}
			if err := ts.Link(intent, db.RelYields, fact); err != nil {
				t.Fatal(err)
			}
			finding, err := ts.RecordFinding(context.Background(), db.RecordFindingInput{
				TaskID: row.ID, ExplorationID: row.ExplorationID, IntentID: intent,
				VulnClass: "Controlled synthetic debug exposure", Name: "SYNTHETIC LAB ONLY public debug marker",
				Severity: "low", Summary: "controlled synthetic finding saved; full evidence stored separately",
				Evidence: "R23_FULL_FINDING_EVIDENCE_ONLY: controlled fixture value=synthetic-r23-marker; never a credential",
				Worker:   "fixture", AssetIDs: []int64{asset},
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := ts.SetIntentState(intent, "done"); err != nil {
				t.Fatal(err)
			}

			tools := NewToolSet(ts, "planner", locale.En)
			tools.SetTaskID(row.ID)
			var detail actool.CoreTool
			for _, tool := range tools.PlannerTools() {
				if tool.Name() == "node_detail" {
					detail = tool
					break
				}
			}
			if detail == nil {
				t.Fatal("node_detail is absent from baseline catalog")
			}
			expected := make(map[int64]map[string]any)
			for _, id := range []int64{finding.NodeID, fact} {
				result, err := detail.Call(context.Background(), json.RawMessage(fmt.Sprintf("{\"id\":%d}", id)), nil)
				if err != nil || result.IsError {
					t.Fatalf("reference node_detail failed: id=%d err=%v result=%s", id, err, result.Flatten())
				}
				var node map[string]any
				if err := json.Unmarshal([]byte(result.Flatten()), &node); err != nil {
					t.Fatal(err)
				}
				if node["id"] != float64(id) {
					t.Fatalf("reference node identity mismatch: %v", node)
				}
				expected[id] = node
			}
			artifactRoot := os.Getenv("R23_ORACLE_OUTPUT")
			if artifactRoot == "" {
				artifactRoot = t.TempDir()
			}
			artifactDir := filepath.Join(artifactRoot, mode)

			if err := os.MkdirAll(artifactDir, 0700); err != nil {
				t.Fatal(err)
			}
			expectedBytes, _ := json.MarshalIndent(expected, "", "  ")
			if err := os.WriteFile(filepath.Join(artifactDir, "expected-node-detail.json"), expectedBytes, 0600); err != nil {
				t.Fatal(err)
			}
			provider := &testPlannerEvidenceProvider{expected: expected, goal: goal, evidence: finding.NodeID, mode: mode, artifactDir: artifactDir}
			planner := NewPlanner(provider, "controlled-r23-oracle", t.TempDir(), nil, 0, 3)
			planner.SetNonStreaming(func() bool { return mode == "sync" })
			ctx, cancel := context.WithTimeout(locale.WithLang(context.Background(), locale.En), 8*time.Second)
			defer cancel()
			var activities []db.Activity
			met, reason, runErr := planner.Plan(ctx, row.ID, d.Assets(), ts, row.Goal, []TriggerEvent{
				{Kind: "finding", IntentID: intent, Detail: "controlled synthetic finding saved; full evidence stored separately"},
				{Kind: "done", IntentID: intent},
			}, func(a db.Activity) { activities = append(activities, a) })
			receipt := map[string]any{"mode": mode, "database": databaseName, "goal_id": goal, "finding_node_id": finding.NodeID, "fact_node_id": fact, "provider_calls": provider.calls, "missing_full_nodes": provider.missing, "met": met, "reason": reason, "run_error": fmt.Sprint(runErr), "real_provider_calls": 0}
			receipt["stream_calls"] = provider.streamCalls
			receipt["complete_calls"] = provider.completeCalls
			receipt["usage_is_controlled_synthetic"] = true
			defer func() {
				b, _ := json.MarshalIndent(receipt, "", "  ")
				if err := os.WriteFile(filepath.Join(artifactDir, "receipt.json"), b, 0600); err != nil {
					t.Error(err)
				}
			}()
			if mode == "stream" && (provider.streamCalls != 1 || provider.completeCalls != 0) {
				t.Errorf("WRONG_PROVIDER_METHOD: mode=stream Stream=%d Complete=%d", provider.streamCalls, provider.completeCalls)
			}
			if mode == "sync" && (provider.completeCalls != 1 || provider.streamCalls != 0) {
				t.Errorf("WRONG_PROVIDER_METHOD: mode=sync Stream=%d Complete=%d", provider.streamCalls, provider.completeCalls)
			}
			if len(provider.missing) > 0 {
				t.Fatalf("FIRST_REQUEST_MISSING_FULL_EVIDENCE: mode=%s missing_node_detail_ids=%v; real Planner.Plan first request omitted exact persisted full nodes; calls=%d; run_error=%v", mode, provider.missing, provider.calls, runErr)
			}
			if runErr != nil || !met {
				t.Fatalf("real proof did not complete: met=%v reason=%q err=%v", met, reason, runErr)
			}
			if provider.calls != 1 {
				t.Errorf("upstream calls=%d want1 after authentic final proof", provider.calls)
			}
			node, err := ts.GetNode(goal)
			if err != nil || node == nil || node.State != "met" {
				t.Fatalf("goal not durably met: %+v err=%v", node, err)
			}
			edges, err := ts.Edges(100)
			if err != nil {
				t.Fatal(err)
			}
			linked := false
			for _, edge := range edges {
				if edge.From == finding.NodeID && edge.Rel == db.RelProves && edge.To == goal {
					linked = true
				}
			}
			if !linked {
				t.Error("authentic evidence --proves--> goal link missing")
			}
			resultSeen := false
			proofSeen := false
			positiveUsage := false
			for _, a := range activities {
				if a.Kind == "tool_result" && a.Tool == "prove_goal" && !a.IsError {
					proofSeen = true
				}
				if a.Kind == "result" && !a.IsError {
					resultSeen = true
				}
				if a.Kind == "usage" && a.InputTokens != nil && *a.InputTokens > 0 && a.OutputTokens != nil && *a.OutputTokens > 0 && a.CacheReadTokens != nil && *a.CacheReadTokens > 0 && a.CacheWriteTokens != nil && *a.CacheWriteTokens > 0 {
					positiveUsage = true
				}
			}
			receipt["successful_proof_result"] = proofSeen
			receipt["successful_run_result"] = resultSeen
			receipt["all_four_positive_usage"] = positiveUsage
			receipt["proves_edge"] = linked
			if !proofSeen || !resultSeen || !positiveUsage {
				t.Errorf("authentic receipts missing: proof=%v result=%v usage4=%v", proofSeen, resultSeen, positiveUsage)
			}
			if !t.Failed() {
				t.Logf("FIRST_REQUEST_FULL_EVIDENCE_AND_AUTHENTIC_PROOF: mode=%s provider_calls=%d finding=%d fact=%d usage4=%v", mode, provider.calls, finding.NodeID, fact, positiveUsage)
			}
		})
	}
}

type testPlannerEvidenceProvider struct {
	expected          map[int64]map[string]any
	goal, evidence    int64
	mode, artifactDir string
	calls             int
	streamCalls       int
	completeCalls     int
	missing           []int64
}

func (p *testPlannerEvidenceProvider) check(ctx context.Context, req llm.CompletionRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.calls++
	if p.calls > 1 {
		return errors.New("unexpected upstream call after authentic final proof")
	}
	b, _ := json.MarshalIndent(req, "", "  ")
	if err := os.WriteFile(filepath.Join(p.artifactDir, "first-request.json"), b, 0600); err != nil {
		return err
	}
	// Parse nested JSON objects in the actual USER data text. Do not require any
	// candidate helper/heading/layout and do not search system prose for evidence.
	candidates := make(map[int64][]map[string]any)
	for _, msg := range req.Messages {
		if msg.Role != llm.RoleUser {
			continue
		}
		for _, block := range msg.Content {
			if block.Type == llm.BlockText {
				testPlannerEvidenceJSON(block.Text, candidates)
			}
		}
	}
	for _, id := range []int64{p.evidence} {
		if len(candidates[id]) == 0 { /* same comparison below also covers empty */
		}
	}
	for id, want := range p.expected {
		found := false
		for _, got := range candidates[id] {
			if reflect.DeepEqual(got, want) {
				found = true
				break
			}
		}
		if !found {
			p.missing = append(p.missing, id)
		}
	}
	if len(p.missing) > 0 {
		return errors.New("controlled oracle: first request lacks full stored evidence")
	}
	return nil
}
func testPlannerEvidenceJSON(text string, out map[int64][]map[string]any) {
	for i := 0; i < len(text); i++ {
		if text[i] != '{' && text[i] != '[' {
			continue
		}
		var value any
		decoder := json.NewDecoder(strings.NewReader(text[i:]))
		if decoder.Decode(&value) == nil {
			testPlannerEvidenceVisit(value, out)
		}
	}
}
func testPlannerEvidenceVisit(v any, out map[int64][]map[string]any) {
	switch x := v.(type) {
	case map[string]any:
		id, ok := x["id"].(float64)
		kind, _ := x["kind"].(string)
		if ok && (kind == db.KindFact || kind == db.KindFinding) {
			out[int64(id)] = append(out[int64(id)], x)
		}
		for _, child := range x {
			testPlannerEvidenceVisit(child, out)
		}
	case []any:
		for _, child := range x {
			testPlannerEvidenceVisit(child, out)
		}
	}
}
func (p *testPlannerEvidenceProvider) Stream(ctx context.Context, req llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
	p.streamCalls++
	return func(yield func(llm.StreamEvent, error) bool) {
		if err := p.check(ctx, req); err != nil {
			yield(llm.StreamEvent{}, err)
			return
		}
		input, _ := json.Marshal(map[string]any{"goal_id": p.goal, "evidence_id": p.evidence, "reason": "full owned synthetic fixture finding and fact support every goal condition"})
		for _, ev := range []llm.StreamEvent{
			{Type: llm.SEMessageStart, Usage: llm.Usage{InputTokens: 11, CacheReadTokens: 3}},
			{Type: llm.SEToolUseStart, ToolID: "r23-authentic-proof-" + p.mode, ToolName: "prove_goal"},
			{Type: llm.SEToolInputJSON, Text: string(input)},
			{Type: llm.SEMessageDelta, StopReason: "tool_use", Usage: llm.Usage{OutputTokens: 7, CacheWriteTokens: 2}},
			{Type: llm.SEMessageStop},
		} {
			if err := ctx.Err(); err != nil {
				yield(llm.StreamEvent{}, err)
				return
			}
			if !yield(ev, nil) {
				return
			}
		}
	}
}
func (p *testPlannerEvidenceProvider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
	p.completeCalls++
	if err := p.check(ctx, req); err != nil {
		return llm.Message{}, "", llm.Usage{}, err
	}
	input, _ := json.Marshal(map[string]any{"goal_id": p.goal, "evidence_id": p.evidence, "reason": "full owned synthetic fixture finding and fact support every goal condition"})
	return llm.Message{Role: llm.RoleAssistant, Content: []llm.ContentBlock{{Type: llm.BlockToolUse, ID: "r23-authentic-proof-" + p.mode, Name: "prove_goal", Input: input}}}, "tool_use", llm.Usage{InputTokens: 11, OutputTokens: 7, CacheReadTokens: 3, CacheWriteTokens: 2}, nil
}
