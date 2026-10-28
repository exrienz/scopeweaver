package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/chatstate"
	"github.com/Autumn-27/artex/locale"
	"github.com/Autumn-27/artex/sidequestion"
	"github.com/Autumn-27/norma/llm"
)

func TestChatFailedCompactRemovesUnpublishedTranscript(t *testing.T) {
	f := newHistoryFixture(t)
	for i := 0; i < 4; i++ {
		f.turn(t, fmt.Sprintf("turn-%d ", i)+strings.Repeat("detail ", 100))
	}
	runs, err := f.s.m.pg.ChatHistoryRuns(t.Context(), f.ref)
	if err != nil {
		t.Fatal(err)
	}
	view, err := f.s.m.pg.ChatContext(t.Context(), f.ref)
	if err != nil {
		t.Fatal(err)
	}
	view.Source = "request"
	view.Request = llm.CompletionRequest{System: []string{"system"}, MaxTokens: 100}
	view.Sequence++
	if err := f.s.m.pg.SaveChatContext(t.Context(), f.ref, runs[len(runs)-1].ID, view); err != nil {
		t.Fatal(err)
	}
	state, err := f.s.m.pg.ChatControlState(t.Context(), f.ref)
	if err != nil {
		t.Fatal(err)
	}
	runtime := chatCompactRuntime{Model: sidequestion.Model{Model: "fixture", Identity: "fixture", WindowTokens: 32000}, MaxTokens: 100, Provider: retestProvider{complete: func(context.Context, llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
		_, err := f.s.m.pg.AdmitChatInput(t.Context(), f.ref, "invalidate-compaction", "queue", "", chatstate.Prompt{Text: "later"})
		if err != nil {
			t.Error(err)
		}
		return llm.UserText("summary"), "end_turn", llm.Usage{}, nil
	}}}
	if _, err := f.s.compactChatContext(t.Context(), f.ref, "conflicting-compact", state.Revision, chatstate.CompactOptions{KeepRecent: 1}, runtime); !errors.Is(err, chatstate.ErrConflict) {
		t.Fatal("expected ordinary publication conflict", err)
	}
	waitRetestIdle(t, f.s)
	paths, err := filepath.Glob(filepath.Join(f.s.m.dir, "transcripts", "chat-rev-*.jsonl*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Fatalf("failed compact left unpublished files: %v", paths)
	}
}

func TestChatTranscriptRestartCleanupPreservesReferencedAndUnknownFiles(t *testing.T) {
	f := newHistoryFixture(t)
	prepare := func(ref chatstate.SessionRef) chatstate.TranscriptRevision {
		t.Helper()
		r, err := chatstate.PrepareTranscript(t.Context(), f.store, ref, nil)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	old := prepare(f.ref)
	if err := f.s.m.pg.PublishChatHistory(t.Context(), f.ref, 0, old); err != nil {
		t.Fatal(err)
	}
	current := prepare(f.ref)
	if err := f.s.m.pg.PublishChatHistory(t.Context(), f.ref, 1, current); err != nil {
		t.Fatal(err)
	}
	orphan := prepare(f.ref)
	corrupt := prepare(f.ref)
	if err := os.WriteFile(f.store.MainPath(corrupt.TranscriptID), []byte("unverified user bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	missingManifest := prepare(f.ref)
	if err := os.Remove(f.store.MainPath(missingManifest.TranscriptID) + ".manifest.json"); err != nil {
		t.Fatal(err)
	}
	unknownOwner := prepare(chatstate.SessionRef{ConversationID: 9223372036854775807})
	state, err := f.s.m.pg.ChatControlState(t.Context(), f.ref)
	if err != nil {
		t.Fatal(err)
	}
	target, err := f.s.m.pg.PrepareChatHistoryTarget(t.Context(), f.ref, chatstate.HistoryChange{ClientID: "crashed-child", Kind: "branch", ExpectedRevision: state.Revision}, true, "turn")
	if err != nil {
		t.Fatal(err)
	}
	child := prepare(target)
	if _, err := f.s.m.pg.AdmitChatInput(t.Context(), f.ref, "unknown", "send", "", chatstate.Prompt{Text: "lost run"}); err != nil {
		t.Fatal(err)
	}
	run, err := f.s.m.pg.ClaimNextChatInput(t.Context(), f.ref.GateKey())
	if err != nil || run == nil {
		t.Fatal(run, err)
	}
	unknownRun := prepare(f.ref)
	if _, err := f.s.m.pg.Exec(`UPDATE chat_control_runs SET transcript_id=$2 WHERE id=$1`, run.ID, unknownRun.TranscriptID); err != nil {
		t.Fatal(err)
	}
	if err := f.s.m.pg.FinishChatRun(t.Context(), run.ID, "unknown"); err != nil {
		t.Fatal(err)
	}
	// This is the startup order: cleanup while hidden owners exist, then SQL recovery.
	if err := f.s.recoverChatTranscripts(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.m.pg.RecoverChatPreparations(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, r := range []chatstate.TranscriptRevision{orphan, child} {
		for _, path := range []string{f.store.MainPath(r.TranscriptID), f.store.MainPath(r.TranscriptID) + ".manifest.json"} {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("startup left confirmed unpublished bytes", path, err)
			}
		}
	}
	for _, r := range []chatstate.TranscriptRevision{old, current, corrupt, missingManifest, unknownOwner, unknownRun} {
		if _, err := os.Stat(f.store.MainPath(r.TranscriptID)); err != nil {
			t.Fatal("startup removed referenced or unknown bytes", r.TranscriptID, err)
		}
	}
	uncertain := prepare(f.ref)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := f.s.removeUnpublishedChatTranscript(ctx, uncertain); err == nil {
		t.Fatal("database uncertainty accepted")
	}
	if _, err := os.Stat(f.store.MainPath(uncertain.TranscriptID)); err != nil {
		t.Fatal("database uncertainty removed bytes", err)
	}
}

func TestChatTranscriptRecoveryValidatesStorage(t *testing.T) {
	originalLanguage := locale.ServerDefault()
	defer locale.SetServerDefault(originalLanguage)
	for _, lang := range locale.Languages() {
		t.Run(string(lang), func(t *testing.T) {
			locale.SetServerDefault(lang)
			s := &Server{m: &Manager{dir: t.TempDir()}}
			if err := s.recoverChatTranscripts(t.Context()); err != nil {
				t.Fatalf("missing transcript directory: %v", err)
			}
			root := filepath.Join(s.m.dir, "transcripts")
			fixture := []byte("untranslated user bytes %s 한국어 中文")
			if err := os.WriteFile(root, fixture, 0600); err != nil {
				t.Fatal(err)
			}
			err := s.recoverChatTranscripts(t.Context())
			want := locale.Text(lang, "transcript path is not a directory: %s", root)
			if err == nil || err.Error() != want {
				t.Fatalf("invalid directory error = %v; want %q", err, want)
			}
			data, err := os.ReadFile(root)
			if err != nil || string(data) != string(fixture) {
				t.Fatalf("transcript fixture changed: %q, %v", data, err)
			}
			if err := os.Remove(root); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(root, 0700); err != nil {
				t.Fatal(err)
			}
			if err := s.recoverChatTranscripts(t.Context()); err != nil {
				t.Fatalf("empty transcript directory: %v", err)
			}
		})
	}
}
