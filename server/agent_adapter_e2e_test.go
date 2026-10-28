package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Autumn-27/artex/internal/dbtest"
	"github.com/jackc/pgx/v5"
)

// TestAgentAdapterE2E exercises the stdio MCP process and Pi extension against
// the real HTTP handler and PostgreSQL. Opt-in because it needs npm ci in adapters/agent.
func TestAgentAdapterE2E(t *testing.T) {
	if os.Getenv("SCOPEWEAVER_ADAPTER_E2E") != "1" {
		t.Skip("opt in with SCOPEWEAVER_ADAPTER_E2E=1 and a disposable scopeweaver_adapter_* database")
	}
	// Fail closed: never initialize/migrate the caller's normal application database.
	dbtest.DSN(t) // skip unless ARTEX_TEST_DSN points at a throwaway DB
	config, err := pgx.ParseConfig(os.Getenv("ARTEX_PG_DSN"))
	if err != nil || !strings.HasPrefix(config.Database, "scopeweaver_adapter_") {
		t.Fatal("ARTEX_PG_DSN must explicitly select a disposable scopeweaver_adapter_* database")
	}
	for _, key := range []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "ARTEX_LLM_PROVIDER"} {
		t.Setenv(key, "")
	}
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := m.Close(); err != nil {
			t.Errorf("close fixture manager: %v", err)
		}
	}()
	var database string
	if err := m.pg.QueryRow("SELECT current_database()").Scan(&database); err != nil || !strings.HasPrefix(database, "scopeweaver_adapter_") {
		t.Fatalf("expected isolated database, got %q: %v", database, err)
	}
	// Only a fresh disposable fixture is accepted; no stored providers or external MCP calls.
	var profiles int
	if err := m.pg.QueryRow("SELECT count(*) FROM llm_profiles").Scan(&profiles); err != nil || profiles != 0 {
		t.Fatalf("fixture must have no LLM profiles: count=%d err=%v", profiles, err)
	}
	if _, err := m.pg.Exec("UPDATE mcp_servers SET enabled=false"); err != nil {
		t.Fatal(err)
	}
	findingID, err := m.pg.AddFinding(0, 0, "fixture", "adapter fixture", "high", "SIMULATED adapter finding", "SIMULATED local evidence; no target was tested.", "fixture", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	dir := t.TempDir()
	s := New(ctx, m, dir, dir, dir)
	defer func() {
		cancel()
		// Task cancellation can still write; join it before closing the pool.
		drainCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := s.DrainChats(drainCtx); err != nil {
			t.Errorf("drain fixture server: %v", err)
		}
	}()
	httpServer := httptest.NewServer(s.Handler())
	defer httpServer.Close()
	const password = "scopeweaver-adapter-disposable-test"
	response, err := http.Post(httpServer.URL+"/api/auth/init", "application/json", strings.NewReader(`{"password":"`+password+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("fixture initialization: HTTP %d", response.StatusCode)
	}
	commandCtx, stop := context.WithTimeout(ctx, 60*time.Second)
	defer stop()
	command := exec.CommandContext(commandCtx, "node", "--test", "../adapters/agent/test/e2e.test.js")
	command.Env = append(os.Environ(), "SCOPEWEAVER_E2E_URL="+httpServer.URL,
		"SCOPEWEAVER_E2E_PASSWORD="+password, "SCOPEWEAVER_E2E_FINDING_ID="+strconv.FormatInt(findingID, 10))
	output, err := command.CombinedOutput()
	t.Log(string(output))
	if err != nil {
		t.Fatalf("adapter E2E failed: %v", err)
	}
}
