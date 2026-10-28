package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Autumn-27/artex/agent"
	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/llmrec"
	"github.com/Autumn-27/artex/locale"
	"github.com/Autumn-27/artex/opencodesdk"
	"github.com/Autumn-27/norma/llm"
)

const sdkCaps = `{"protocol":"scopeweaver-opencode-sdk","version":1,"runtime":"native","sdk_version":"2.1.4","tool_execution":"capture-only","streaming":true}`

func TestSDKErrorAPILocalizesOwnedBridgeDiagnostics(t *testing.T) {
	t.Setenv("SCOPEWEAVER_CONFIG", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("SCOPEWEAVER_LOCAL_LLM_ONLY", "false")
	bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"code":"unauthorized","message":"untrusted bridge-secret evidence"}}`)
	}))
	defer bridge.Close()
	s := &Server{m: &Manager{}}
	for _, h := range []http.HandlerFunc{s.testLLM, s.pgListModels} {
		for lang, want := range map[locale.Lang]string{locale.En: "Bridge authentication required.", locale.Ko: "브리지 인증이 필요합니다.", locale.ZhHans: "需要桥接身份验证。", locale.ZhHant: "需要橋接驗證。", locale.Ja: "ブリッジの認証が必要です。", locale.Es: "Se requiere autenticación del puente."} {
			raw, err := json.Marshal(map[string]any{"provider": opencodesdk.Format, "model": "opencode/test", "base_url": bridge.URL, "api_key": "bridge-secret", "streaming": false})
			if err != nil {
				t.Fatal(err)
			}
			rr := httptest.NewRecorder()
			h(&localeWriter{rr, lang}, httptest.NewRequest("POST", "/", bytes.NewReader(raw)))
			var got struct {
				OK    bool   `json:"ok"`
				Error string `json:"error"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if rr.Code != http.StatusOK || got.OK || !strings.Contains(got.Error, want) || strings.Contains(got.Error, "untrusted") || strings.Contains(got.Error, "bridge-secret") {
				t.Errorf("%s API display=%s, want %q without upstream prose or credentials", lang, rr.Body.String(), want)
			}
		}
	}
}

