package server

import (
	"strings"
	"testing"

	"github.com/Autumn-27/artex/agent"
	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/internal/dbtest"
	"github.com/Autumn-27/artex/locale"
)

func reporterSeedFixture(t *testing.T) (*Server, *db.DB) {
	t.Helper()
	dsn, _ := proofCreateStockSchema(t, dbtest.DSN(t))
	pg, err := db.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pg.Close() })
	return &Server{m: &Manager{pg: pg}}, pg
}

func reporterStartupFixture(t *testing.T) (*Server, *db.DB) {
	t.Helper()
	s, pg := reporterSeedFixture(t)
	priorAugment, priorResolve, priorTraffic := agent.ToolAugment, agent.ToolResolve, agent.FindingTrafficBindingEnabled
	t.Cleanup(func() {
		agent.ToolAugment, agent.ToolResolve, agent.FindingTrafficBindingEnabled = priorAugment, priorResolve, priorTraffic
	})
	wireTools(pg, nil)
	wireAgentAugment(pg, t.TempDir(), s.hostTools)
	if err := pg.SetBool(settingAgentTrafficBinding, true); err != nil {
		t.Fatal(err)
	}
	return s, pg
}

func TestReporterStartupRetriesDependentMigrations(t *testing.T) {
	s, pg := reporterStartupFixture(t)
	if _, err := pg.Exec(`ALTER TABLE agents ADD CONSTRAINT reporter_seed_failure CHECK (key <> 'reporter')`); err != nil {
		t.Fatal(err)
	}
	s.seedOrchestrationTools()
	flags := []string{"reporter_agent_seed_v1", "reporter_trigger_evidence_version_v1", "finding_traffic_tools_v1", "finding_workflow_tools_v2_reporter"}
	for _, flag := range flags {
		if value, _, err := pg.GetSetting(flag); err != nil || value == "true" {
			t.Errorf("failed creation completed %s: value=%q err=%v", flag, value, err)
		}
	}
	if _, err := pg.Exec(`ALTER TABLE agents DROP CONSTRAINT reporter_seed_failure`); err != nil {
		t.Fatal(err)
	}
	s.seedOrchestrationTools()
	for _, flag := range flags {
		if value, exists, err := pg.GetSetting(flag); err != nil || !exists || value != "true" {
			t.Errorf("successful startup did not complete %s: value=%q exists=%v err=%v", flag, value, exists, err)
		}
	}
	for _, key := range []string{"get_finding_traffic", "bind_finding_traffic", "update_finding_report", "traffic_search", "traffic_get", "traffic_blob"} {
		row, err := pg.GetTool(key)
		if err != nil || row == nil || !contains(row.Agents, "reporter") {
			t.Errorf("startup retry did not bind %s to reporter: tool=%+v err=%v", key, row, err)
		}
	}
	tools, def, cleanup := agent.AugmentTools(locale.WithLang(t.Context(), locale.En), "reporter", nil)
	defer cleanup()
	if !strings.Contains(def.FindingGuidance, "Automatic traffic association before reporting") {
		t.Fatal("startup retry did not assemble the reporter finding workflow")
	}
	for _, key := range []string{"get_finding_traffic", "bind_finding_traffic", "update_finding_report"} {
		workflowTool(t, tools, key)
	}
}

func TestReporterStartupPreservesCustomizationAndUnbinding(t *testing.T) {
	s, pg := reporterStartupFixture(t)
	a, err := pg.CreateAgent("reporter", "My reporter", "My description")
	if err != nil {
		t.Fatal(err)
	}
	const prompt = "My report template"
	if err := pg.SeedPromptIfEmpty(a.ID, prompt); err != nil {
		t.Fatal(err)
	}
	s.seedOrchestrationTools()
	for _, key := range []string{"get_finding_traffic", "bind_finding_traffic", "traffic_search"} {
		if err := pg.RemoveAgentFromTool("reporter", key); err != nil {
			t.Fatal(err)
		}
	}
	s.seedOrchestrationTools()
	for _, key := range []string{"get_finding_traffic", "bind_finding_traffic", "traffic_search"} {
		row, err := pg.GetTool(key)
		if err != nil || row == nil || contains(row.Agents, "reporter") {
			t.Fatalf("repeated startup undid user unbinding: tool=%s row=%+v err=%v", key, row, err)
		}
	}
	got, err := pg.GetAgentByKey("reporter")
	if err != nil || got == nil || got.ID != a.ID || got.Name != a.Name || got.Description != a.Description {
		t.Fatalf("startup changed custom reporter: agent=%+v err=%v", got, err)
	}
	if got, err := pg.CurrentPrompt(a.ID); err != nil || got != prompt {
		t.Fatalf("startup changed custom reporter prompt: prompt=%q err=%v", got, err)
	}
	if triggers, err := pg.ListTriggersFor("reporter"); err != nil || len(triggers) != 0 {
		t.Fatalf("startup added a trigger to custom reporter: triggers=%+v err=%v", triggers, err)
	}
	if err := pg.DeleteAgent("reporter"); err != nil {
		t.Fatal(err)
	}
	s.seedOrchestrationTools()
	if got, err := pg.GetAgentByKey("reporter"); err != nil || got != nil {
		t.Fatalf("startup recreated user-deleted reporter: agent=%+v err=%v", got, err)
	}
}

