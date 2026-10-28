package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Autumn-27/artex/opencodesdk"
	"github.com/Autumn-27/norma/llm"
)

func TestConfigFromOpenCodeSDK(t *testing.T) {
	c := ConfigFrom(" opencode-sdk ", "opencode/test/model", "", "bridge-token", "")
	if c.Provider() != opencodesdk.Format || c.Format != llm.Format(opencodesdk.Format) || c.BaseURL != opencodesdk.DefaultBaseURL || c.Model != "opencode/test/model" || !c.Ready() {
		t.Fatalf("config=%+v", c)
	}
	c.APIKey = ""
	if c.Ready() {
		t.Fatal("SDK must require its bridge token even on loopback")
	}
}

func TestSDKConnectionHonorsConfiguredTokenLimit(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, tc := range []struct {
			name, provider   string
			configured, want int
		}{
			{"SDK configured", opencodesdk.Format, 1024, 1024},
			{"SDK unspecified", opencodesdk.Format, 0, 8192},
			{"SDK nonpositive", opencodesdk.Format, -1, 8192},
			{"OpenAI default unchanged", "openai", 1024, 8192},
		} {
			t.Run(fmt.Sprintf("%s/stream=%v", tc.name, stream), func(t *testing.T) {
				var calls, maxTokens atomic.Int32
				bridge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("Authorization") != "Bearer bridge-token" {
						t.Error("connection request did not use its dedicated token")
						http.Error(w, "unauthorized", http.StatusUnauthorized)
						return
					}
					if r.URL.Path == "/v1/capabilities" {
						fmt.Fprint(w, `{"protocol":"scopeweaver-opencode-sdk","version":1,"runtime":"native","sdk_version":"2.1.4","tool_execution":"capture-only","streaming":true}`)
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
					if tc.provider == opencodesdk.Format {
						if stream {
							w.Header().Set("Content-Type", "application/x-ndjson")
							fmt.Fprintln(w, `{"type":"message_start"}`)
							fmt.Fprintln(w, `{"type":"text_delta","text":"OK"}`)
							fmt.Fprintln(w, `{"type":"message_delta","stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1,"cache_read_tokens":0,"cache_write_tokens":0}}`)
							fmt.Fprintln(w, `{"type":"message_stop"}`)
						} else {
							fmt.Fprint(w, `{"message":{"role":"assistant","content":[{"type":"text","text":"OK"}]},"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1,"cache_read_tokens":0,"cache_write_tokens":0}}`)
						}
					} else if stream {
						w.Header().Set("Content-Type", "text/event-stream")
						fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"OK\"},\"finish_reason\":null}]}\n\ndata: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":1}}\n\ndata: [DONE]\n\n")
					} else {
						fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`)
					}
				}))
				defer bridge.Close()
				cfg := ConfigFrom(tc.provider, "fixture/model", bridge.URL, "bridge-token", "")
				cfg.Stream, cfg.MaxTokens = stream, tc.configured
				_, reply, err := TestConnection(t.Context(), cfg)
				if err != nil || reply != "OK" || calls.Load() != 1 || int(maxTokens.Load()) != tc.want {
					t.Fatalf("reply=%q error=%v inference calls=%d max_tokens=%d want 1/%d", reply, err, calls.Load(), maxTokens.Load(), tc.want)
				}
			})
		}
	}
}
func TestFromEnvOpenCodeSDKTokenIsolation(t *testing.T) {
	for _, tc := range []struct {
		name, modern, dedicated string
		explicit                bool
		want                    string
	}{{"dedicated only", "", "dedicated-token", false, "dedicated-token"}, {"generic override", "generic-token", "dedicated-token", true, "generic-token"}, {"explicit blank suppresses fallback", "", "dedicated-token", true, ""}, {"never cloud fallback", "", "", false, ""}} {
		t.Run(tc.name, func(t *testing.T) {
			isolateLLMEnvironment(t)
			t.Setenv("SCOPEWEAVER_LLM_PROVIDER", opencodesdk.Format)
			t.Setenv("SCOPEWEAVER_LLM_MODEL", "opencode/test")
			t.Setenv("SCOPEWEAVER_OPENCODE_TOKEN", tc.dedicated)
			t.Setenv("OPENAI_API_KEY", "cloud-openai")
			t.Setenv("OPENCODE_API_KEY", "cloud-go")
			if tc.explicit {
				t.Setenv("SCOPEWEAVER_LLM_API_KEY", tc.modern)
			}
			cfg, ok := FromEnv()
			if cfg.APIKey != tc.want || ok != (tc.want != "") {
				t.Fatalf("key=%q ready=%v want=%q", cfg.APIKey, ok, tc.want)
			}
		})
	}
}
func TestOpenCodeSDKPolicyAndProxy(t *testing.T) {
	c := ConfigFrom(opencodesdk.Format, "opencode/test", "", "bridge-token", "http://127.0.0.1:9999")
	if _, err := c.NewProvider(); err == nil {
		t.Fatal("SDK accepted bridge proxy")
	}
	c.Proxy = ""
	var strict atomic.Bool
	strict.Store(true)
	c.PolicyResolver = func() (bool, []string) { return strict.Load(), nil }
	if _, err := c.NewProvider(); err == nil || !strings.Contains(err.Error(), "local_llm_only") {
		t.Fatalf("policy error=%v", err)
	}
	strict.Store(false)
	p, err := c.NewProvider()
	if err != nil {
		t.Fatal(err)
	}
	strict.Store(true)
	if _, _, _, err = p.Complete(t.Context(), llm.CompletionRequest{}); err == nil || !strings.Contains(err.Error(), "local_llm_only") {
		t.Fatalf("cached provider ignored policy change: %v", err)
	}
}

func TestSDKQuotaClassificationUsesStructuredEvidence(t *testing.T) {
	cases := []struct {
		name          string
		status        int
		code, message string
		quota         bool
	}{
		{"genuine quota", 402, "quota_exhausted", "Account allowance unavailable", true},
		{"generic payment failure", 402, "sdk_failure", "quota exceeded", false},
		{"transient rate limit", 429, "rate_limit", "quota exceeded RPM", false},
		{"untrusted 429 quota code", 429, "quota_exhausted", "quota exhausted", false},
		{"auth", 401, "quota_exhausted", "quota exhausted", false},
		{"forbidden", 403, "quota_exhausted", "quota exhausted", false},
		{"server failure", 503, "quota_exhausted", "quota exhausted", false},
	}
	for _, stream := range []bool{false, true} {
		for _, tc := range cases {
			if stream && tc.status != 402 {
				continue
			}
			t.Run(fmt.Sprintf("%v/%s", stream, tc.name), func(t *testing.T) {
				b := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/v1/capabilities" {
						fmt.Fprint(w, `{"protocol":"scopeweaver-opencode-sdk","version":1,"runtime":"native","sdk_version":"2.1.4","tool_execution":"capture-only","streaming":true}`)
						return
					}
					if r.Method == "DELETE" {
						return
					}
					if stream {
						w.Header().Set("Content-Type", "application/x-ndjson")
						_ = json.NewEncoder(w).Encode(map[string]any{"type": "error", "code": tc.code, "message": tc.message})
					} else {
						w.WriteHeader(tc.status)
						_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": tc.code, "message": tc.message}})
					}
				}))
				defer b.Close()
				c := ConfigFrom(opencodesdk.Format, "opencode/test", b.URL, "bridge-token", "")
				p, err := c.NewProvider()
				if err != nil {
					t.Fatal(err)
				}
				var callErr error
				if stream {
					for _, e := range p.Stream(t.Context(), llm.CompletionRequest{MaxTokens: 64}) {
						if e != nil {
							callErr = e
						}
					}
				} else {
					_, _, _, callErr = p.Complete(t.Context(), llm.CompletionRequest{MaxTokens: 64})
				}
				if callErr == nil || IsQuotaExhaustedError(callErr) != tc.quota {
					t.Fatalf("quota=%v error=%v want=%v", IsQuotaExhaustedError(callErr), callErr, tc.quota)
				}
				var bridge *opencodesdk.BridgeError
				if !errors.As(callErr, &bridge) {
					t.Fatalf("lost structured bridge error: %v", callErr)
				}
				if !stream && bridge.Status != tc.status {
					t.Fatalf("transport rewrote status %d to %d", tc.status, bridge.Status)
				}
			})
		}
	}
	for _, model := range []string{"", "model", "p/", "/m", "p//m", "p/m m"} {
		c := ConfigFrom(opencodesdk.Format, model, "", "token", "")
		if _, err := c.NewProvider(); err == nil {
			t.Fatalf("configured invalid SDK model %q", model)
		}
	}
}
