package server

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestShutdownFlushesAfterBackgroundPublisher(t *testing.T) {
	s, ref, _ := newShutdownFlushFixture(t)
	// Begin with no pending SQL, so the only final checkpoint is published by
	// an already-owned producer after the periodic publisher has stopped.
	if err := s.flushSideSnapshots(); err != nil {
		t.Fatal(err)
	}
	if err := s.flushChatContexts(); err != nil {
		t.Fatal(err)
	}
	snap, err := s.m.pg.SideSnapshot(t.Context(), ref.Key())
	if err != nil || snap == nil {
		t.Fatalf("initial checkpoint: %v %v", snap, err)
	}
	snap.Version++
	release := make(chan struct{})
	released := false
	s.backgroundWG.Go(func() {
		<-release
		s.side.mu.Lock()
		s.side.pending[ref.Key()] = *snap
		s.side.latest[ref.Key()] = *snap
		s.side.mu.Unlock()
	})
	defer func() {
		if !released {
			close(release)
		}
		drainCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := s.DrainChats(drainCtx); err != nil {
			t.Error(err)
		}
	}()
	drainCtx, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	err = s.DrainChats(drainCtx)
	stop()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("drain did not wait for publisher: %v", err)
	}
	close(release)
	released = true
	drainCtx, stop = context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := s.DrainChats(drainCtx); err != nil {
		t.Fatal(err)
	}
	final, err := s.m.pg.SideSnapshot(t.Context(), ref.Key())
	if err != nil || final == nil || final.Version != snap.Version {
		t.Fatalf("drain missed final producer checkpoint: got=%v want version=%d err=%v", final, snap.Version, err)
	}
}
