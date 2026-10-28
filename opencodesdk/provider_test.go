package opencodesdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Autumn-27/artex/locale"
	"github.com/Autumn-27/norma/compaction"
	"github.com/Autumn-27/norma/llm"
	"github.com/Autumn-27/norma/transcript"
)

func TestBridgeErrorOwnedDiagnosticsLocalized(t *testing.T) {
	for _, tc := range []struct {
		code   string
		status int
		want   map[locale.Lang]string
	}{
		{"unauthorized", 401, map[locale.Lang]string{locale.En: "Bridge authentication required.", locale.Ko: "브리지 인증이 필요합니다.", locale.ZhHans: "需要桥接身份验证。", locale.ZhHant: "需要橋接驗證。", locale.Ja: "ブリッジの認証が必要です。", locale.Es: "Se requiere autenticación del puente."}},
		{"quota_exhausted", 402, map[locale.Lang]string{locale.En: "The upstream provider reported exhausted quota or billing credit.", locale.Ko: "상위 제공자가 할당량 또는 결제 크레딧 소진을 보고했습니다.", locale.ZhHans: "上游提供者报告配额或计费余额已耗尽。", locale.ZhHant: "上游提供者回報配額或計費餘額已耗盡。", locale.Ja: "上流のプロバイダーがクォータまたは請求クレジットを使い切ったことを報告しました。", locale.Es: "El proveedor ha informado de que se ha agotado la cuota o el saldo de facturación."}},
		{"canceled", 409, map[locale.Lang]string{locale.En: "The request was canceled.", locale.Ko: "요청이 취소되었습니다.", locale.ZhHans: "请求已取消。", locale.ZhHant: "請求已取消。", locale.Ja: "リクエストがキャンセルされました。", locale.Es: "La solicitud se ha cancelado."}},
		{"context_length_exceeded", 400, map[locale.Lang]string{locale.En: "The upstream model reported that the context length was exceeded.", locale.Ko: "상위 모델이 컨텍스트 길이 초과를 보고했습니다.", locale.ZhHans: "上游模型报告上下文长度已超出限制。", locale.ZhHant: "上游模型回報已超出上下文長度限制。", locale.Ja: "上流のモデルがコンテキスト長の上限超過を報告しました。", locale.Es: "El modelo del proveedor ha informado de que se ha superado el límite de longitud del contexto."}},
		{"free_tier_restricted", 403, map[locale.Lang]string{locale.En: "The hosted free tier is restricted for this SDK runtime.", locale.Ko: "이 SDK 런타임에서는 호스팅 무료 요금제 이용이 제한됩니다.", locale.ZhHans: "此 SDK 运行时使用托管免费套餐受到限制。", locale.ZhHant: "此 SDK 執行環境使用託管免費方案受到限制。", locale.Ja: "この SDK ランタイムでは、ホスト型サービスの無料プランの利用が制限されています。", locale.Es: "El uso del nivel gratuito alojado está restringido para este entorno de ejecución del SDK."}},
	} {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/stream=%v", tc.code, stream), func(t *testing.T) {
				p := newFixture(t, func(w http.ResponseWriter, r *http.Request) {
					if r.Method == "DELETE" {
						return
					}
					if r.URL.Path == "/v1/capabilities" {
						fmt.Fprint(w, capabilities)
						return
					}
					if stream {
						w.Header().Set("Content-Type", "application/x-ndjson")
						fmt.Fprintf(w, `{"type":"error","code":%q,"message":"untrusted bridge-secret evidence"}`+"\n", tc.code)
					} else {
						w.WriteHeader(tc.status)
						fmt.Fprintf(w, `{"error":{"code":%q,"message":"untrusted bridge-secret evidence"}}`, tc.code)
					}
				})
				var err error
				if stream {
					for _, e := range p.Stream(t.Context(), inputFixture()) {
						if e != nil {
							err = e
						}
					}
				} else {
					_, err = p.Models(t.Context())
				}
				var bridge *BridgeError
				if !errors.As(err, &bridge) || bridge.Code != tc.code || (!stream && bridge.Status != tc.status) || bridge.QuotaExhausted() != (tc.code == "quota_exhausted") {
					t.Fatalf("classification lost: %#v, %v", bridge, err)
				}
				for lang, want := range tc.want {
					got := locale.ErrorMessage(lang, err)
					if !strings.Contains(got, want) || strings.Contains(got, "untrusted") || strings.Contains(got, "bridge-secret") {
						t.Errorf("%s display=%q, want %q without upstream prose or credentials", lang, got, want)
					}
				}
			})
		}
	}
}

