package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/internal/dbtest"
)

// R19 lifecycle regressions for the log sink's DB writer. StopDB must detach
// persistence for its EXACT DB, bound the writer's in-flight DB work (cancel),
// and join it so no writer outlives the caller's pg.Close — while the ring,
// stderr passthrough and SSE subscribers keep working, and dbCh is never closed
// (add sends after unlocking, so a close would race an enqueue).

func newTestSink() *logSinkT {
	return &logSinkT{cap: 256, subs: map[chan LogLine]struct{}{}, out: io.Discard}
}

// waitFor polls cond until true or the deadline; fails the test on timeout.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func serverLogsCount(t *testing.T, pg *db.DB) int {
	t.Helper()
	var n int
	if err := pg.QueryRow("SELECT count(*) FROM server_logs").Scan(&n); err != nil {
		t.Fatalf("count server_logs: %v", err)
	}
	return n
}

func serverLogsMarkerCount(t *testing.T, pg *db.DB, marker string) int {
	t.Helper()
	var n int
	if err := pg.QueryRow("SELECT count(*) FROM server_logs WHERE text = $1", marker).Scan(&n); err != nil {
		t.Fatalf("count marker rows: %v", err)
	}
	return n
}

// Without any DB binding StopDB is nil-safe, idempotent and a no-op, and the
// ring/subscribers keep working.
func TestStopDBNilSafeWithoutBinding(t *testing.T) {
	s := newTestSink()
	s.StopDB(nil)
	s.StopDB(nil)
	s.add(LogLine{Level: "info", Text: "ring only"})
	lines, cursor := s.recent(0, 10)
	if len(lines) != 1 || cursor != 1 || lines[0].Text != "ring only" {
		t.Fatalf("ring broken without binding: %v %d", lines, cursor)
	}
	ch, unsub := s.subscribe()
	defer unsub()
	s.add(LogLine{Level: "info", Text: "sub"})
	select {
	case got := <-ch:
		if got.Text != "sub" {
			t.Fatalf("subscriber got %q", got.Text)
		}
	default:
		t.Fatal("subscriber did not receive the line")
	}
}

// Bounded stop, deterministically: the writer's INSERT is blocked on a table
// lock we hold (observed via pg_stat_activity, so no timing guess). StopDB must
// still COMPLETE — its context cancel bounds the stuck insert — and afterwards
// no writer may outlive the stop: new lines reach the ring only, the row count
// stays frozen, and closing pg is safe.
func TestStopDBCompletesWhileInsertBlocked(t *testing.T) {
	dsn := dbtest.DSN(t)
	pg, err := db.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pg.Close()
	watch, err := db.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer watch.Close()

	s := newTestSink()
	s.SetDB(context.Background(), pg)
	s.add(LogLine{Level: "info", Text: "before lock"})
	waitFor(t, "initial insert", func() bool { return serverLogsCount(t, watch) >= 1 })

	tx, err := pg.Begin()
	if err != nil {
		t.Fatal(err)
	}
	// Roll back no matter how this test fails; otherwise the held lock would
	// hang the deferred closes below.
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec("LOCK TABLE server_logs IN ACCESS EXCLUSIVE MODE"); err != nil {
		t.Fatal(err)
	}
	s.add(LogLine{Level: "info", Text: "blocked insert"})
	waitFor(t, "writer blocked on the held lock", func() bool {
		var waiting int
		return watch.QueryRow(
			`SELECT count(*) FROM pg_stat_activity WHERE wait_event_type = 'Lock' AND query LIKE '%server_logs%'`,
		).Scan(&waiting) == nil && waiting >= 1
	})

	done := make(chan struct{})
	go func() {
		s.StopDB(pg)
		close(done)
	}()
	select {
	case <-done:
		// Bounded completion: the cancel unblocked the stuck insert even though
		// we still hold the table lock.
	case <-time.After(5 * time.Second):
		t.Fatal("StopDB did not complete while its insert was blocked: the join is unbounded")
	}

	// StopDB completed while the lock was held. Release it before querying
	// the table, since the observation also needs a conflicting table lock.
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	// No writer outlives the stop: the ring keeps working, persistence is gone.
	before := serverLogsCount(t, watch)
	s.add(LogLine{Level: "info", Text: "after stop"})
	if lines, _ := s.recent(0, 10); len(lines) < 2 {
		t.Fatalf("ring lost lines after stop: %d", len(lines))
	}
	waitFor(t, "ring settled", func() bool { return true })
	if after := serverLogsCount(t, watch); after != before {
		t.Fatalf("rows written after StopDB: before=%d after=%d", before, after)
	}
	s.StopDB(pg) // idempotent
	if err := pg.Close(); err != nil {
		t.Fatalf("closing pg after StopDB failed: %v", err)
	}
}

