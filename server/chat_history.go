package server

import (
	"context"
	"crypto/sha256"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/Autumn-27/artex/chatstate"
	"github.com/Autumn-27/artex/locale"
	"github.com/Autumn-27/norma/llm"
	"github.com/Autumn-27/norma/transcript"
)

type ChatHistoryResult struct {
	Session         chatstate.SessionRef `json:"session"`
	HistoryRevision int64                `json:"history_revision,string"`
	RunID           string               `json:"run_id,omitempty"`
}

func (s *Server) chatHistoryResult(ctx context.Context, state chatstate.ControlState) ChatHistoryResult {
	result := ChatHistoryResult{Session: state.Session, HistoryRevision: state.HistoryRevision}
	if state.StartInputID != "" {
		if run, err := s.m.pg.ChatRunForInput(ctx, state.StartInputID); err == nil {
			result.RunID = run.ID
		}
	}
	return result
}
func (s *Server) changeChatHistory(ctx context.Context, ref chatstate.SessionRef, change chatstate.HistoryChange) (out ChatHistoryResult, err error) {
	operationCtx, release, beginErr := s.beginChatMutation(ctx, ref)
	if beginErr != nil {
		return out, beginErr
	}
	defer release()
	ctx = operationCtx
	request := change
	defer func() {
		if err != nil {
			if result, e := s.m.pg.ChatHistoryChangeResult(ctx, ref, request); e == nil && result != nil {
				out = s.chatHistoryResult(ctx, *result)
				err = nil
			}
		}
	}()

	if change.Kind != "retry" && change.Kind != "edit" && change.Kind != "branch" {
		return out, locale.Errorf("invalid history operation")
	}
	if result, e := s.m.pg.ChatHistoryChangeResult(ctx, ref, change); e != nil {
		return out, e
	} else if result != nil {
		return s.chatHistoryResult(ctx, *result), nil
	}
	if err = s.ensureLegacyChatHistory(ctx, ref); err != nil {
		return out, err
	}
	state, err := s.m.pg.ChatControlState(ctx, ref)
	if err != nil {
		return out, err
	}
	if state.Active != nil {
		return out, chatstate.ErrBusy
	}
	if state.Revision != change.ExpectedRevision {
		return out, chatstate.ErrConflict
	}
	runs, err := s.m.pg.ChatHistoryRuns(ctx, ref)
	if err != nil {
		return out, err
	}
	if len(runs) == 0 {
		return out, chatstate.ErrAmbiguousHistory
	}
	selected := runs[len(runs)-1]
	if change.TurnID != "" {
		found := false
		for _, run := range runs {
			if run.TurnID == change.TurnID {
				selected = run
				found = true
				break
			}
		}
		if !found {
			return out, chatstate.ErrAmbiguousHistory
		}
	}
	if change.Kind == "retry" && selected.ID != runs[len(runs)-1].ID {
		return out, chatstate.ErrConflict
	}
	if selected.StartRecordUUID == "" {
		return out, chatstate.ErrAmbiguousHistory
	}
	if change.Kind == "branch" && (selected.Status != "completed" || selected.EndRecordUUID == "") {
		return out, chatstate.ErrAmbiguousHistory
	}
	if change.Kind == "edit" {
		if change.Replacement == nil {
			return out, locale.Errorf("replacement is required")
		}
		if err = s.validateChatPrompt(ref, *change.Replacement); err != nil {
			return out, err
		}
	} else if err = s.validateChatPrompt(ref, selected.Prompt, true); err != nil {
		return out, err
	}
	active, err := s.m.pg.ActiveChatTranscript(ctx, ref)
	if err != nil {
		return out, err
	}
	store := transcript.NewStore(filepath.Join(s.m.dir, "transcripts"))
	if active.ID != "" {
		if err = chatstate.ValidateTranscriptPrefix(store, *active); err != nil {
			return out, err
		}
	}
	boundary := selected.StartRecordUUID
	include := false
	if change.Kind == "branch" {
		boundary = selected.EndRecordUUID
		include = true
	}
	records, err := chatstate.LoadChatTranscriptThrough(store, active.TranscriptID, boundary)
	if err != nil {
		return out, err
	}
	end := len(records)
	if !include {
		end--
	}
	prefix := append([]transcript.Record(nil), records[:end]...)
	branch := change.Kind == "branch" || change.Kind == "edit" && selected.ID != runs[len(runs)-1].ID
	target, err := s.m.pg.PrepareChatHistoryTarget(ctx, ref, change, branch, selected.TurnID)
	if err != nil {
		return out, err
	}
	committed := false
	defer func() {
		if !committed {
			if e := s.m.pg.AbandonChatHistoryPreparation(context.Background(), ref, change.ClientID); errors.Is(e, chatstate.ErrGone) {
				_ = s.m.pg.DropPreparedChatOrphan(context.Background(), target)
			}
			if target.ConversationID > 0 && target != ref {
				if exists, e := s.m.pg.ChatConversationExists(context.Background(), target.ConversationID); e == nil && !exists {
					_ = os.RemoveAll(s.chatPromptDir(target))
				}
			}
		}
	}()
	origins, e := chatstate.ChatRecordOrigins(prefix)
	if e != nil {
		return out, e
	}
	copied := map[string]chatstate.Prompt{}
	if target.ConversationID > 0 && target != ref {
		included := map[string]bool{}
		for _, record := range prefix {
			included[record.UUID] = true
		}
		for _, run := range runs {
			if !included[run.StartRecordUUID] {
				continue
			}
			prompt, e := CopyChatAttachments(ctx, s.chatPromptDir(ref), s.chatPromptDir(target), run.Prompt)
			if e != nil {
				return out, e
			}
			copied[run.ID] = prompt
			for index, record := range prefix {
				// Compaction retains new copies of user records in its working set.
				// Rebase the exact generated manifest in both originals and copies.
				if origins[record.UUID] == run.StartRecordUUID && record.Message != nil && record.Message.Role == llm.RoleUser {
					message := *record.Message
					message.Content = append([]llm.ContentBlock(nil), message.Content...)
					for blockIndex, block := range message.Content {
						if block.Type != llm.BlockText {
							continue
						}
						for _, lang := range locale.Languages() {
							old := composeAgentMessageForLanguage("", run.Prompt.Attachments, s.chatPromptDir(ref), lang)
							if old != "" && strings.HasSuffix(block.Text, old) {
								message.Content[blockIndex].Text = strings.TrimSuffix(block.Text, old) + composeAgentMessageForLanguage("", prompt.Attachments, s.chatPromptDir(target), lang)
								break
							}
						}
					}
					prefix[index].Message = &message
				}
			}
		}
		if change.Kind == "edit" && change.Replacement != nil {
			replacement, e := CopyChatAttachments(ctx, s.chatPromptDir(ref), s.chatPromptDir(target), *change.Replacement)
			if e != nil {
				return out, e
			}
			change.Replacement = &replacement
		}
	}
	revision, err := chatstate.PrepareTranscript(ctx, store, target, prefix)
	if err != nil {
		return out, err
	}
	defer s.cleanupPreparedChatTranscript(revision)
	revision.SourceRunID = selected.ID
	revision.ImportedPrompts = copied
	// Use the original request fingerprint; copied replacement manifests are handed
	// through the revision so attachment rebasing does not change idempotency.
	if branch && change.Kind == "edit" && change.Replacement != nil {
		revision.ImportedPrompts[selected.ID] = *change.Replacement
	}
	published, err := s.m.pg.CommitChatHistoryChange(ctx, ref, target, request, revision)
	if err != nil {
		return out, err
	}
	committed = true
	if published.StartInputID != "" {
		s.dispatchChatHistoryInput(published.Session.GateKey(), published.StartInputID)
	}
	return s.chatHistoryResult(ctx, published), nil
}

