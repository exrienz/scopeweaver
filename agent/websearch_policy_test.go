package agent

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/locale"
	"github.com/Autumn-27/norma/agentcore"
	"github.com/Autumn-27/norma/permission"
	actool "github.com/Autumn-27/norma/tool"
)

func deepSeekSearchOptions() agentcore.Options {
	return agentcore.Options{EnableWebSearch: true, WebSearchBackend: "deepseek", DeepSeekSearchBaseURL: "http://127.0.0.1:1/anthropic", DeepSeekSearchAPIKey: "fixture-only", DeepSeekSearchModel: "fixture-model", WebSearchProxy: "http://127.0.0.1:2"}
}

func TestDeepSeekSearchLivePolicyAndCapturedBackend(t *testing.T) {
	strict, calls := false, 0
	settings := WebSearchOpts{Enabled: true, Backend: "deepseek", PolicyResolver: func() (bool, []string) { return strict, []string{"127.0.0.0/8"} }}
	opts := deepSeekSearchOptions()
	secureWebSearch(&opts, settings.PolicyResolver)
	if opts.EnableWebSearch || len(opts.Tools) != 1 {
		t.Fatalf("SDK search not replaced: enabled=%v tools=%d", opts.EnableWebSearch, len(opts.Tools))
	}
	wrapped, ok := opts.Tools[0].(*localOnlyDeepSeekSearch)
	if !ok {
		t.Fatal("missing live policy wrapper")
	}
	sdk, _ := actool.NewWebSearch(actool.WebSearchConfig{Backend: opts.WebSearchBackend, DeepSeekBaseURL: opts.DeepSeekSearchBaseURL, DeepSeekAPIKey: opts.DeepSeekSearchAPIKey, DeepSeekModel: opts.DeepSeekSearchModel, Proxy: opts.WebSearchProxy})
	input := json.RawMessage(`{"query":"fixture"}`)
	if wrapped.Name() != sdk.Name() || wrapped.Description() != sdk.Description() || !reflect.DeepEqual(wrapped.InputSchema(), sdk.InputSchema()) || wrapped.IsReadOnly(input) != sdk.IsReadOnly(input) || wrapped.IsConcurrencySafe(input) != sdk.IsConcurrencySafe(input) || !reflect.DeepEqual(wrapped.CheckPermissions(t.Context(), input, permission.Context{}), sdk.CheckPermissions(t.Context(), input, permission.Context{})) {
		t.Fatal("search tool contract changed")
	}
	// Replace only the network implementation: the same wrapper and live resolver
	// must protect a stale running DeepSeek tool after global backend changes.
	wrapped.CoreTool = actool.Build(actool.Spec{Name: "web_search", Run: func(context.Context, json.RawMessage, *actool.ToolContext) (actool.Result, error) {
		calls++
		return actool.Text("fixture result"), nil
	}})
	result, err := wrapped.Call(t.Context(), input, nil)
	if err != nil || result.IsError || calls != 1 {
		t.Fatalf("strict-off behavior changed: %+v err=%v calls=%d", result, err, calls)
	}
	settings.Backend = "ddgs"
	strict = true
	for _, lang := range locale.Languages() {
		result, err = wrapped.Call(locale.WithLang(t.Context(), lang), input, nil)
		expected := locale.Text(lang, DeepSeekLocalOnlyMessage)
		if err != nil || !result.IsError || result.Flatten() != expected || calls != 1 {
			t.Fatalf("strict live toggle bypassed (%s): %+v err=%v calls=%d", lang, result, err, calls)
		}
		if lang != locale.En && expected == DeepSeekLocalOnlyMessage {
			t.Fatalf("missing %s policy translation", lang)
		}
	}
	strict = false
	result, err = wrapped.Call(t.Context(), input, nil)
	if err != nil || result.IsError || calls != 2 {
		t.Fatal("disabling strict policy did not restore existing tool")
	}
}

func TestSearchPolicyAllBackendsLiveAndDisabled(t *testing.T) {
	for _, backend := range []string{"", "ddgs", "brave-free", "tavily"} {
		t.Run(backend, func(t *testing.T) {
			strict, calls := false, 0
			opts := agentcore.Options{EnableWebSearch: true, WebSearchBackend: backend, BraveSearchAPIKey: "fixture", TavilySearchAPIKey: "fixture"}
			secureWebSearch(&opts, func() (bool, []string) { return strict, nil })
			if opts.EnableWebSearch || len(opts.Tools) != 1 {
				t.Fatalf("unwrapped backend %q: %+v", backend, opts)
			}
			wrapped := opts.Tools[0].(*localOnlyDeepSeekSearch)
			wrapped.CoreTool = actool.Build(actool.Spec{Name: "web_search", Run: func(context.Context, json.RawMessage, *actool.ToolContext) (actool.Result, error) {
				calls++
				return actool.Text("ok"), nil
			}})
			_, _ = wrapped.Call(t.Context(), json.RawMessage(`{"query":"x"}`), nil)
			strict = true
			result, err := wrapped.Call(t.Context(), json.RawMessage(`{"query":"x"}`), nil)
			if calls != 1 || err != nil || !result.IsError {
				t.Fatalf("strict backend bypass: %d %+v %v", calls, result, err)
			}
			strict = false
			_, _ = wrapped.Call(t.Context(), json.RawMessage(`{"query":"x"}`), nil)
			if calls != 2 {
				t.Fatal("strict-off behavior changed")
			}
		})
	}
	opts := deepSeekSearchOptions()
	opts.EnableWebSearch = false
	secureWebSearch(&opts, func() (bool, []string) { return true, nil })
	if opts.EnableWebSearch || len(opts.Tools) != 0 {
		t.Fatal("disabled search added")
	}
}

func TestDeepSeekSearchDefaultPolicyReadsEnvironmentLive(t *testing.T) {
	opts := deepSeekSearchOptions()
	opts.WebSearchBackend = " DeepSeek "
	secureWebSearch(&opts, nil)
	if len(opts.Tools) != 1 {
		t.Fatal("normalized DeepSeek backend missing")
	}
	t.Setenv("SCOPEWEAVER_LOCAL_LLM_ONLY", "true")
	result, err := opts.Tools[0].Call(t.Context(), json.RawMessage(`{"query":"fixture"}`), nil)
	if err != nil || !result.IsError || !strings.Contains(result.Flatten(), "local_llm_only") {
		t.Fatalf("environment policy bypassed: %+v %v", result, err)
	}
}
