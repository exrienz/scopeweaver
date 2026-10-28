package server

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestShutdownDeadlineBoundsAdmissionMutex(t *testing.T) {
	s := &Server{}
	// Chat admission/mutation can hold chatMu while PostgreSQL is blocked.
	// Acquiring the shutdown barrier must also remain inside the caller's budget.
	s.chatMu.Lock()
	locked := true
	finished := make(chan error, 1)
	defer func() {
		if locked {
			s.chatMu.Unlock()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.DrainChats(ctx); err != nil {
			t.Error(err)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	go func() { finished <- s.DrainChats(ctx) }()
	select {
	case err := <-finished:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("drain with held admission mutex: %v", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Error("drain exceeded its deadline waiting for admission mutex")
	}
	s.chatMu.Unlock()
	locked = false
}
