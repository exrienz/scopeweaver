package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Autumn-27/artex/db"
)

func TestShutdownWaitsForEnrichmentWrite(t *testing.T) {
	m := backportTestManager(t)
	probe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<title>shutdown enrichment fixture</title>"))
	}))
	defer probe.Close()
	id, err := m.assets.UpsertHTTPService(db.UpsertHTTPServiceReq{URL: probe.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer m.pg.Exec(`DELETE FROM assets WHERE id=$1`, id)
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{ctx: ctx, m: m, engine: NewEngine(m)}
	lock, err := m.pg.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		_ = lock.Rollback()
		// The baseline Close does not join. Poll only in cleanup so successful
		// drain must itself establish the final persisted state below.
		deadline := time.Now().Add(5 * time.Second)
		for {
			var title string
			err := m.pg.QueryRow(`SELECT COALESCE(page_title,'') FROM assets WHERE id=$1`, id).Scan(&title)
			if err != nil {
				t.Error(err)
				break
			}
			if title == "shutdown enrichment fixture" {
				break
			}
			if time.Now().After(deadline) {
				t.Error("enrichment did not finish during cleanup")
				break
			}
			time.Sleep(time.Millisecond)
		}
		m.enrich.Close()
		drainCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := s.DrainChats(drainCtx); err != nil {
			t.Error(err)
		}
	}()
	if _, err := lock.Exec(`SELECT id FROM assets WHERE id=$1 FOR UPDATE`, id); err != nil {
		t.Fatal(err)
	}
	m.enrich.ProbeSite(id, probe.URL)
	waitForShutdownTaskWrite(t, m, "%INSERT INTO assets%")
	cancel()
	drainCtx, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	err = s.DrainChats(drainCtx)
	stop()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("drain returned while enrichment held a DB write: %v", err)
	}
	if err := lock.Rollback(); err != nil {
		t.Fatal(err)
	}
	drainCtx, stop = context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := s.DrainChats(drainCtx); err != nil {
		t.Fatal(err)
	}
	var title string
	if err := m.pg.QueryRow(`SELECT COALESCE(page_title,'') FROM assets WHERE id=$1`, id).Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != "shutdown enrichment fixture" {
		t.Errorf("successful drain preceded enrichment commit: title=%q", title)
	}
}
