package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestShutdownRejectsLateHandlerBackgroundWork(t *testing.T) {
	m := backportTestManager(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dir := t.TempDir()
	s := New(ctx, m, dir, dir, dir)
	body, send := io.Pipe()
	defer body.Close()
	defer send.Close()
	request := httptest.NewRequest(http.MethodPut, "/api/settings", body)
	response := httptest.NewRecorder()
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		// Decode blocks until after drain, like a handler that survives HTTP Close.
		s.putSettings(response, request)
	}()
	cancel()
	drainCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := s.DrainChats(drainCtx); err != nil {
		t.Fatal(err)
	}
	// Any late reconciliation remains blocked here, making admission observable.
	s.concMu.Lock()
	defer s.concMu.Unlock()
	if _, err := io.WriteString(send, `{"task_concurrency_enabled":false}`); err != nil {
		t.Fatal(err)
	}
	select {
	case <-finished:
	case <-drainCtx.Done():
		t.Fatal("settings handler did not finish")
	}
	if response.Code != http.StatusOK {
		t.Fatalf("settings response: %d %s", response.Code, response.Body)
	}
	waited := make(chan struct{})
	go func() { s.backgroundWG.Wait(); close(waited) }()
	select {
	case <-waited:
	case <-time.After(50 * time.Millisecond):
		t.Error("handler registered background reconciliation after drain completed")
	}
	// Release and join even on the failing baseline before manager cleanup.
	s.concMu.Unlock()
	select {
	case <-waited:
	case <-drainCtx.Done():
		t.Error("late reconciliation did not finish")
	}
	s.concMu.Lock()
}

func TestShutdownWaitsForSchedulerInit(t *testing.T) {
	m := backportTestManager(t)
	lock, err := m.pg.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Rollback()
	if _, err = lock.Exec(`LOCK TABLE scheduler_state IN ACCESS EXCLUSIVE MODE`); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dir := t.TempDir()
	s := New(ctx, m, dir, dir, dir)
	defer func() {
		cancel()
		_ = lock.Rollback()
		drainCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := s.DrainChats(drainCtx); err != nil {
			t.Errorf("cleanup drain: %v", err)
		}
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var blocked bool
		err = m.pg.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_stat_activity
			WHERE datname=current_database() AND wait_event_type='Lock'
			AND query LIKE 'SELECT value FROM scheduler_state%')`).Scan(&blocked)
		if err != nil {
			t.Fatal(err)
		}
		if blocked {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("scheduler never entered init")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	drainCtx, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	err = s.DrainChats(drainCtx)
	stop()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("drain returned with scheduler init still using the database: %v", err)
	}
	if err := lock.Rollback(); err != nil {
		t.Fatal(err)
	}
	drainCtx, stop = context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := s.DrainChats(drainCtx); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	t.Log("cancelled during scheduler init: drain waited for DB use before store close")
}

func TestShutdownWaitsForArchiveWorker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{ctx: ctx}
	s.archiveWG.Add(1)
	cancel()
	drainCtx, stop := context.WithTimeout(context.Background(), 20*time.Millisecond)
	err := s.DrainChats(drainCtx)
	stop()
	// Release the admitted worker even on the failing baseline.
	s.archiveWG.Done()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("drain returned before archive worker finished: %v", err)
	}
	drainCtx, stop = context.WithTimeout(context.Background(), time.Second)
	defer stop()
	if err := s.DrainChats(drainCtx); err != nil {
		t.Fatal(err)
	}
}

func TestSchedulerAlreadyCancelledSkipsInit(t *testing.T) {
	m := backportTestManager(t)
	before, err := m.pg.GetSchedState(schedKeyLastFinding)
	if err != nil {
		t.Fatal(err)
	}
	defer m.pg.SetSchedState(schedKeyLastFinding, before)
	if err := m.pg.SetSchedState(schedKeyLastFinding, ""); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	newScheduler(&Server{m: m}).Run(ctx)
	if got, err := m.pg.GetSchedState(schedKeyLastFinding); err != nil || got != "" {
		t.Fatalf("cancelled scheduler initialized state: value=%q err=%v", got, err)
	}
}