func sdkServer(t *testing.T, token string, calls *atomic.Int32) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer "+token {
			t.Errorf("unscoped token=%q", r.Header.Get("Authorization"))
			http.Error(w, "wrong key", 401)
			return
		}
		switch r.URL.Path {
		case "/v1/capabilities":
			fmt.Fprint(w, sdkCaps)
		case "/v1/models":
			fmt.Fprint(w, `{"models":[{"id":"opencode/test/model","name":"Fixture"}]}`)
		case "/v1/complete":
			fmt.Fprint(w, `{"message":{"role":"assistant","content":[{"type":"text","text":"OK"}]},"stop_reason":"end_turn","usage":{"input_tokens":96,"output_tokens":13,"cache_read_tokens":5,"cache_write_tokens":7}}`)
		default:
			if r.Method != "DELETE" {
				t.Errorf("unexpected inference path %s", r.URL.Path)
			}
		}
	}))
	t.Cleanup(s.Close)
	return s
}
func callSDKHandler(t *testing.T, h http.HandlerFunc, body map[string]any) map[string]any {
	t.Helper()
	raw, e := json.Marshal(body)
	if e != nil {
		t.Fatal(e)
	}
	rr := httptest.NewRecorder()
	h(rr, httptest.NewRequest("POST", "/", bytes.NewReader(raw)))
	var got map[string]any
	if e = json.Unmarshal(rr.Body.Bytes(), &got); e != nil || rr.Code != 200 {
		t.Fatalf("HTTP %d body=%s decode=%v", rr.Code, rr.Body, e)
	}
	return got
}
func TestSDKDiscoveryAndConnectionScopedCredentials(t *testing.T) {
	t.Setenv("SCOPEWEAVER_CONFIG", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("SCOPEWEAVER_LOCAL_LLM_ONLY", "false")
	for _, source := range []string{"explicit", "same active SDK", "different active SDK", "active cloud"} {
		t.Run(source, func(t *testing.T) {
			var calls atomic.Int32
			bridge := sdkServer(t, "bridge-token", &calls)
			s := &Server{m: &Manager{}, llmCfg: agent.ConfigFrom(opencodesdk.Format, "opencode/test", bridge.URL, "bridge-token", "")}
			body := map[string]any{"provider": opencodesdk.Format, "model": "opencode/test/model", "base_url": bridge.URL, "streaming": false}
			if source == "explicit" {
				body["api_key"] = "bridge-token"
			} else if source == "different active SDK" {
				s.llmCfg.BaseURL = bridge.URL + "/other"
			} else if source == "active cloud" {
				s.llmCfg = agent.ConfigFrom("openai", "m", bridge.URL, "cloud-token", "")
			}
			got := callSDKHandler(t, s.testLLM, body)
			want := source == "explicit" || source == "same active SDK"
			if got["ok"] != want {
				t.Fatalf("connection=%v want ok=%v", got, want)
			}
			if !want && calls.Load() != 0 {
				t.Fatal("wrong-scope credentials reached bridge")
			}
			if source == "explicit" {
				before := calls.Load()
				got = callSDKHandler(t, s.pgListModels, body)
				if got["ok"] != true || got["models"].([]any)[0] != "opencode/test/model" || calls.Load() != before+2 {
					t.Fatalf("models=%v calls=%d before=%d", got, calls.Load(), before)
				}
			}
		})
	}
}
func TestSDKStoredTokenMatchesEndpointAndFormat(t *testing.T) {
	t.Setenv("SCOPEWEAVER_CONFIG", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("SCOPEWEAVER_LOCAL_LLM_ONLY", "false")
	m := backportTestManager(t)
	s := &Server{m: m}
	var calls atomic.Int32
	bridge := sdkServer(t, "stored-token", &calls)
	id, e := m.pg.SaveProfile(&db.LLMProfile{Name: "SDK-scoped-token", Format: opencodesdk.Format, BaseURL: bridge.URL, Model: "opencode/test/model", APIKey: "stored-token", Streaming: false})
	if e != nil {
		t.Fatal(e)
	}
	defer m.pg.DeleteProfile(id)
	for _, handler := range []http.HandlerFunc{s.testLLM, s.pgListModels} {
		for _, scope := range []string{"same", "different path", "different format"} {
			body := map[string]any{"provider": opencodesdk.Format, "base_url": bridge.URL + "/", "model": "opencode/test/model", "profile_id": id, "streaming": false}
			if scope == "different path" {
				body["base_url"] = bridge.URL + "/other"
			}
			if scope == "different format" {
				body["provider"] = "openai"
				body["base_url"] = "https://no-credentials.invalid"
			}
			before := calls.Load()
			got := callSDKHandler(t, handler, body)
			if scope == "same" {
				if got["ok"] != true {
					t.Fatalf("stored token=%v", got)
				}
			} else {
				if got["ok"] != false || calls.Load() != before {
					t.Fatalf("unscoped stored token reuse: %v calls=%d", got, calls.Load()-before)
				}
			}
		}
	}
}

func TestSDKConnectionHTTPHonorsFormAndStoredTokenLimits(t *testing.T) {
	t.Setenv("SCOPEWEAVER_CONFIG", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("SCOPEWEAVER_LOCAL_LLM_ONLY", "false")
	m := backportTestManager(t)
	for _, stream := range []bool{false, true} {
		for _, tc := range []struct {
			name               string
			form, stored, want int
		}{
			{"submitted", 1024, 0, 1024},
			{"stored profile", 0, 1024, 1024},
			{"submitted overrides stored", 512, 2048, 512},
			{"unspecified", 0, 0, 8192},
		} {
			t.Run(fmt.Sprintf("%s/stream=%v", tc.name, stream), func(t *testing.T) {
				var calls, maxTokens atomic.Int32
				bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("Authorization") != "Bearer bridge-token" {
						t.Error("connection request did not use its scoped token")
						http.Error(w, "unauthorized", http.StatusUnauthorized)
						return
					}
					if r.URL.Path == "/v1/capabilities" {
						fmt.Fprint(w, sdkCaps)
						return
					}
					if r.Method == http.MethodDelete {
						return
					}
					var body struct {
						MaxTokens int `json:"max_tokens"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
						http.Error(w, "invalid request", http.StatusBadRequest)
						return
					}
					calls.Add(1)
					maxTokens.Store(int32(body.MaxTokens))
					if stream {
						w.Header().Set("Content-Type", "application/x-ndjson")
						fmt.Fprintln(w, `{"type":"message_start"}`)
						fmt.Fprintln(w, `{"type":"text_delta","text":"OK"}`)
						fmt.Fprintln(w, `{"type":"message_delta","stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1,"cache_read_tokens":0,"cache_write_tokens":0}}`)
						fmt.Fprintln(w, `{"type":"message_stop"}`)
					} else {
						fmt.Fprint(w, `{"message":{"role":"assistant","content":[{"type":"text","text":"OK"}]},"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1,"cache_read_tokens":0,"cache_write_tokens":0}}`)
					}
				}))
				defer bridge.Close()
				s := &Server{m: m}
				body := map[string]any{"provider": opencodesdk.Format, "model": "opencode/test", "base_url": bridge.URL, "streaming": stream}
				if tc.form > 0 {
					body["max_tokens"] = tc.form
				}
				if tc.stored > 0 {
					id, err := m.pg.SaveProfile(&db.LLMProfile{Name: "SDK-connection-token-cap", Format: opencodesdk.Format, BaseURL: bridge.URL, Model: "opencode/test", APIKey: "bridge-token", MaxTokens: tc.stored, Streaming: stream})
					if err != nil {
						t.Fatal(err)
					}
					defer m.pg.DeleteProfile(id)
					body["profile_id"] = id
				} else {
					body["api_key"] = "bridge-token"
				}
				api := httptest.NewServer(http.HandlerFunc(s.testLLM))
				defer api.Close()
				raw, err := json.Marshal(body)
				if err != nil {
					t.Fatal(err)
				}
				response, err := api.Client().Post(api.URL, "application/json", bytes.NewReader(raw))
				if err != nil {
					t.Fatal(err)
				}
				defer response.Body.Close()
				var got struct {
					OK           bool `json:"ok"`
					Reply, Error string
				}
				if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
					t.Fatal(err)
				}
				if response.StatusCode != http.StatusOK || !got.OK || got.Reply != "OK" || calls.Load() != 1 || int(maxTokens.Load()) != tc.want {
					t.Fatalf("HTTP %d result=%+v inference calls=%d max_tokens=%d want 1/%d", response.StatusCode, got, calls.Load(), maxTokens.Load(), tc.want)
				}
			})
		}
	}
}
func TestSDKMetadataLocalOnlyRejectedBeforeBridge(t *testing.T) {
	t.Setenv("SCOPEWEAVER_CONFIG", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("SCOPEWEAVER_LOCAL_LLM_ONLY", "true")
	var calls atomic.Int32
	b := sdkServer(t, "token", &calls)
	s := &Server{m: &Manager{}}
	got := callSDKHandler(t, s.pgListModels, map[string]any{"provider": opencodesdk.Format, "base_url": b.URL, "api_key": "token"})
	if got["ok"] != false || !strings.Contains(got["error"].(string), "local_llm_only") || calls.Load() != 0 {
		t.Fatalf("policy=%v calls=%d", got, calls.Load())
	}
}

func TestSDKUsageCanonicalLedger(t *testing.T) {
	t.Setenv("SCOPEWEAVER_CONFIG", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("SCOPEWEAVER_LOCAL_LLM_ONLY", "false")
	m := backportTestManager(t)
	var calls atomic.Int32
	b := sdkServer(t, "token", &calls)
	cfg := agent.ConfigFrom(opencodesdk.Format, "opencode/cache-fixture", b.URL, "token", "")
	p, err := cfg.NewProvider()
	if err != nil {
		t.Fatal(err)
	}
	ctx := llmrec.WithTaskID(t.Context(), "sdk-cache-canonical")
	defer m.pg.Exec(`DELETE FROM llm_usage WHERE task_id=$1`, "sdk-cache-canonical")
	defer m.pg.Exec(`DELETE FROM llm_records WHERE task_id=$1`, "sdk-cache-canonical")
	recorded := llmrec.Wrap(p, m.pg, cfg.Model, "sdk-cache-fixture", "", "", func() bool { return false })
	_, _, usage, err := recorded.Complete(ctx, llm.CompletionRequest{Messages: []llm.Message{llm.UserText("ping")}, MaxTokens: 64})
	if err != nil {
		t.Fatal(err)
	}
	if usage.InputTokens != 108 || usage.CacheReadTokens != 5 || usage.CacheWriteTokens != 7 || usage.OutputTokens != 13 {
		t.Fatalf("canonical usage=%+v", usage)
	}
	stats, err := m.pg.TokenByModel("sdk-cache-canonical")
	if err != nil || len(stats) != 1 || stats[0].InputTokens != 108 || stats[0].CacheReadTokens != 5 || stats[0].CacheWriteTokens != 7 {
		t.Fatalf("ledger=%+v err=%v", stats, err)
	}
}

func TestSDKQuotaOnlyAdvancesTaskChain(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, tc := range []struct {
			code   string
			status int
			quota  bool
		}{{"quota_exhausted", 402, true}, {"rate_limit", 429, false}, {"sdk_failure", 503, false}, {"quota_exhausted", 401, false}, {"sdk_failure", 402, false}, {"free_tier_restricted", 403, false}} {
			t.Run(fmt.Sprintf("%v/%s/%d", stream, tc.code, tc.status), func(t *testing.T) {
				first := &scriptedLLMProvider{err: &opencodesdk.BridgeError{Code: tc.code, Status: tc.status, Message: "quota exceeded"}}
				second := &scriptedLLMProvider{events: []llm.StreamEvent{{Type: llm.SEMessageStart}, {Type: llm.SETextDelta, Text: "backup"}, {Type: llm.SEMessageDelta, StopReason: "end_turn"}, {Type: llm.SEMessageStop}}}
				active := int64(1)
				rotations := 0
				hooks := taskLLMStreamHooks{current: func() (taskLLMSelection, error) {
					p := first
					if active == 2 {
						p = second
					}
					return taskLLMSelection{profileID: active, provider: p, retry: agent.RetryConfig{StreamAttempts: -1}}, nil
				}, exhaust: func(taskLLMSelection, error) (db.TaskLLMTransition, error) {
					rotations++
					active = 2
					return db.TaskLLMTransition{Advanced: true, NextProfileID: &active}, nil
				}}
				var err error
				if stream {
					_, err = collectTaskLLMStream(streamTaskLLM(t.Context(), "sdk-quota", llm.CompletionRequest{}, hooks))
				} else {
					_, _, _, err = completeTaskLLM(t.Context(), "sdk-quota", llm.CompletionRequest{}, hooks)
				}
				if tc.quota {
					if err != nil || rotations != 1 || second.calls != 1 {
						t.Fatalf("quota failover err=%v rotations=%d backup=%d", err, rotations, second.calls)
					}
				} else if err == nil || rotations != 0 || second.calls != 0 {
					t.Fatalf("nonquota moved profile: err=%v rotations=%d backup=%d", err, rotations, second.calls)
				}
			})
		}
	}
}
func TestSDKModelRequiredBeforeSaveAndActivation(t *testing.T) {
	t.Setenv("SCOPEWEAVER_CONFIG", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("SCOPEWEAVER_LOCAL_LLM_ONLY", "false")
	m := backportTestManager(t)
	s := &Server{m: m}
	before, err := m.pg.ListProfiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, model := range []string{"", "model", "p/", "/m", "p//m"} {
		raw, _ := json.Marshal(map[string]any{"name": "sdk-invalid-model", "format": opencodesdk.Format, "model": model, "base_url": "http://127.0.0.1:4097", "api_key": "token"})
		rr := httptest.NewRecorder()
		s.pgSaveProfile(rr, httptest.NewRequest("POST", "/", bytes.NewReader(raw)))
		if rr.Code != 400 {
			t.Fatalf("invalid model %q save HTTP %d: %s", model, rr.Code, rr.Body)
		}
	}
	after, err := m.pg.ListProfiles()
	if err != nil || len(after) != len(before) {
		t.Fatalf("invalid profiles stored before=%d after=%d err=%v", len(before), len(after), err)
	}
	id, err := m.pg.SaveProfile(&db.LLMProfile{Name: "sdk-legacy-invalid-model", Format: opencodesdk.Format, Model: "opencode/test", BaseURL: "http://127.0.0.1:4097", APIKey: "token"})
	if err != nil {
		t.Fatal(err)
	}
	defer m.pg.DeleteProfile(id)
	if _, err = m.pg.Exec(`UPDATE llm_profiles SET model='' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	activeBefore, err := m.pg.ActiveProfile()
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"id": id})
	rr := httptest.NewRecorder()
	s.pgActivateProfile(rr, httptest.NewRequest("POST", "/", bytes.NewReader(raw)))
	if rr.Code != 400 {
		t.Fatalf("activation HTTP %d: %s", rr.Code, rr.Body)
	}
	activeAfter, err := m.pg.ActiveProfile()
	if err != nil {
		t.Fatal(err)
	}
	if (activeBefore == nil) != (activeAfter == nil) || (activeBefore != nil && activeBefore.ID != activeAfter.ID) {
		t.Fatal("invalid SDK model changed active profile")
	}
}

func TestSDKActiveTokenClearDisablesCachedFallback(t *testing.T) {
	t.Setenv("SCOPEWEAVER_CONFIG", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("SCOPEWEAVER_LOCAL_LLM_ONLY", "false")
	m := backportTestManager(t)
	previous, err := m.pg.ActiveProfile()
	if err != nil {
		t.Fatal(err)
	}
	p := &db.LLMProfile{Name: "sdk-active-token-clear", Format: opencodesdk.Format, BaseURL: "http://127.0.0.1:4097/old", Model: "opencode/test", APIKey: "old-token"}
	id, err := m.pg.SaveProfile(p)
	if err != nil {
		t.Fatal(err)
	}
	p.ID = id
	defer func() {
		if previous != nil {
			_ = m.pg.SetActiveProfile(previous.ID)
		} else {
			_, _ = m.pg.Exec(`UPDATE llm_profiles SET is_default=false WHERE id=$1`, id)
		}
		_ = m.pg.DeleteProfile(id)
	}()
	if err = m.pg.SetActiveProfile(id); err != nil {
		t.Fatal(err)
	}
	fake := &scriptedLLMProvider{}
	s := &Server{m: m, llmOn: true, llmCfg: agent.ConfigFrom(opencodesdk.Format, p.Model, p.BaseURL, p.APIKey, ""), llmProv: fake, llmDirect: fake}
	p.BaseURL = "http://127.0.0.1:4097/new"
	p.APIKey = ""
	if _, err = m.pg.SaveProfile(p); err != nil {
		t.Fatal(err)
	}
	s.reapplyActiveProfile()
	if _, _, ok := s.globalProvider(); ok {
		t.Fatal("old SDK token remains available through cached global provider after scope change")
	}
	if s.llmCfg.APIKey != "" || s.llmCfg.BaseURL != p.BaseURL {
		t.Fatalf("stale SDK config=%+v", s.llmCfg)
	}
}

func TestSDKUnavailableProfileBlocksCloudEnvironmentFallback(t *testing.T) {
	t.Setenv("SCOPEWEAVER_CONFIG", filepath.Join(t.TempDir(), "missing.json"))
	t.Setenv("SCOPEWEAVER_LOCAL_LLM_ONLY", "false")
	t.Setenv("SCOPEWEAVER_LLM_PROVIDER", "openai")
	t.Setenv("SCOPEWEAVER_LLM_MODEL", "environment-model")
	t.Setenv("SCOPEWEAVER_LLM_API_KEY", "unrelated-cloud-key")
	t.Setenv("SCOPEWEAVER_LLM_BASE_URL", "http://127.0.0.1:1")
	m := backportTestManager(t)
	s := &Server{m: m}
	previous, err := m.pg.ActiveProfile()
	if err != nil {
		t.Fatal(err)
	}
	p := &db.LLMProfile{Name: "sdk-block-environment-fallback", Format: opencodesdk.Format, Model: "opencode/test", BaseURL: "http://127.0.0.1:4097"}
	id, err := m.pg.SaveProfile(p)
	if err != nil {
		t.Fatal(err)
	}
	p.ID = id
	defer func() {
		if previous != nil {
			_ = m.pg.SetActiveProfile(previous.ID)
		} else {
			_, _ = m.pg.Exec(`UPDATE llm_profiles SET is_default=false WHERE id=$1`, id)
		}
		_ = m.pg.DeleteProfile(id)
	}()
	if err = m.pg.SetActiveProfile(id); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.environmentLLMFallback(); ok {
		t.Fatal("unready saved SDK profile inherited cloud environment configuration")
	}
	p.Format = "openai"
	if _, err = m.pg.SaveProfile(p); err != nil {
		t.Fatal(err)
	}
	cfg, ok := s.environmentLLMFallback()
	if !ok || cfg.Provider() != "openai" || cfg.APIKey != "unrelated-cloud-key" {
		t.Fatalf("legacy environment fallback changed: provider=%s ready=%v", cfg.Provider(), ok)
	}
}