func TestBridgeErrorContextOverflowPreservesMachineIdentity(t *testing.T) {
	for code, status := range map[string]int{"context_length_exceeded": http.StatusBadRequest, "sdk_failure": http.StatusBadRequest, "free_tier_restricted": http.StatusForbidden} {
		err := &BridgeError{Code: code, Status: status, Message: "arbitrary context_length_exceeded prose"}
		if got := compaction.IsOverflow(err); got != (code == "context_length_exceeded") {
			t.Errorf("%s: compaction overflow=%v, internal error=%q", code, got, err.Error())
		}
		if code == "context_length_exceeded" && !strings.Contains(err.Error(), code) {
			t.Errorf("machine marker lost: %q", err.Error())
		}
		if err.QuotaExhausted() {
			t.Errorf("%s incorrectly classified as quota", code)
		}
	}
}

func TestBridgeErrorDiagnosticCatalogAndUnknownEvidence(t *testing.T) {
	for code, diagnostic := range bridgeDiagnostics {
		for _, lang := range locale.Languages() {
			translated, ok := locale.Raw(lang, diagnostic.Error())
			got := locale.ErrorMessage(lang, &BridgeError{Code: code, Message: "untrusted upstream prose"})
			if !ok || translated == "" || got != translated {
				t.Errorf("%s/%s missing owned diagnostic: %q", code, lang, got)
			}
		}
	}
	p := newFixture(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprint(w, `{"error":{"code":"future_upstream_code","message":"Bridge authentication required. bridge-secret 原文"}}`)
	})
	_, err := p.Models(t.Context())
	var bridge *BridgeError
	if !errors.As(err, &bridge) {
		t.Fatal(err)
	}
	for _, lang := range locale.Languages() {
		if got := locale.ErrorMessage(lang, bridge); got != "future_upstream_code: Bridge authentication required. [redacted] 原文" {
			t.Errorf("%s unknown evidence changed or leaked credentials: %q", lang, got)
		}
	}
}

const capabilities = `{"protocol":"scopeweaver-opencode-sdk","version":1,"runtime":"native","sdk_version":"2.1.4","tool_execution":"capture-only","streaming":true}`
const completeFixture = `{"message":{"role":"assistant","content":[{"type":"text","text":"Ready"},{"type":"tool_use","id":"host-call","name":"host_read","input":{"id":9223372036854775807}}]},"stop_reason":"tool_use","usage":{"input_tokens":11,"output_tokens":7,"cache_read_tokens":3,"cache_write_tokens":2}}`
const streamFixture = `{"type":"message_start"}
{"type":"text_delta","text":"Ready"}
{"type":"tool_use_start","tool_id":"host-call","tool_name":"host_read"}
{"type":"tool_input_delta","text":"{\"id\":9223372036854775807}"}
{"type":"message_delta","stop_reason":"tool_use","usage":{"input_tokens":11,"output_tokens":7,"cache_read_tokens":3,"cache_write_tokens":2}}
{"type":"message_stop"}
`

