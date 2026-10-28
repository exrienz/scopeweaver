package server

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Autumn-27/artex/chatstate"
	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/locale"
	"github.com/Autumn-27/norma/llm"
	"github.com/Autumn-27/norma/transcript"
	"os"
	"path/filepath"
)

func (s *Server) ensureLegacyChatHistory(ctx context.Context, ref chatstate.SessionRef) error {
	state, err := s.m.pg.ChatControlState(ctx, ref)
	if err != nil {
		return err
	}
	if state.Active != nil || state.ContextChanging {
		return nil
	}
	rows, err := s.m.pg.ChatLegacyTimeline(ctx, ref)
	if err != nil {
		return err
	}
	var groups [][]db.Activity
	for _, row := range rows {
		if row.RunID != "" {
			break
		}
		if row.Kind == "user" {
			groups = append(groups, nil)
		}
		if len(groups) > 0 {
			groups[len(groups)-1] = append(groups[len(groups)-1], row)
		}
	}
	if len(groups) == 0 {
		return nil
	}
	active, err := s.m.pg.ActiveChatTranscript(ctx, ref)
	if err != nil {
		return err
	}
	store := transcript.NewStore(filepath.Join(s.m.dir, "transcripts"))
	records, err := store.Load(store.MainPath(active.TranscriptID))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	known, err := s.m.pg.ChatHistoryRuns(ctx, ref)
	if err != nil {
		return err
	}
	if len(known) > 0 {
		for i, r := range records {
			if r.UUID == known[0].StartRecordUUID {
				records = records[:i]
				break
			}
		}
	}
	// Compressed or incomplete legacy transcripts are deliberately read-only.
	if err = chatstate.ValidateChatRecords(records); err != nil {
		return nil
	}
	var candidates []transcript.Record
	for _, r := range records {
		if r.Message == nil || r.Message.Role != llm.RoleUser {
			continue
		}
		tool := false
		for _, b := range r.Message.Content {
			if b.Type == llm.BlockToolResult {
				tool = true
			}
		}
		if !tool {
			candidates = append(candidates, r)
		}
	}
	if len(candidates) != len(groups) {
		return nil
	}
	prompts := make([]chatstate.Prompt, len(groups))
	full := make([]string, len(groups))
	for i, g := range groups {
		row := g[0]
		text := row.Detail
		if text == "" {
			text = row.Summary
		}
		prompts[i] = chatstate.Prompt{Text: text}
		actual := candidates[i].Message.Text()
		if actual == text {
			full[i] = text
			continue
		}
		var dto struct {
			Text        string                 `json:"text"`
			Attachments []chatstate.Attachment `json:"attachments"`
		}
		if json.Unmarshal([]byte(text), &dto) != nil || len(dto.Attachments) == 0 {
			return nil
		}
		matched := false
		for _, lang := range locale.Languages() {
			value := composeAgentMessageForLanguage(dto.Text, dto.Attachments, s.chatPromptDir(ref), lang)
			if value == actual {
				full[i] = value
				prompts[i] = chatstate.Prompt{Text: dto.Text, Attachments: dto.Attachments}
				matched = true
				break
			}
		}
		if !matched {
			return nil
		}
	}
	starts, err := chatstate.MapLegacyChatTurns(records, full)
	if err != nil {
		return nil
	}
	turns := make([]db.LegacyChatTurn, len(groups))
	for i, g := range groups {
		if g[len(g)-1].Kind != "result" {
			return nil
		}
		end := ""
		seen := false
		for _, r := range records {
			if i+1 < len(starts) && r.UUID == starts[i+1] {
				break
			}
			if r.UUID == starts[i] {
				seen = true
			}
			if seen && r.Message != nil {
				end = r.UUID
			}
		}
		if end == "" || end == starts[i] {
			return nil
		}
		turns[i] = db.LegacyChatTurn{FirstActivityID: g[0].ID, LastActivityID: g[len(g)-1].ID, StartRecordUUID: starts[i], EndRecordUUID: end, Prompt: prompts[i]}
	}
	err = s.m.pg.ImportLegacyChatTurns(ctx, ref, state.Revision, active.TranscriptID, turns)
	if errors.Is(err, chatstate.ErrConflict) || errors.Is(err, chatstate.ErrBusy) {
		return nil
	}
	return err
}