func TestReporterStartupPreservesPreviouslyCompletedDeletion(t *testing.T) {
	s, pg := reporterStartupFixture(t)
	if err := pg.SetSetting("reporter_agent_seed_v1", "true"); err != nil {
		t.Fatal(err)
	}
	s.seedOrchestrationTools()
	if got, err := pg.GetAgentByKey("reporter"); err != nil || got != nil {
		t.Fatalf("startup repaired a previously completed user deletion: agent=%+v err=%v", got, err)
	}
	if value, _, err := pg.GetSetting("finding_traffic_tools_v1"); err != nil || value != "true" {
		t.Fatalf("completed deletion blocked existing migrations: value=%q err=%v", value, err)
	}
}

func TestReporterSeedRetriesFailedCreationAndPreservesDeletion(t *testing.T) {
	s, pg := reporterSeedFixture(t)
	const flag = "reporter_agent_seed_v1"
	// Fail the real INSERT without interfering with the settings table.
	if _, err := pg.Exec(`ALTER TABLE agents ADD CONSTRAINT reporter_seed_failure CHECK (key <> 'reporter')`); err != nil {
		t.Fatal(err)
	}
	s.seedReporterAgent()
	if got, exists, err := pg.GetSetting(flag); err != nil || exists {
		t.Fatalf("failed creation marked seed complete: value=%q exists=%v err=%v", got, exists, err)
	}
	if got, err := pg.GetAgentByKey("reporter"); err != nil || got != nil {
		t.Fatalf("injected failure did not prevent creation: agent=%+v err=%v", got, err)
	}
	if _, err := pg.Exec(`ALTER TABLE agents DROP CONSTRAINT reporter_seed_failure`); err != nil {
		t.Fatal(err)
	}
	s.seedReporterAgent()
	a, err := pg.GetAgentByKey("reporter")
	if err != nil || a == nil {
		t.Fatalf("seed did not retry: agent=%+v err=%v", a, err)
	}
	if got, exists, err := pg.GetSetting(flag); err != nil || !exists || got != "true" {
		t.Fatalf("successful retry not recorded: value=%q exists=%v err=%v", got, exists, err)
	}
	prompt, err := pg.CurrentPrompt(a.ID)
	if err != nil || prompt != agent.ReporterDefaultPrompt {
		t.Fatalf("retry did not seed stock prompt: err=%v", err)
	}
	triggers, err := pg.ListTriggersFor("reporter")
	if err != nil || len(triggers) != 1 || triggers[0].AgentKey != "reporter" || triggers[0].ToolCallMessage != reporterToolCallMessage {
		t.Fatalf("retry did not seed finding trigger: triggers=%+v err=%v", triggers, err)
	}
	if err := pg.DeleteAgent("reporter"); err != nil {
		t.Fatal(err)
	}
	s.seedReporterAgent()
	if got, err := pg.GetAgentByKey("reporter"); err != nil || got != nil {
		t.Fatalf("seed recreated a user-deleted reporter: agent=%+v err=%v", got, err)
	}
}

func TestReporterSeedPreservesExistingCustomAgent(t *testing.T) {
	s, pg := reporterSeedFixture(t)
	a, err := pg.CreateAgent("reporter", "My reporter", "My description")
	if err != nil {
		t.Fatal(err)
	}
	const customPrompt = "My custom report template"
	if err := pg.SeedPromptIfEmpty(a.ID, customPrompt); err != nil {
		t.Fatal(err)
	}
	s.seedReporterAgent()
	got, err := pg.GetAgentByKey("reporter")
	if err != nil || got == nil || got.ID != a.ID || got.Name != a.Name || got.Description != a.Description {
		t.Fatalf("seed changed custom reporter: agent=%+v err=%v", got, err)
	}
	prompt, err := pg.CurrentPrompt(got.ID)
	if err != nil || prompt != customPrompt {
		t.Fatalf("seed changed custom prompt: prompt=%q err=%v", prompt, err)
	}
	if triggers, err := pg.ListTriggersFor("reporter"); err != nil || len(triggers) != 0 {
		t.Fatalf("seed added a trigger to a custom agent: triggers=%+v err=%v", triggers, err)
	}
	if value, exists, err := pg.GetSetting("reporter_agent_seed_v1"); err != nil || !exists || value != "true" {
		t.Fatalf("existing reporter not recorded: value=%q exists=%v err=%v", value, exists, err)
	}
	if err := pg.DeleteAgent("reporter"); err != nil {
		t.Fatal(err)
	}
	s.seedReporterAgent()
	if got, err := pg.GetAgentByKey("reporter"); err != nil || got != nil {
		t.Fatalf("seed recreated a user-deleted custom reporter: agent=%+v err=%v", got, err)
	}
}