// Restore runs before the writer starts. StopDB must cancel that SELECT too,
// otherwise it waits forever for a writer that cannot reach its launch.
func TestStopDBCompletesWhileRestoreBlocked(t *testing.T) {
	pg, err := db.Open(dbtest.DSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer pg.Close()
	tx, err := pg.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec("LOCK TABLE server_logs IN ACCESS EXCLUSIVE MODE"); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	s := newTestSink()
	var background sync.WaitGroup
	restored := make(chan struct{})
	go func() {
		defer close(restored)
		s.SetDB(t.Context(), pg, &background)
	}()
	defer func() {
		_ = tx.Rollback()
		<-restored
		s.StopDB(pg)
		background.Wait()
	}()
	waitFor(t, "restore blocked on the held lock", func() bool {
		var blocked bool
		return pg.QueryRow(`SELECT EXISTS (SELECT 1 FROM pg_stat_activity
			WHERE datname=current_database() AND wait_event_type='Lock'
			AND query LIKE 'SELECT id, created_at, level, tag, text%')`).Scan(&blocked) == nil && blocked
	})
	stopped := make(chan struct{})
	go func() {
		s.StopDB(pg)
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("StopDB did not cancel the blocked restore and join its writer")
	}
	<-restored
	background.Wait()
	if _, err := tx.Exec("SELECT 1"); err != nil {
		t.Fatalf("test lock must remain held until StopDB completes: %v", err)
	}
}

// A different manager's DB (a second pool to the same database) must not stop
// the sink bound to the first: persistence continues (verified by a unique
// owned marker, not by global row counts) until the exact owner stops.
func TestStopDBWrongDBDoesNotStopBoundSink(t *testing.T) {
	dsn := dbtest.DSN(t)
	pg, err := db.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pg.Close()
	other, err := db.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()

	s := newTestSink()
	s.SetDB(context.Background(), pg)
	marker := fmt.Sprintf("wrongdb-marker-%d", time.Now().UnixNano())
	s.StopDB(nil)
	s.StopDB(other)
	s.add(LogLine{Level: "info", Text: marker})
	waitFor(t, "marker row persisted after wrong-DB stops", func() bool {
		return serverLogsMarkerCount(t, other, marker) == 1
	})
	s.StopDB(pg)
	s.StopDB(pg)
	s.add(LogLine{Level: "info", Text: marker + "-after"})
	waitFor(t, "ring settled after real stop", func() bool { return true })
	if n := serverLogsMarkerCount(t, other, marker+"-after"); n != 0 {
		t.Fatalf("rows written after the owner's StopDB: %d", n)
	}
}

// An already-canceled context (before SetDB or before StopDB, with a pending
// write) must not break the bounded stop/join; the ring keeps working.
func TestStopDBWithCanceledContextAndPendingWrite(t *testing.T) {
	dsn := dbtest.DSN(t)
	pg, err := db.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pg.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := newTestSink()
	s.SetDB(ctx, pg)
	s.add(LogLine{Level: "info", Text: "canceled ctx"}) // pending write in queue
	done := make(chan struct{})
	go func() {
		s.StopDB(pg)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("StopDB did not complete with a canceled context and a pending write")
	}
	s.add(LogLine{Level: "info", Text: "ring still works"})
	if lines, _ := s.recent(0, 10); len(lines) == 0 {
		t.Fatal("ring stopped working after cancel+stop")
	}
}

// Concurrent same-owner stops must all wait for the same completion (no early
// return bypassing the join), close/cancel happens exactly once, and writers
// racing the stop are safe (dbCh is never closed).
func TestStopDBConcurrentSameOwnerStopsShareJoin(t *testing.T) {
	dsn := dbtest.DSN(t)
	pg, err := db.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pg.Close()

	s := newTestSink()
	s.SetDB(context.Background(), pg)
	const writers, each = 8, 50
	var wg sync.WaitGroup
	stop := make(chan struct{})
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < each; j++ {
				select {
				case <-stop:
					return
				default:
				}
				s.add(LogLine{Level: "info", Text: fmt.Sprintf("w%d-%d", i, j)})
			}
		}(i)
	}
	var stopWG sync.WaitGroup
	for i := 0; i < 8; i++ {
		stopWG.Add(1)
		go func() {
			defer stopWG.Done()
			s.StopDB(pg) // concurrent same-owner stops
		}()
	}
	stopWG.Wait()
	close(stop)
	wg.Wait()
	s.StopDB(pg) // idempotent after concurrent stops

	// Ring invariant under the configured cap: every add bumped seq exactly once
	// (cursor), and the ring holds at most cap lines (truncation is expected —
	// 400 candidate lines into cap 256).
	lines, cursor := s.recent(0, 4096)
	if int64(len(lines)) != cursor && int64(len(lines)) != int64(s.cap) {
		t.Fatalf("ring inconsistent: lines=%d cursor=%d cap=%d", len(lines), cursor, s.cap)
	}
	if int64(len(lines)) > int64(s.cap) {
		t.Fatalf("ring exceeded cap: lines=%d cap=%d", len(lines), s.cap)
	}
}

// After a stop, the stderr passthrough keeps receiving bytes (the terminal log
// stream survives DB persistence shutdown).
func TestStopDBKeepsStderrPassthroughWorking(t *testing.T) {
	dsn := dbtest.DSN(t)
	pg, err := db.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pg.Close()
	var out bytes.Buffer
	s := &logSinkT{cap: 16, subs: map[chan LogLine]struct{}{}, out: &out}
	s.SetDB(context.Background(), pg)
	s.StopDB(pg)
	if _, err := s.Write([]byte("still visible on stderr\n")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "still visible on stderr") {
		t.Fatalf("stderr passthrough broken after stop: %q", out.String())
	}
}
