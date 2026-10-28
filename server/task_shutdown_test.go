package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Autumn-27/artex/agent"
)

func waitForShutdownTaskWrite(t *testing.T, m *Manager, queryPattern string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var blocked bool
		err := m.pg.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_stat_activity
			WHERE datname=current_database() AND wait_event_type='Lock'
			AND query LIKE $1)`, queryPattern).Scan(&blocked)
		if err != nil {
			t.Fatal(err)
		}
		if blocked {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("task write never blocked: %s", queryPattern)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestShutdownWaitsForRestoredTaskWrite(t *testing.T) {
	m := backportTestManager(t)
	task, err := m.CreateTask(t.Name(), "preserve cancelled work", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = m.DeleteTask(task.ID, DeleteTaskOptions{}) }()
	intentID, err := task.Store.AddIntent(map[string]any{"summary": "shutdown fixture"}, 1, nil, "planner")
	if err != nil {
		t.Fatal(err)
	}
	// Restore from PostgreSQL, rather than registering a synthetic runtime.
	m.mu.Lock()
	delete(m.tasks, task.ID)
	m.mu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{ctx: ctx, m: m, engine: NewEngine(m)}
	s.engine.SetAuthoritativeAgentResolver(func(*Task) (*agent.Planner, *agent.Worker) {
		return nil, new(agent.Worker)
	})
	lock, err := m.pg.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		_ = lock.Rollback()
		// The baseline drain returns early. Join its runtime explicitly as well,
		// so the red run cannot close the database during worker cleanup.
		s.engine.StopTask(task.ID)
		drainCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := s.DrainChats(drainCtx); err != nil {
			t.Errorf("cleanup drain: %v", err)
		}
	}()
	if _, err := lock.Exec(`SELECT id FROM exploration_nodes WHERE id=$1 FOR UPDATE`, intentID); err != nil {
		t.Fatal(err)
	}
	s.restoreTaskRuntimes()
	waitForShutdownTaskWrite(t, m, "UPDATE exploration_nodes SET state='running'%")
	cancel()
	drainCtx, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	err = s.DrainChats(drainCtx)
	stop()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("drain returned with restored task ClaimIntent still using the database: %v", err)
	}
	if err := lock.Rollback(); err != nil {
		t.Fatal(err)
	}
	drainCtx, stop = context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := s.DrainChats(drainCtx); err != nil {
		t.Fatal(err)
	}
	node, err := task.Store.GetNode(intentID)
	if err != nil || node == nil || node.State != "open" {
		t.Fatalf("cancelled worker did not release its intent: node=%+v err=%v", node, err)
	}
	t.Log("drain times out on the restored worker's blocked claim, then joins its cancellation write after release")
}

func TestShutdownRejectsNewTaskOperations(t *testing.T) {
	s := &Server{engine: NewEngine(nil)}
	if err := s.DrainChats(t.Context()); err != nil {
		t.Fatal(err)
	}
	if s.engine.beginTaskOperation("42") {
		s.engine.decInflight("42")
		t.Fatal("task operation admitted after shutdown drain completed")
	}
}

func TestShutdownWaitsForGoalBootstrap(t *testing.T) {
	m := backportTestManager(t)
	task, err := m.CreateTask(t.Name(), "bootstrap shutdown fixture", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = m.DeleteTask(task.ID, DeleteTaskOptions{}) }()
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{ctx: ctx, m: m, engine: NewEngine(m), taskAgents: map[string]*taskAgentBundle{}}
	lock, err := m.pg.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		_ = lock.Rollback()
		drainCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		// Also join the bootstrap's existing inflight ownership on the baseline.
		if err := s.waitTaskQuiescent(drainCtx, task.ID); err != nil {
			t.Errorf("cleanup bootstrap: %v", err)
		}
		if err := s.DrainChats(drainCtx); err != nil {
			t.Errorf("cleanup drain: %v", err)
		}
	}()
	if _, err := lock.Exec(`LOCK TABLE activity IN ACCESS EXCLUSIVE MODE`); err != nil {
		t.Fatal(err)
	}
	s.startAdmittedTask(task, "bootstrap")
	waitForShutdownTaskWrite(t, m, "%INSERT INTO activity(%")
	cancel()
	drainCtx, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	err = s.DrainChats(drainCtx)
	stop()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("drain returned with goal bootstrap still using the database: %v", err)
	}
	if err := lock.Rollback(); err != nil {
		t.Fatal(err)
	}
	drainCtx, stop = context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := s.DrainChats(drainCtx); err != nil {
		t.Fatal(err)
	}
}

func TestShutdownWaitsForDeadlineWrite(t *testing.T) {
	m := backportTestManager(t)
	task, err := m.CreateTask(t.Name(), "deadline shutdown fixture", nil, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = m.DeleteTask(task.ID, DeleteTaskOptions{}) }()
	if _, err := m.pg.Exec(`UPDATE tasks SET deadline_at=now()-interval '1 second' WHERE id=$1`, task.ID); err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	delete(m.tasks, task.ID)
	m.mu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{ctx: ctx, m: m, engine: NewEngine(m)}
	lock, err := m.pg.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		_ = lock.Rollback()
		s.engine.StopTask(task.ID)
		drainCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := s.DrainChats(drainCtx); err != nil {
			t.Errorf("cleanup drain: %v", err)
		}
	}()
	if _, err := lock.Exec(`SELECT id FROM tasks WHERE id=$1 FOR UPDATE`, task.ID); err != nil {
		t.Fatal(err)
	}
	// Match the restart path for an inactive timed task, which owns a deadline
	// coordinator independently of any planner/worker loops.
	for _, restored := range m.LoadExisting() {
		s.engine.startDeadlineCoordinator(s.ctx, restored)
	}
	deadline := time.Now().Add(5 * time.Second)
	for !s.engine.isSettling(task.ID) {
		if time.Now().After(deadline) {
			t.Fatal("restored deadline coordinator never started settlement")
		}
		time.Sleep(time.Millisecond)
	}
	// With no planner configured, cancellation ends the final-planner wait and
	// settlement proceeds to its guarded terminal write before drain admission closes.
	cancel()
	waitForShutdownTaskWrite(t, m, "%UPDATE tasks%SET status =%")
	drainCtx, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	err = s.DrainChats(drainCtx)
	stop()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("drain returned with deadline settlement still using the database: %v", err)
	}
	if err := lock.Rollback(); err != nil {
		t.Fatal(err)
	}
	drainCtx, stop = context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := s.DrainChats(drainCtx); err != nil {
		t.Fatal(err)
	}
	if status := m.TaskStatus(task.ID); status != "timeout" {
		t.Fatalf("deadline settlement status=%s, want timeout", status)
	}
}

func TestShutdownWaitsForAdmittedTaskOperation(t *testing.T) {
	s := &Server{engine: NewEngine(nil)}
	if !s.engine.beginTaskOperation("42") {
		t.Fatal("initial operation rejected")
	}
	drainCtx, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	err := s.DrainChats(drainCtx)
	stop()
	s.engine.decInflight("42")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("drain returned while an admitted task operation was still active: %v", err)
	}
	drainCtx, stop = context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := s.DrainChats(drainCtx); err != nil {
		t.Fatal(err)
	}
}
