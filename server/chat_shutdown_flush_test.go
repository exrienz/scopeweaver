package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Autumn-27/artex/chatstate"
	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/internal/dbtest"
	"github.com/Autumn-27/artex/locale"
	"github.com/Autumn-27/artex/sidequestion"
	"github.com/Autumn-27/norma/llm"
)

func newShutdownFlushFixture(t *testing.T) (*Server, chatstate.SessionRef, int64) {
	t.Helper()
	s, fid := newRetestServer(t)
	c, err := s.m.pg.CreateConversation("auto", "shutdown final flush", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.m.pg.DeleteConversation(c.ID) })
	ref := chatstate.SessionRef{ConversationID: c.ID}
	if _, err := s.m.pg.ChatControlState(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	s.side = &sideQuestionState{pending: map[string]sidequestion.Snapshot{}, latest: map[string]sidequestion.Snapshot{}, contextPending: map[string]pendingChatContext{}, runs: map[string]sideRun{}, done: make(chan struct{})}
	// All writes below arrive after the periodic publisher has already stopped.
	close(s.side.done)
	snap := sidequestion.Snapshot{Parent: sidequestion.Parent{ConversationID: c.ID}, RunID: 1, Version: 1, Request: llm.CompletionRequest{Messages: []llm.Message{llm.UserText("last checkpoint")}}}
	s.side.pending[snap.Parent.Key()] = snap
	s.side.latest[snap.Parent.Key()] = snap
	if err := s.captureChatContext(t.Context(), ref, "", chatstate.ContextView{Sequence: 1, Used: 123}); err != nil {
		t.Fatal(err)
	}
	return s, ref, fid
}

func TestChatShutdownFinalFlushAfterPublisherStops(t *testing.T) {
	s, ref, fid := newShutdownFlushFixture(t)
	ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := s.DrainChats(ctx); err != nil {
		t.Fatal(err)
	}
	if len(s.side.pending) != 0 || len(s.side.contextPending) != 0 {
		t.Fatal("successful final flush retained pending writes")
	}
	if err := s.m.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := db.Open(dbtest.DSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	defer reopened.DeleteConversation(ref.ConversationID)
	defer reopened.DeleteFinding(fid)
	snap, err := reopened.SideSnapshot(t.Context(), ref.Key())
	if err != nil || snap == nil || snap.Request.Messages[0].Text() != "last checkpoint" {
		t.Fatalf("durable last snapshot: %+v %v", snap, err)
	}
	view, err := reopened.ChatContext(t.Context(), ref)
	if err != nil || view.Sequence != 1 || view.Used != 123 {
		t.Fatalf("durable last context: %+v %v", view, err)
	}
}

func TestChatShutdownFinalFlushFailure(t *testing.T) {
	for _, kind := range []string{"snapshot", "context"} {
		t.Run(kind, func(t *testing.T) {
			s, ref, _ := newShutdownFlushFixture(t)
			table, event := "side_question_sessions", "INSERT OR UPDATE"
			if kind == "context" {
				table, event = "chat_control_sessions", "UPDATE OF context_snapshot"
			}
			if _, err := s.m.pg.Exec(`CREATE FUNCTION shutdown_test_reject() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'owned shutdown final write failure'; END $$`); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _, _ = s.m.pg.Exec(`DROP FUNCTION shutdown_test_reject() CASCADE`) })
			if _, err := s.m.pg.Exec(`CREATE TRIGGER shutdown_test_reject BEFORE ` + event + ` ON ` + table + ` FOR EACH ROW EXECUTE FUNCTION shutdown_test_reject()`); err != nil {
				t.Fatal(err)
			}
			ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			for attempt := 0; attempt < 2; attempt++ {
				err := s.DrainChats(ctx)
				if err == nil || !strings.Contains(err.Error(), "owned shutdown final write failure") {
					t.Fatalf("drain %d hid final %s failure: %v", attempt, kind, err)
				}
			}
			if kind == "snapshot" && len(s.side.pending) != 1 {
				t.Fatal("failed snapshot was discarded")
			}
			if kind == "context" {
				if _, ok := s.side.contextPending[ref.Key()]; !ok {
					t.Fatal("failed context was discarded")
				}
			}
			t.Logf("%s final-write error propagated across repeated drains; failed pending entry retained", kind)
		})
	}
}

func TestChatShutdownFinalFlushIgnoresGoneAndConflict(t *testing.T) {
	for _, kind := range []string{"gone", "conflict"} {
		t.Run(kind, func(t *testing.T) {
			s, ref, _ := newShutdownFlushFixture(t)
			if kind == "gone" {
				if err := s.m.pg.DeleteConversation(ref.ConversationID); err != nil {
					t.Fatal(err)
				}
			} else {
				snap := s.side.pending[ref.Key()]
				snap.HistoryRevision = 99
				s.side.pending[ref.Key()] = snap
				item := s.side.contextPending[ref.Key()]
				item.view.HistoryRevision = 99
				s.side.contextPending[ref.Key()] = item
			}
			ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			if err := s.DrainChats(ctx); err != nil {
				t.Fatal(err)
			}
			if len(s.side.pending) != 0 || len(s.side.contextPending) != 0 {
				t.Fatal("benign obsolete write remained pending")
			}
		})
	}
}

func TestChatShutdownAdmissionTranslationsAndClosingBarrier(t *testing.T) {
	for _, state := range []string{"cancelled", "closing"} {
		s, ref, _ := newShutdownFlushFixture(t)
		setRetestProvider(s, retestProvider{complete: func(context.Context, llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
			t.Fatal("late admission reached provider")
			return llm.Message{}, "", llm.Usage{}, nil
		}})
		if state == "cancelled" {
			ctx, cancel := context.WithCancel(s.ctx)
			s.ctx = ctx
			cancel()
		} else {
			if err := s.DrainChats(t.Context()); err != nil {
				t.Fatal(err)
			}
		}
		for _, lang := range locale.Languages() {
			for _, kind := range []string{"chat", "side"} {
				w := httptest.NewRecorder()
				handler := withLocale(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if kind == "chat" {
						s.admitChatPrompt(w, r, ref, "late", "send", "", chatstate.Prompt{Text: "late"})
					} else {
						s.handleSideQuestions(w, r, sidequestion.Parent{ConversationID: ref.ConversationID})
					}
				}))
				r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"question":"late","client_request_id":"late"}`))
				r.Header.Set("Accept-Language", string(lang))
				handler.ServeHTTP(w, r)
				var body struct {
					Error string `json:"error"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if w.Code != http.StatusServiceUnavailable || body.Error != locale.Text(lang, "server is shutting down") {
					t.Fatalf("%s %s %s: %d %s", state, kind, lang, w.Code, w.Body)
				}
			}
		}
	}
}
