package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

// The GLM-5.3-Flash E2E receipts (docs/evidence/standalone-cli/flash-low-diagnosis.md,
// candidate-r7-verification.json) showed two successful report_finding calls for the
// same expected finding via two distinct intents (3 and 6). WHY the second intent
// existed — creation round, trigger or author — was NOT captured and remains
// unknown; these tests pin the source-level guidance contracts that keep weak
// models on the intended path without forbidding legitimate fresh validation.

// plannerRoundGoalTailEnglish is the mandatory-minimum tail repeated in every planner
// round input. It must carve out prove_goal precedence so an empty frontier with unmet
// (not yet proven) goals does not force a redundant re-verification dispatch.
func TestPlannerRoundGoalTailCarvesOutProveGoalPrecedence(t *testing.T) {
	for _, marker := range []string{
		"prove_goal",
		"already recorded finding",
		"every condition",
		"fresh validation/retests remain legitimate", // narrowed, not a blanket re-report ban
	} {
		if !strings.Contains(plannerRoundGoalTailEnglish, marker) {
			t.Fatalf("round goal tail missing %q: %q", marker, plannerRoundGoalTailEnglish)
		}
	}
}

// The planner system prompt must not advertise intent IDs as parent_ids: addOneIntent
// rejects any non fact/finding anchor, so "facts/intents/findings" guidance produces
// guaranteed-to-fail calls.
func TestPlannerPromptParentIDsMatchAnchorContract(t *testing.T) {
	if strings.Contains(plannerDefaultTmpl, "facts/intents/findings") {
		t.Fatal("planner prompt still lists intents as acceptable parent_ids; addOneIntent rejects intent/goal/hint anchors")
	}
	idx := strings.LastIndex(plannerDefaultTmpl, "parent_ids:")
	if idx < 0 {
		t.Fatal("planner prompt lost the parent_ids guidance")
	}
	tail := plannerDefaultTmpl[idx:]
	if !strings.Contains(tail, "fact/finding") {
		t.Fatalf("planner prompt parent_ids guidance must state the fact/finding-only contract: %q", tail)
	}
}

// The mandatory-minimum rule inside the system prompt needs the same prove-goal-first
// exception as the round tail, otherwise the wake-on-finding round is still instructed
// to dispatch when the correct action is proving the goal the finding satisfies.
func TestPlannerPromptMinimumDispatchException(t *testing.T) {
	for _, marker := range []string{
		"already recorded finding",
		"prove_goal",
		"EVERY condition", // direct prove_goal requires evidence covering every goal condition
		"explicitly requested fresh validation/retests", // narrowing must not forbid legitimate retests
	} {
		if !strings.Contains(plannerDefaultTmpl, marker) {
			t.Fatalf("planner prompt minimum-dispatch rule missing %q", marker)
		}
	}
}

// Slow-reasoning models burn the task budget re-checking just-dispatched work in the
// same session (R7 Flash/high receipt: get_worker_output returned "(This worker has no
// output yet)" and a follow-up turn concluded "No new intents this round"). The prompt
// must tell the planner to end the round after dispatching.
func TestPlannerPromptEndsRoundAfterDispatch(t *testing.T) {
	for _, marker := range []string{
		"do not re-check just-dispatched work", // keep the no-polling rule
		"no errors map, no zero ids",           // ending requires successful dispatch
		"rejected items",                       // rejected items must be corrected, not dropped
	} {
		if !strings.Contains(plannerDefaultTmpl, marker) {
			t.Fatalf("planner prompt missing dispatch-termination marker %q", marker)
		}
	}
}

// A worker must not copy an already recorded finding from borrowed earlier evidence;
// explicitly requested fresh validation/retests, materially new evidence and distinct
// findings remain legitimate. The R7 Flash/low duplicate (finding 2 via intent 6)
// cited "observed in prior worker WebFetch" — evidence from an earlier run.
func TestWorkerPromptForbidsDuplicateFindingReports(t *testing.T) {
	for _, marker := range []string{
		"solely from borrowed earlier evidence",  // narrowed: only borrowed-evidence re-reports
		"remain legitimate report_finding uses", // fresh validation/retests/distinct findings preserved
	} {
		if !strings.Contains(workerDefaultTmpl, marker) {
			t.Fatalf("worker prompt missing duplicate-report marker %q", marker)
		}
	}
}

// The advertised add_intent batch schema must describe the per-item fields. With
// items as a bare {"type":"object"}, a schema-following model can omit summary (the
// handler rejects it) and has no schema-level statement that parent_ids anchors are
// fact/finding node IDs.
func TestAddIntentBatchSchemaDescribesItems(t *testing.T) {
	var addIntent interface {
		Name() string
		InputSchema() map[string]any
	}
	for _, tool := range NewToolSet(nil, "").PlannerTools() {
		if tool.Name() == "add_intent" {
			addIntent = tool
			break
		}
	}
	if addIntent == nil {
		t.Fatal("add_intent missing from planner tools")
	}
	raw, err := json.Marshal(addIntent.InputSchema())
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Properties struct {
			Intents struct {
				Items map[string]any `json:"items"`
			} `json:"intents"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	items := schema.Properties.Intents.Items
	if items["type"] != "object" {
		t.Fatalf("intents.items type = %v, want object", items["type"])
	}
	props, _ := items["properties"].(map[string]any)
	if props == nil {
		t.Fatalf("intents.items has no properties: %v", items)
	}
	for _, name := range []string{"summary", "asset_ids", "parent_ids", "priority"} {
		if props[name] == nil {
			t.Errorf("intents.items.properties missing %q: %v", name, props)
		}
	}
	var required []string
	switch v := items["required"].(type) {
	case []string:
		required = v
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				required = append(required, s)
			}
		}
	}
	if len(required) != 1 || required[0] != "summary" {
		t.Fatalf("intents.items.required = %v, want exactly [summary] (handler rejects empty summary)", required)
	}
	parentIDs, _ := props["parent_ids"].(map[string]any)
	if parentIDs == nil {
		t.Fatalf("intents.items.properties.parent_ids is not an object: %v", props["parent_ids"])
	}
	if desc, _ := parentIDs["description"].(string); !strings.Contains(desc, "fact/finding") {
		t.Fatalf("intents.items parent_ids description must state the fact/finding-only contract: %v", desc)
	}
}