func inputFixture() llm.CompletionRequest {
	return llm.CompletionRequest{System: []string{"host system"}, Messages: []llm.Message{llm.UserText("host input")}, Tools: []llm.ToolSchema{{Name: "host_read", Description: "host-owned", InputSchema: map[string]any{"type": "object"}}}, MaxTokens: 777, Thinking: "disabled", Stop: []string{"STOP"}}
}
func newFixture(t *testing.T, handler http.HandlerFunc) *Provider {
	t.Helper()
	s := httptest.NewServer(handler)
	t.Cleanup(s.Close)
	p, e := New(s.URL, "bridge-secret", "opencode/model/with/slashes", Options{ThinkingType: "enabled", ReasoningEffort: "low"})
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func TestCompleteAndNativeStreamContract(t *testing.T) {
	var mu sync.Mutex
	var inputs []completionInput
	var paths []string
	p := newFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer bridge-secret" {
			t.Error("wrong bridge authentication")
		}
		mu.Lock()
		paths = append(paths, r.Method+" "+r.URL.Path)
		mu.Unlock()
		switch r.URL.Path {
		case "/v1/capabilities":
			fmt.Fprint(w, capabilities)
		case "/v1/models":
			fmt.Fprint(w, `{"models":[{"id":"opencode/model/with/slashes","name":"Fixture"}]}`)
		case "/v1/complete", "/v1/stream":
			var in completionInput
			if e := json.NewDecoder(r.Body).Decode(&in); e != nil {
				t.Error(e)
			}
			mu.Lock()
			inputs = append(inputs, in)
			mu.Unlock()
			if strings.HasSuffix(r.URL.Path, "stream") {
				w.Header().Set("Content-Type", "application/x-ndjson")
				fmt.Fprint(w, streamFixture)
			} else {
				fmt.Fprint(w, completeFixture)
			}
		default:
			if r.Method != "DELETE" {
				t.Errorf("unexpected %s", r.URL.Path)
			}
		}
	})
	models, e := p.Models(t.Context())
	if e != nil || len(models) != 1 || models[0].ID != "opencode/model/with/slashes" {
		t.Fatalf("models=%v err=%v", models, e)
	}
	mu.Lock()
	if len(inputs) != 0 {
		t.Fatal("model discovery invoked inference")
	}
	mu.Unlock()
	ctx := transcript.WithSessionID(t.Context(), "conv-sdk-host-session")
	msg, stop, usage, e := p.Complete(ctx, inputFixture())
	if e != nil {
		t.Fatal(e)
	}
	if msg.Text() != "Ready" || stop != "tool_use" || usage.InputTokens != 16 || usage.OutputTokens != 7 || usage.CacheReadTokens != 3 || len(msg.ToolUses()) != 1 {
		t.Fatalf("complete=%+v %s %+v", msg, stop, usage)
	}
	var events []llm.StreamEvent
	for ev, e := range p.Stream(ctx, inputFixture()) {
		if e != nil {
			t.Fatal(e)
		}
		events = append(events, ev)
	}
	if len(events) != 6 || events[1].Text != msg.Text() || events[2].ToolID != msg.ToolUses()[0].ID || events[3].Text != string(msg.ToolUses()[0].Input) || events[4].Usage != usage || events[4].StopReason != stop || events[5].Type != llm.SEMessageStop {
		t.Fatalf("stream=%+v", events)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(inputs) != 2 {
		t.Fatalf("inference count=%d", len(inputs))
	}
	for _, in := range inputs {
		if in.RequestID == "" || in.SessionID != "conv-sdk-host-session" || in.Model != "opencode/model/with/slashes" || in.MaxTokens != 777 || in.Thinking != "disabled" || in.ThinkingType != "enabled" || in.ReasoningEffort != "low" || len(in.Tools) != 1 || in.System[0] != "host system" || in.Messages[0].Text() != "host input" {
			t.Fatalf("lost request fields: %+v", in)
		}
	}
	if inputs[0].RequestID == inputs[1].RequestID {
		t.Fatal("calls share cleanup identity")
	}
}
func TestEndpointValidationAndCredentialScope(t *testing.T) {
	for _, raw := range []string{"https://example.com", "http://10.0.0.1", "http://127.0.0.1/?x", "http://u:p@localhost", "http://localhost/#f", "file:///tmp/bridge", "http://localhost/?"} {
		if _, e := NormalizeBaseURL(raw); e == nil {
			t.Errorf("accepted %q", raw)
		}
	}
	for _, pair := range [][2]string{{"HTTP://LOCALHOST:80/bridge/", "http://localhost/bridge"}, {"http://127.0.0.1:4097/", "http://127.0.0.1:4097"}} {
		if !SameEndpoint(pair[0], pair[1]) {
			t.Errorf("equivalent endpoints differ: %v", pair)
		}
	}
	for _, pair := range [][2]string{{"http://localhost", "http://127.0.0.1"}, {"http://localhost/a", "http://localhost/b"}, {"http://localhost:4097", "http://localhost:4098"}} {
		if SameEndpoint(pair[0], pair[1]) {
			t.Errorf("different endpoints equated: %v", pair)
		}
	}
	for _, token := range []string{"", "x\ny"} {
		if _, e := New("http://127.0.0.1", token, "p/m", Options{}); e == nil {
			t.Error("accepted missing/malformed token")
		}
	}
}
func TestCapabilitiesBeforeInferenceAndDynamicPolicy(t *testing.T) {
	var calls atomic.Int32
	var strict atomic.Bool
	var capBody atomic.Value
	capBody.Store(capabilities)
	p := newFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path == "/v1/capabilities" {
			fmt.Fprint(w, capBody.Load())
		} else {
			fmt.Fprint(w, completeFixture)
		}
	})
	p.opts.LocalOnly = strict.Load
	if _, e := p.Capabilities(t.Context()); e != nil {
		t.Fatal(e)
	}
	before := calls.Load()
	strict.Store(true)
	if _, _, _, e := p.Complete(t.Context(), inputFixture()); e == nil || !strings.Contains(e.Error(), "local_llm_only") {
		t.Fatalf("strict policy error=%v", e)
	}
	if calls.Load() != before {
		t.Fatal("policy violation reached bridge")
	}
	strict.Store(false)
	for _, bad := range []string{strings.Replace(capabilities, "2.1.4", "2.1.5", 1), strings.Replace(capabilities, "capture-only", "execute", 1), strings.Replace(capabilities, `"version":1`, `"version":2`, 1), `{`} {
		capBody.Store(bad)
		before = calls.Load()
		if _, _, _, e := p.Complete(t.Context(), inputFixture()); e == nil {
			t.Fatal("unsupported capabilities permitted inference")
		}
		if calls.Load() != before+1 {
			t.Fatal("inference called after invalid capabilities")
		}
	}
}
func TestStreamRejectsBrokenProtocol(t *testing.T) {
	cases := map[string]string{"unknown": `{"type":"surprise"}`, "missing start": `{"type":"text_delta","text":"x"}`, "malformed": `{`, "missing terminal": `{"type":"message_start"}`, "unknown tool": strings.Replace(streamFixture, "host_read", "not_offered", 1), "duplicate stop": streamFixture + "{\"type\":\"message_stop\"}\n", "missing usage": strings.Replace(streamFixture, `,"usage":{"input_tokens":11,"output_tokens":7,"cache_read_tokens":3,"cache_write_tokens":2}`, "", 1), "bad args": strings.Replace(streamFixture, `{\"id\":9223372036854775807}`, `{`, 1), "explicit error": "{\"type\":\"error\",\"code\":\"unsupported\",\"message\":\"bridge-secret\"}\n"}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			p := newFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/v1/capabilities" {
					fmt.Fprint(w, capabilities)
					return
				}
				if r.Method == "DELETE" {
					return
				}
				w.Header().Set("Content-Type", "application/x-ndjson")
				fmt.Fprint(w, body)
			})
			var got error
			for _, e := range p.Stream(t.Context(), inputFixture()) {
				if e != nil {
					got = e
				}
			}
			if got == nil {
				t.Fatal("broken protocol succeeded")
			}
			if strings.Contains(got.Error(), "bridge-secret") {
				t.Fatal("error leaked token")
			}
		})
	}
}
func TestCompleteRejectsMalformedData(t *testing.T) {
	for _, body := range []string{`{}`, `{`, strings.Replace(completeFixture, "host_read", "unknown", 1), strings.Replace(completeFixture, `"output_tokens":7`, `"output_tokens":-1`, 1), completeFixture + `{}`} {
		p := newFixture(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/v1/capabilities" {
				fmt.Fprint(w, capabilities)
			} else if r.Method != "DELETE" {
				fmt.Fprint(w, body)
			}
		})
		if _, _, _, e := p.Complete(t.Context(), inputFixture()); e == nil {
			t.Fatal("malformed response succeeded")
		}
	}
}
func TestCancellationDeletesOnlyOwnedRequest(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprint(stream), func(t *testing.T) {
			entered := make(chan string, 1)
			deleted := make(chan string, 1)
			p := newFixture(t, func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path == "/v1/capabilities":
					fmt.Fprint(w, capabilities)
				case r.Method == "DELETE":
					deleted <- strings.TrimPrefix(r.URL.Path, "/v1/requests/")
				case r.Method == "POST":
					var in completionInput
					_ = json.NewDecoder(r.Body).Decode(&in)
					entered <- in.RequestID
					<-r.Context().Done()
				}
			})
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan error, 1)
			go func() {
				if stream {
					var got error
					for _, e := range p.Stream(ctx, inputFixture()) {
						if e != nil {
							got = e
						}
					}
					done <- got
				} else {
					_, _, _, e := p.Complete(ctx, inputFixture())
					done <- e
				}
			}()
			id := <-entered
			cancel()
			select {
			case e := <-done:
				if e == nil {
					t.Fatal("cancel succeeded")
				}
			case <-time.After(3 * time.Second):
				t.Fatal("cancel hung")
			}
			select {
			case got := <-deleted:
				if got != id {
					t.Fatalf("cleanup %q want %q", got, id)
				}
			case <-time.After(time.Second):
				t.Fatal("no explicit cancellation cleanup")
			}
		})
	}
}
func TestRedirectCannotLeakToken(t *testing.T) {
	var leaked atomic.Int32
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked.Add(1) }))
	defer foreign.Close()
	p := newFixture(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, foreign.URL, http.StatusFound) })
	if _, e := p.Capabilities(t.Context()); e == nil || !strings.Contains(e.Error(), "302") {
		t.Fatalf("redirect error=%v", e)
	}
	if leaked.Load() != 0 {
		t.Fatal("redirect reached other endpoint")
	}
}