// CopyChatAttachments validates each source within uploads and publishes each
// destination atomically. Repeated preparation accepts only identical bytes.
func CopyChatAttachments(ctx context.Context, sourceDir, targetDir string, prompt chatstate.Prompt) (chatstate.Prompt, error) {
	out := chatstate.Prompt{Text: prompt.Text, Attachments: append([]chatstate.Attachment(nil), prompt.Attachments...)}
	for index, a := range out.Attachments {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		clean := filepath.ToSlash(filepath.Clean(a.Path))
		if a.Size < 0 || clean != a.Path || !strings.HasPrefix(clean, "uploads/") || filepath.IsAbs(a.Path) || strings.Contains(clean, "../") {
			return out, locale.Errorf("invalid attachment manifest")
		}
		source := filepath.Join(sourceDir, filepath.FromSlash(a.Path))
		resolved, err := filepath.EvalSymlinks(source)
		if err != nil {
			return out, err
		}
		root, err := filepath.EvalSymlinks(filepath.Join(sourceDir, "uploads"))
		if err != nil {
			return out, err
		}
		relative, err := filepath.Rel(root, resolved)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return out, locale.Errorf("attachment outside source")
		}
		if a.Abs != "" && filepath.Clean(a.Abs) != filepath.Clean(source) {
			return out, locale.Errorf("attachment owner mismatch")
		}
		info, err := os.Stat(resolved)
		if err != nil || !info.Mode().IsRegular() || info.Size() != a.Size {
			return out, locale.Errorf("attachment unavailable")
		}
		target := filepath.Join(targetDir, filepath.FromSlash(a.Path))
		if err = prepareChildUploadDirectory(targetDir, a.Path); err != nil {
			return out, err
		}
		if info, e := os.Lstat(target); e == nil && (!info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0) {
			return out, locale.Errorf("invalid child upload")
		} else if e != nil && !errors.Is(e, os.ErrNotExist) {
			return out, e
		}
		// Existing child uploads are immutable under this preparation path.
		bytes, err := os.ReadFile(resolved)
		if err != nil {
			return out, err
		}
		if int64(len(bytes)) != a.Size {
			return out, locale.Errorf("attachment changed while copying")
		}
		if existing, e := os.ReadFile(target); e == nil {
			if sha256.Sum256(existing) != sha256.Sum256(bytes) {
				return out, chatstate.ErrConflict
			}
		} else if !errors.Is(e, os.ErrNotExist) {
			return out, e
		} else {
			f, e := os.CreateTemp(filepath.Dir(target), ".copy-*")
			if e != nil {
				return out, e
			}
			temp := f.Name()
			if _, e = f.Write(bytes); e == nil {
				e = f.Sync()
			}
			closeErr := f.Close()
			if e == nil {
				e = closeErr
			}
			if e == nil {
				e = os.Link(temp, target)
				if errors.Is(e, os.ErrExist) {
					existing, readErr := os.ReadFile(target)
					if readErr == nil && sha256.Sum256(existing) == sha256.Sum256(bytes) {
						e = nil
					}
				}
			}
			os.Remove(temp)
			if e != nil {
				return out, e
			}
			if e = syncChatUploadDirectory(filepath.Dir(target)); e != nil {
				return out, e
			}
		}
		out.Attachments[index].Abs = target
	}
	return out, nil
}
func (s *Server) chatHistoryHTTP(w http.ResponseWriter, r *http.Request, ref chatstate.SessionRef, kind string) {
	var change chatstate.HistoryChange
	if !decodeConversationRequest(w, r, &change) {
		return
	}
	change.Kind = kind
	result, err := s.changeChatHistory(r.Context(), ref, change)
	if err != nil {
		chatControlError(w, err)
		return
	}
	writeJSON(w, 200, result)
}

