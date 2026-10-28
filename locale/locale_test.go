package locale

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := map[string]struct {
		want Lang
		ok   bool
	}{
		"en":    {En, true},
		"EN":    {En, true},
		"en-US": {En, true},
		"en_GB": {En, true},
		"ko":    {Ko, true},
		"ko-KR": {Ko, true},
		"KO":    {Ko, true},
		"fr":    {Default, false},
		"":      {Default, false},
		"zh-CN": {ZhHans, true},
		"zh-TW": {ZhHant, true},
		"ja-JP": {Ja, true},
		"es-MX": {Es, true},
	}
	for in, want := range cases {
		got, ok := Normalize(in)
		if got != want.want || ok != want.ok {
			t.Errorf("Normalize(%q) = (%q,%v), want (%q,%v)", in, got, ok, want.want, want.ok)
		}
	}
}

func TestFromRequestPrecedence(t *testing.T) {
	SetServerDefault(En)
	// query wins over everything
	r := newReq("/x?lang=ko", "en-US,en;q=0.9", "en")
	if got := FromRequest(r); got != Ko {
		t.Errorf("query should win: got %q", got)
	}
	// Accept-Language wins over cookie when no query
	r = newReq("/x", "ko-KR,ko;q=0.9,en;q=0.5", "en")
	if got := FromRequest(r); got != Ko {
		t.Errorf("accept-language should win over cookie: got %q", got)
	}
	// cookie wins when no query and no supported Accept-Language
	r = newReq("/x", "fr-FR,fr;q=0.9", "ko")
	if got := FromRequest(r); got != Ko {
		t.Errorf("cookie should win: got %q", got)
	}
	// server default when nothing matches
	SetServerDefault(Ko)
	r = newReq("/x", "", "")
	if got := FromRequest(r); got != Ko {
		t.Errorf("server default should apply: got %q", got)
	}
	SetServerDefault(En)
	r = newReq("/x", "", "")
	if got := FromRequest(r); got != En {
		t.Errorf("default en: got %q", got)
	}
}

func TestEveryShippedLanguageNegotiatesAcrossBoundaries(t *testing.T) {
	old := ServerDefault()
	defer SetServerDefault(old)
	for _, lang := range Languages() {
		t.Run(string(lang), func(t *testing.T) {
			if got, ok := Normalize(string(lang)); !ok || got != lang {
				t.Fatalf("canonical locale did not normalize: %s, %v", got, ok)
			}
			for _, r := range []*http.Request{
				newReq("/x?lang="+string(lang), "fr", "en"),
				newReq("/x", string(lang)+";q=1,en;q=0.1", "en"),
				newReq("/x", "fr", string(lang)),
			} {
				if got := FromRequest(r); got != lang {
					t.Fatalf("request locale %s, want %s", got, lang)
				}
			}
			SetServerDefault(lang)
			if FromRequest(newReq("/x", "fr", "unsupported")) != lang || FromContext(WithLang(context.Background(), lang)) != lang {
				t.Fatal("server or context preference lost")
			}
			if got, ok := FromEnv(func(key string) string {
				if key == "SCOPEWEAVER_LANGUAGE" {
					return string(lang)
				}
				return ""
			}); !ok || got != lang {
				t.Fatal("explicit environment preference lost")
			}
		})
	}
}

func TestAcceptLanguageQWeights(t *testing.T) {
	// ko has higher q than en -> ko
	r := newReq("/x", "en;q=0.3, ko;q=0.9", "")
	if got := FromRequest(r); got != Ko {
		t.Errorf("q-weight pick: got %q", got)
	}
	// unsupported primary with supported fallback
	r = newReq("/x", "fr, en-US;q=0.8", "")
	if got := FromRequest(r); got != En {
		t.Errorf("fallback to en: got %q", got)
	}
}