func TestPolicyChangeDuringCapabilityCheckBlocksInference(t *testing.T) {
	var strict atomic.Bool
	var posts atomic.Int32
	p := newFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/capabilities" {
			strict.Store(true)
			fmt.Fprint(w, capabilities)
		} else if r.Method == "POST" {
			posts.Add(1)
			fmt.Fprint(w, completeFixture)
		}
	})
	p.opts.LocalOnly = strict.Load
	if _, _, _, err := p.Complete(t.Context(), inputFixture()); err == nil || !strings.Contains(err.Error(), "local_llm_only") {
		t.Fatalf("policy error=%v", err)
	}
	if posts.Load() != 0 {
		t.Fatal("inference crossed policy change after capabilities")
	}
}

func TestSupportedRuntimePairs(t *testing.T) {
	for _, tc := range []struct {
		runtime, version string
		ok               bool
	}{{"native", "2.1.4", true}, {"legacy", "1.19.4", true}, {"native", "1.19.4", false}, {"legacy", "2.1.4", false}, {"native2", "2.1.4", false}, {"legacy1", "1.19.4", false}} {
		t.Run(tc.runtime+"/"+tc.version, func(t *testing.T) {
			body := strings.Replace(strings.Replace(capabilities, `"runtime":"native"`, `"runtime":"`+tc.runtime+`"`, 1), `"sdk_version":"2.1.4"`, `"sdk_version":"`+tc.version+`"`, 1)
			p := newFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/capabilities" {
					t.Errorf("unexpected inference on %s", r.URL.Path)
				}
				fmt.Fprint(w, body)
			})
			_, err := p.Capabilities(t.Context())
			if (err == nil) != tc.ok {
				t.Fatalf("capabilities error=%v want supported=%v", err, tc.ok)
			}
		})
	}
}
