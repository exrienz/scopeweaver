package server

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestShutdownWaitsForLogSinkWriter(t *testing.T) {
	m := backportTestManager(t)
	tag := t.Name()
	t.Cleanup(func() {
		if _, err := m.pg.Exec(`DELETE FROM server_logs WHERE tag=$1`, tag); err != nil {
			t.Errorf("cleanup log fixture: %v", err)
		}
	})
	// Use a fresh sink so SetDB's once-only writer belongs to this server.
	previous := logSink
	logSink = &logSinkT{cap: 100, subs: map[chan LogLine]struct{}{}, out: io.Discard}
	defer func() { logSink = previous }()
	sink := logSink
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dir := t.TempDir()
	s := New(ctx, m, dir, dir, dir)
	lock, err := m.pg.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		_ = lock.Rollback()
		drainCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := s.DrainChats(drainCtx); err != nil {
			t.Errorf("cleanup drain: %v", err)
		}
	}()
	if _, err := lock.Exec(`LOCK TABLE server_logs IN ACCESS EXCLUSIVE MODE`); err != nil {
		t.Fatal(err)
	}
	_, _ = sink.Write([]byte("[" + tag + "] in-flight log\n"))
	_, inFlightSeq := sink.recent(0, 100)
	deadline := time.Now().Add(5 * time.Second)
	for {
		var blocked bool
		err := m.pg.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_stat_activity
			WHERE datname=current_database() AND wait_event_type='Lock'
			AND query LIKE 'INSERT INTO server_logs%')`).Scan(&blocked)
		if err != nil {
			t.Fatal(err)
		}
		if blocked {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("logsink writer never entered InsertLog")
		}
		time.Sleep(time.Millisecond)
	}
	_, _ = sink.Write([]byte("[" + tag + "] queued log\n"))
	cancel()
	_, _ = sink.Write([]byte("[" + tag + "] log after cancellation\n"))
	drainCtx, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	err = s.DrainChats(drainCtx)
	stop()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("drain returned with logsink InsertLog still using the database: %v", err)
	}
	if err := lock.Rollback(); err != nil {
		t.Fatal(err)
	}
	drainCtx, stop = context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := s.DrainChats(drainCtx); err != nil {
		t.Fatal(err)
	}
	// Also wait for the in-flight entry on the failing baseline, whose drain
	// returns early, so test cleanup does not close its database mid-insert.
	deadline = time.Now().Add(5 * time.Second)
	for {
		lines, _ := sink.recent(0, 100)
		persisted := false
		for _, line := range lines {
			if line.Seq == inFlightSeq && line.DBID != 0 {
				persisted = true
			}
		}
		if persisted {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("in-flight log did not finish")
		}
		time.Sleep(time.Millisecond)
	}
	var count int
	if err := m.pg.QueryRow(`SELECT count(*) FROM server_logs WHERE tag=$1`, tag).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("persisted %d shutdown-test logs, want only the single in-flight insert", count)
	}
	t.Log("drain waits for an in-flight log insert; queued and later logs stay out of the DB after cancellation")
}
