package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Autumn-27/artex/agent"
)

func TestShutdownWaitsForTriggeredSetupWrite(t *testing.T) {
	m := backportTestManager(t)
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{ctx: ctx, m: m, engine: NewEngine(m), chatAgent: new(agent.ChatAgent),
		chatBusy: map[string]bool{}, triggerQ: map[string][]triggeredRun{},
		triggerActive: map[string]int{}, triggerCfg: map[string]triggerBehavior{}}
	lock, err := m.pg.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		_ = lock.Rollback()
		// The baseline has no join. Keep its setup continuation alive only in
		// cleanup; the assertions below rely exclusively on DrainChats.
		deadline := time.Now().Add(5 * time.Second)
		for {
			s.queueMu.Lock()
			active := s.triggerActive["assistant"]
			s.queueMu.Unlock()
			if active == 0 {
				break
			}
			if time.Now().After(deadline) {
				t.Error("trigger did not finish during cleanup")
				break
			}
			time.Sleep(time.Millisecond)
		}
		drainCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := s.DrainChats(drainCtx); err != nil {
			t.Error(err)
		}
		_, _ = m.pg.Exec(`DELETE FROM conversations WHERE title=$1`, t.Name())
	}()
	if _, err := lock.Exec(`LOCK TABLE conversations IN SHARE MODE`); err != nil {
		t.Fatal(err)
	}
	s.StartTriggeredRun("assistant", t.Name(), "shutdown trigger fixture", 0, false, "", "")
	waitForShutdownTaskWrite(t, m, "%INSERT INTO conversations%")
	cancel()
	drainCtx, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	err = s.DrainChats(drainCtx)
	stop()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("drain returned while triggered setup held a DB write: %v", err)
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
	if err := m.pg.QueryRow(`SELECT EXISTS (SELECT 1 FROM conversations c JOIN conversation_activities a ON a.conversation_id=c.id WHERE c.title=$1 AND a.kind='user')`, t.Name()).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("successful drain preceded triggered setup's final activity write")
	}
}
