package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestShutdownWaitsForHTTPWriter(t *testing.T) {
	s := authTestServer(t)
	s.engine = NewEngine(s.m)
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	lock, err := s.m.pg.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	finished := make(chan struct{})
	started := false
	defer func() {
		cancel()
		_ = lock.Rollback()
		if started {
			select {
			case <-finished:
			case <-time.After(5 * time.Second):
				t.Error("HTTP writer did not finish")
			}
		}
		drainCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := s.DrainChats(drainCtx); err != nil {
			t.Error(err)
		}
		_, _ = s.m.pg.Exec(`DELETE FROM task_categories WHERE name=$1`, t.Name())
	}()
	if _, err := lock.Exec(`LOCK TABLE task_categories IN SHARE MODE`); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/task-categories", strings.NewReader(`{"name":"`+t.Name()+`"}`))
	token, err := signJWT(s.jwtKey)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler := s.Handler()
	started = true
	go func() { defer close(finished); handler.ServeHTTP(response, request) }()
	waitForShutdownTaskWrite(t, s.m, "%INSERT INTO task_categories%")
	cancel()
	drainCtx, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	err = s.DrainChats(drainCtx)
	stop()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("drain returned while HTTP handler held a DB write: %v", err)
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
		t.Error("successful drain preceded HTTP writer's commit")
	}
}

func TestShutdownClosesHTTPAndDeleteAdmission(t *testing.T) {
	s := authTestServer(t)
	s.engine = NewEngine(s.m)
	if err := s.DrainChats(t.Context()); err != nil {
		t.Fatal(err)
	}
	if s.engine.BeginDelete("42") {
		s.engine.AbortDelete("42", false)
		t.Error("BeginDelete admitted work after shutdown drain")
	}
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	response := httptest.NewRecorder()
	s.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Errorf("handler admitted request after shutdown: %d", response.Code)
	}
}
