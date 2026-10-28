package server

import (
	"context"
	"net/http"
	"time"

	"github.com/Autumn-27/artex/agent"
	"github.com/Autumn-27/artex/locale"
	"github.com/Autumn-27/artex/opencodesdk"
)

// Bridge tokens cannot cross formats or normalized endpoint paths, including SDK-to-SDK edits.
func sdkCredentialCompatible(cfg agent.Config, storedFormat, storedURL string) bool {
	if cfg.Provider() != opencodesdk.Format && storedFormat != opencodesdk.Format {
		return true
	}
	return cfg.Provider() == opencodesdk.Format && storedFormat == opencodesdk.Format && opencodesdk.SameEndpoint(cfg.BaseURL, storedURL)
}
func (s *Server) listOpenCodeSDKModels(w http.ResponseWriter, r *http.Request, cfg agent.Config) {
	fail := func(err error) {
		writeJSON(w, 200, map[string]any{"ok": false, "error": locale.ErrorMessage(responseLanguage(w), err)})
	}
	client, err := cfg.HTTPClient()
	if err != nil {
		fail(err)
		return
	}
	policy := func() bool {
		if cfg.PolicyResolver != nil {
			strict, _ := cfg.PolicyResolver()
			return strict
		}
		return cfg.LocalLLMOnly
	}
	p, err := opencodesdk.New(cfg.BaseURL, cfg.APIKey, "", opencodesdk.Options{Client: client, LocalOnly: policy})
	if err != nil {
		fail(err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	models, err := p.Models(ctx)
	if err != nil {
		fail(err)
		return
	}
	ids := make([]string, 0, len(models))
	for _, m := range models {
		ids = append(ids, m.ID)
	}
	writeJSON(w, 200, map[string]any{"ok": true, "models": ids})
}

// SDK credential scope changes can make the active profile unready; never retain its old authenticated fallback.
func (s *Server) clearUnavailableSDKFallback(cfg agent.Config) {
	p, err := s.m.pg.ActiveProfile()
	if err != nil || p == nil {
		return
	}
	current := agent.ConfigFrom(p.Format, p.Model, p.BaseURL, p.APIKey, p.Proxy)
	if current.Ready() {
		return
	}
	s.cfgMu.Lock()
	clear := cfg.Provider() == opencodesdk.Format || s.llmCfg.Provider() == opencodesdk.Format
	if clear {
		s.llmOn = false
		s.llmProv = nil
		s.llmDirect = nil
		s.chatAgent = nil
		s.llmCfg = cfg
	}
	s.cfgMu.Unlock()
	if clear {
		s.invalidateTaskAgents()
	}
}

// An unavailable persisted SDK profile must not silently become a cloud environment profile on restart.
func (s *Server) environmentLLMFallback() (agent.Config, bool) {
	if s.m != nil && s.m.pg != nil {
		if p, err := s.m.pg.ActiveProfile(); err == nil && p != nil && p.Format == opencodesdk.Format {
			return agent.Config{}, false
		}
	}
	return agent.FromEnv()
}