func TestFromEnvPrefersScopeWeaver(t *testing.T) {
	env := map[string]string{"SCOPEWEAVER_LANGUAGE": "ko", "ARTEX_LANGUAGE": "en"}
	if l, ok := FromEnv(func(k string) string { return env[k] }); !ok || l != Ko {
		t.Errorf("SCOPEWEAVER_LANGUAGE should win: got %q,%v", l, ok)
	}
	env = map[string]string{"ARTEX_LANGUAGE": "ko"}
	if l, ok := FromEnv(func(k string) string { return env[k] }); !ok || l != Ko {
		t.Errorf("legacy ARTEX_LANGUAGE honored: got %q,%v", l, ok)
	}
	if _, ok := FromEnv(func(string) string { return "" }); ok {
		t.Errorf("empty env should not resolve")
	}
	// OS LANG must be ignored
	env = map[string]string{"LANG": "ko_KR.UTF-8"}
	if _, ok := FromEnv(func(k string) string { return env[k] }); ok {
		t.Errorf("OS LANG must not be consulted")
	}
}

func TestContextRoundTrip(t *testing.T) {
	SetServerDefault(En)
	ctx := WithLang(context.Background(), Ko)
	if got := FromContext(ctx); got != Ko {
		t.Errorf("FromContext = %q", got)
	}
	if got := FromContext(context.Background()); got != En {
		t.Errorf("FromContext default = %q", got)
	}
}

func TestCatalogFallback(t *testing.T) {
	Register("test.hello", "Hello %s", "")
	if got := T(Ko, "test.hello", "world"); got != "Hello world" {
		t.Errorf("Ko should fall back to En: %q", got)
	}
	Register("test.bye", "Bye", "안녕")
	if got := T(Ko, "test.bye"); got != "안녕" {
		t.Errorf("Ko lookup: %q", got)
	}
	if got := T(En, "test.missing.key"); got != "test.missing.key" {
		t.Errorf("unknown key returns key: %q", got)
	}
}

func newReq(target, acceptLang, cookie string) *http.Request {
	r, _ := http.NewRequest(http.MethodGet, "http://x"+target, nil)
	if acceptLang != "" {
		r.Header.Set("Accept-Language", acceptLang)
	}
	if cookie != "" {
		r.AddCookie(&http.Cookie{Name: CookieName, Value: cookie})
	}
	return r
}

func TestRejectedLanguageWeights(t *testing.T) {
	for _, header := range []string{"ko;q=0", "ko;q=-1", "ko;q=2", "ko;q=NaN", "ko;q=Inf", "ko;q=broken"} {
		if got, ok := fromAcceptLanguage(header); ok {
			t.Errorf("accepted excluded/malformed language %q: %s", header, got)
		}
	}
	if got, ok := fromAcceptLanguage("ko;q=0,en;q=0.5"); !ok || got != En {
		t.Fatal("q=0 language was selected")
	}
	if got, ok := fromAcceptLanguage("ko;q=0.5,en;q=0.5"); !ok || got != Ko {
		t.Fatal("equal weights lost header order")
	}
}

func TestLocalizedErrorPreservesWrapping(t *testing.T) {
	cause := errors.New("user supplied 原文 %s")
	Register("Operation %s failed: %w", "Operation %s failed: %w", "작업 %s 실패: %w")
	err := Errorf("Operation %s failed: %w", "raw-id", cause)
	if !errors.Is(err, cause) {
		t.Fatal("error wrapping lost")
	}
	if got := ErrorMessage(Ko, err); got != "작업 raw-id 실패: user supplied 原文 %s" {
		t.Fatalf("raw cause changed: %q", got)
	}
}

func TestJoinedErrorsPreserveRawCauses(t *testing.T) {
	Register("Missing task", "Missing task", "작업 없음")
	raw := errors.New("driver 原文 %s")
	joined := errors.Join(NewError("Missing task"), raw)
	if got := ErrorMessage(Ko, joined); got != "작업 없음\ndriver 原文 %s" {
		t.Fatalf("joined localization altered raw content: %q", got)
	}
	if !errors.Is(joined, raw) {
		t.Fatal("joined cause identity lost")
	}
}
