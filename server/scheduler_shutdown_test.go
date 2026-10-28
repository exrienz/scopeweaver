package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Autumn-27/artex/agent"
	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/internal/dbtest"
)

func TestSchedulerCanceledBeforeRunDoesNotInitialize(t *testing.T) {
	dsn, _ := proofCreateStockSchema(t, dbtest.DSN(t))
	pg, err := db.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pg.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	newScheduler(&Server{m: &Manager{pg: pg}}).Run(ctx)
	var states int
	if err := pg.QueryRow("SELECT count(*) FROM scheduler_state").Scan(&states); err != nil {
		t.Fatal(err)
	}
	if states != 0 {
		t.Fatalf("canceled scheduler initialized %d watermark rows", states)
	}
}

// Hold the real startup scheduler inside SQL past Close's deadline. A timeout
// must leave its pool open; releasing the SQL permits concurrent close retries.
func TestManagerCloseBoundsStalledSchedulerWithoutClosingDatabase(t *testing.T) {
	dsn, _ := proofCreateStockSchema(t, dbtest.DSN(t))
	t.Setenv("ARTEX_PG_DSN", dsn)
	t.Setenv("AGENT_CORE_DISABLE_INTERACTIVE_SHELL", "1")
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if _, err := m.pg.Exec("UPDATE mcp_servers SET enabled=false"); err != nil {
		t.Fatal(err)
	}
	watch, err := db.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer watch.Close()
	tx, err := watch.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	var ownerPID int
	if err := tx.QueryRow("SELECT pg_backend_pid()").Scan(&ownerPID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec("LOCK TABLE scheduler_state IN ACCESS EXCLUSIVE MODE"); err != nil {
		t.Fatal(err)
	}
	priorAugment, priorResolve, priorPrompt, priorTraffic := agent.ToolAugment, agent.ToolResolve, agent.PromptOverride, agent.FindingTrafficBindingEnabled
	defer func() {
		agent.ToolAugment, agent.ToolResolve, agent.PromptOverride, agent.FindingTrafficBindingEnabled = priorAugment, priorResolve, priorPrompt, priorTraffic
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s, err := newServer(ctx, m, m.dir, m.dir, m.dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		// Release SQL even if an assertion fails, before joining other producers.
		_ = tx.Rollback()
		drainCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := s.DrainChats(drainCtx); err != nil {
			t.Errorf("cleanup drain: %v", err)
		}
		s.archiveWG.Wait()
	}()
	blocked := func() bool {
		var waiting bool
		err := watch.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND $1=ANY(pg_blocking_pids(pid)) AND query LIKE '%FROM scheduler_state%')`, ownerPID).Scan(&waiting)
		return err == nil && waiting
	}
	waitFor(t, "real scheduler initialization blocked on held SQL", blocked)
	cancel()
	results := make(chan error, 2)
	started := time.Now()
	for range 2 {
		go func() { results <- m.Close() }()
	}
	for range 2 {
		select {
		case err := <-results:
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("stalled scheduler close = %v, want explicit deadline failure", err)
			}
		case <-time.After(7 * time.Second):
			t.Fatal("stalled scheduler made Close unbounded")
		}
	}
	if elapsed := time.Since(started); elapsed > 7*time.Second {
		t.Fatalf("concurrent Close exceeded bound: %v", elapsed)
	}
	pingCtx, pingCancel := context.WithTimeout(context.Background(), time.Second)
	defer pingCancel()
	if err := m.pg.PingContext(pingCtx); err != nil {
		t.Fatalf("timeout closed the database under its active scheduler: %v", err)
	}
	if !blocked() {
		t.Fatal("held SQL did not remain active past the shutdown deadline")
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		go func() { results <- m.Close() }()
	}
	for range 2 {
		select {
		case err := <-results:
			if err != nil {
				t.Fatalf("close retry after scheduler release: %v", err)
			}
		case <-time.After(7 * time.Second):
			t.Fatal("close retry did not join the released scheduler")
		}
	}
	if err := m.pg.Ping(); err == nil {
		t.Fatal("successful Close left its database pool open")
	}
	var states int
	if err := watch.QueryRow("SELECT count(*) FROM scheduler_state").Scan(&states); err != nil || states != 1 {
		t.Fatalf("canceled scheduler must finish only its in-flight watermark: states=%d err=%v", states, err)
	}
	if value, err := watch.GetSchedState(schedKeyLastFinding); err != nil || value != "0" {
		t.Fatalf("in-flight finding watermark was not completed: value=%q err=%v", value, err)
	}
	t.Log("concurrent Close timed out without closing the DB; held SQL released, only the in-flight watermark persisted, and concurrent retries joined before pool closure")
}
