package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/llmpool"
	"github.com/Autumn-27/norma/llm"
)

func TestShutdownWaitsForLLMHealthWrites(t *testing.T) {
	for _, clear := range []bool{false, true} {
		name := "save"
		if clear {
			name = "clear"
		}
		t.Run(name, func(t *testing.T) {
			m := backportTestManager(t)
			id, err := m.pg.SaveProfile(&db.LLMProfile{Name: t.Name(), Format: "openai", Model: "fixture"})
			if err != nil {
				t.Fatal(err)
			}
			defer m.pg.DeleteProfile(id)
			ctx, cancel := context.WithCancel(context.Background())
			dir := t.TempDir()
			s := New(ctx, m, dir, dir, dir)
			if clear {
				if err := m.pg.SaveLLMHealth(db.LLMHealth{ProfileID: id, Fails: 1}); err != nil {
					t.Fatal(err)
				}
				s.llmHealth.Restore(id, llmpool.State{Fails: 1})
			}
			lock, err := m.pg.BeginTx(t.Context(), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				cancel()
				_ = lock.Rollback()
				// Baseline callbacks are detached. Keep their database open even when
				// the regression assertion fails before ownership is repaired.
				deadline := time.Now().Add(5 * time.Second)
				for {
					var exists bool
					err := m.pg.QueryRow(`SELECT EXISTS (SELECT 1 FROM llm_profile_health WHERE profile_id=$1)`, id).Scan(&exists)
					if err != nil {
						t.Error(err)
						break
					}
					if exists != clear {
						break
					}
					if time.Now().After(deadline) {
						t.Error("health callback did not finish during cleanup")
						break
					}
					time.Sleep(time.Millisecond)
				}
				drainCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
				defer stop()
				if err := s.DrainChats(drainCtx); err != nil {
					t.Error(err)
				}
			}()
			if _, err := lock.Exec(`LOCK TABLE llm_profile_health IN SHARE MODE`); err != nil {
				t.Fatal(err)
			}
			pattern := "%INSERT INTO llm_profile_health%"
			if clear {
				pattern = "DELETE FROM llm_profile_health%"
				// Pool cancellation also calls Pass; it must own this DELETE.
				cancelled, stop := context.WithCancel(context.Background())
				stop()
				pool := llmpool.New([]*llmpool.Member{{ID: id, Prov: retestProvider{complete: func(context.Context, llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
					return llm.Message{}, "", llm.Usage{}, context.Canceled
				}}}}, s.llmHealth)
				_, _, _, _ = pool.Complete(cancelled, llm.CompletionRequest{})
			} else {
				s.llmHealth.Trip(id, "shutdown health fixture", true)
			}
			waitForShutdownTaskWrite(t, m, pattern)
			cancel()
			drainCtx, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
			err = s.DrainChats(drainCtx)
			stop()
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("drain returned while LLM health %s still held its DB write: %v", name, err)
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
			if err := m.pg.QueryRow(`SELECT EXISTS (SELECT 1 FROM llm_profile_health WHERE profile_id=$1)`, id).Scan(&exists); err != nil {
				t.Fatal(err)
			}
			if exists == clear {
				t.Errorf("successful drain preceded final health %s", name)
			}
		})
	}
}