func (s *Server) chatAttemptsHTTP(w http.ResponseWriter, r *http.Request, ref chatstate.SessionRef, parts []string) {
	if len(parts) == 1 {
		turn := r.URL.Query().Get("turn_id")
		if turn == "" {
			writeErr(w, 400, "turn_id is required")
			return
		}
		attempts, err := s.m.pg.ChatAttempts(r.Context(), ref, turn)
		if err != nil {
			chatControlError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"attempts": attempts})
		return
	}
	if len(parts) == 3 && parts[2] == "activities" {
		before, _ := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
		limit := atoiDefault(r.URL.Query().Get("limit"), 200)
		items, more, err := s.m.pg.ChatAttemptActivities(r.Context(), ref, parts[1], before, limit)
		if err != nil {
			chatControlError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"items": activityDTOs(items), "has_more": more})
		return
	}
	http.NotFound(w, r)
}

func prepareChildUploadDirectory(root, relative string) error {
	if err := os.MkdirAll(root, 0700); err != nil {
		return err
	}
	if err := syncChatUploadDirectory(filepath.Dir(root)); err != nil {
		return err
	}
	current := root
	for _, part := range strings.Split(filepath.ToSlash(filepath.Dir(relative)), "/") {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if err = os.Mkdir(current, 0700); err != nil && !errors.Is(err, os.ErrExist) {
				return err
			}
			info, err = os.Lstat(current)
			if err == nil {
				if e := syncChatUploadDirectory(filepath.Dir(current)); e != nil {
					return e
				}
			}
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return locale.Errorf("invalid child upload directory")
		}
	}
	return nil
}

func syncChatUploadDirectory(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	err = directory.Sync()
	if errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.ENOTSUP) {
		return nil
	}
	return err
}
