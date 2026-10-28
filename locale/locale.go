// Package locale provides ScopeWeaver's backend internationalization: the
// supported interface languages, request-safe locale negotiation, and a small
// message catalog used for human-facing API/report/notification/CLI text.
//
// Locale is an INTERFACE/OUTPUT preference only. It never alters user content,
// raw HTTP evidence, machine identifiers, tool arguments, asset addresses, code,
// or saved historical data — callers localize explicit human messages via T(),
// not by recursively translating arbitrary strings.
package locale

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
)

// Lang is a supported interface language code.
type Lang string

const (
	// En is English, the product default.
	En Lang = "en"
	// Ko is Korean, the selectable alternative.
	Ko Lang = "ko"
	// ZhHans is Simplified Chinese.
	ZhHans Lang = "zh-Hans"
	// ZhHant is Traditional Chinese.
	ZhHant Lang = "zh-Hant"
	// Ja is Japanese.
	Ja Lang = "ja"
	// Es is Spanish.
	Es Lang = "es"
)

// Default is the fallback language when nothing else is negotiated.
const Default = En

// CookieName is the browser cookie the UI sets to persist the chosen locale.
const CookieName = "scopeweaver_locale"

// QueryParam is the URL query key used on SSE streams and download links, where
// a header cannot be attached easily.
const QueryParam = "lang"

// Supported reports whether l is a language ScopeWeaver ships translations for.
func Supported(l Lang) bool {
	for _, d := range definitions {
		if d.ID == l {
			return true
		}
	}
	return false
}

// Normalize preserves the historical permissive EN/KO primary-subtag behavior.
// Chinese matching keeps scripts: explicit Hans wins the region; Hant is never
// mapped to Hans, and unscripted TW/HK/MO select Hant. Bare zh follows the
// Unicode likely-subtag default zh-Hans-CN. Unsupported tags permit fallback.
func Normalize(s string) (Lang, bool) {
	s = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(s)), "_", "-")
	if s == "" {
		return Default, false
	}
	primary := s
	if i := strings.IndexAny(primary, "-;"); i >= 0 {
		primary = primary[:i]
	}
	// Preserve old callers that accept arbitrary EN/KO suffixes and semicolons.
	if primary == string(En) || primary == string(Ko) {
		return Lang(primary), true
	}
	parts := strings.Split(s, "-")
	for _, part := range parts {
		if part == "" || strings.ContainsAny(part, " ;") {
			return Default, false
		}
	}
	return matchDefinitions(s, parts, definitions)
}

// matchDefinitions uses declared aliases for additional languages too. Chinese
// explicit script matching precedes region aliases: Hans-TW stays Hans; Hant-CN
// stays Hant, never Hans.
func matchDefinitions(s string, parts []string, defs []Definition) (Lang, bool) {
	exact := func(token string) (Lang, bool) {
		for _, d := range defs {
			if strings.ToLower(string(d.ID)) == token {
				return d.ID, true
			}
			for _, alias := range d.Aliases {
				if strings.ToLower(alias) == token {
					return d.ID, true
				}
			}
		}
		return Default, false
	}
	if parts[0] == "zh" {
		script := ""
		for _, part := range parts[1:] {
			if len(part) == 4 {
				if script != "" && script != part {
					return Default, false
				}
				script = part
			}
		}
		if script != "" {
			return exact("zh-" + script)
		}
		if lang, ok := exact(s); ok {
			return lang, true
		}
		// No undeclared Chinese region may silently pick a script.
		return Default, false
	}
	if lang, ok := exact(s); ok {
		return lang, true
	}
	// Longest registered canonical/alias prefix, with subtag boundaries, handles
	// future language/script/region declarations without a per-language switch.
	for n := len(parts) - 1; n >= 1; n-- {
		if lang, ok := exact(strings.Join(parts[:n], "-")); ok {
			return lang, true
		}
	}
	return Default, false
}

// serverDefault holds the configured server-wide default language. It is set once
// at startup (from the settings table / env) and only replaced when an operator
// changes the setting; per-request negotiation never writes it, so there is no
// process-global per-request race.
var serverDefault atomic.Value // stores Lang

// SetServerDefault records the server-wide default language. A zero/unsupported
// value is coerced to Default so callers can pass through raw settings values.
func SetServerDefault(l Lang) {
	if !Supported(l) {
		l = Default
	}
	serverDefault.Store(l)
}

// ServerDefault returns the configured server-wide default language, or Default
// when none has been set.
func ServerDefault() Lang {
	if v, ok := serverDefault.Load().(Lang); ok && Supported(v) {
		return v
	}
	return Default
}

// FromEnv resolves a server default from the ScopeWeaver/ARTEX language env vars.
// SCOPEWEAVER_LANGUAGE wins over the legacy ARTEX_LANGUAGE. It intentionally does
// NOT consult the OS LANG/LC_* locale, so a Korean desktop never silently flips
// the product to Korean. ok is false when neither var names a supported language.
func FromEnv(getenv func(string) string) (Lang, bool) {
	for _, key := range []string{"SCOPEWEAVER_LANGUAGE", "ARTEX_LANGUAGE"} {
		if raw := getenv(key); strings.TrimSpace(raw) != "" {
			if l, ok := Normalize(raw); ok {
				return l, true
			}
		}
	}
	return Default, false
}

type ctxKey struct{}

// WithLang returns a copy of ctx carrying the negotiated request language.
func WithLang(ctx context.Context, l Lang) context.Context {
	if !Supported(l) {
		l = Default
	}
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext returns the language stored by WithLang, or the server default when
// the context carries none.
func FromContext(ctx context.Context) Lang {
	if ctx != nil {
		if l, ok := ctx.Value(ctxKey{}).(Lang); ok && Supported(l) {
			return l
		}
	}
	return ServerDefault()
}

// FromRequest negotiates the language for an HTTP request using the agreed
// precedence: ?lang= query > Accept-Language header > scopeweaver_locale cookie >
// server default. It is pure (reads only the request) and so is safe to call from
// concurrent handlers.
func FromRequest(r *http.Request) Lang {
	if r == nil {
		return ServerDefault()
	}
	if raw := r.URL.Query().Get(QueryParam); raw != "" {
		if l, ok := Normalize(raw); ok {
			return l
		}
	}
	if l, ok := fromAcceptLanguage(r.Header.Get("Accept-Language")); ok {
		return l
	}
	if c, err := r.Cookie(CookieName); err == nil {
		if l, ok := Normalize(c.Value); ok {
			return l
		}
	}
	return ServerDefault()
}

// fromAcceptLanguage picks the highest-priority supported language from a
// standard Accept-Language header, honoring q-weights. Entries without an
// explicit q default to q=1.0 and keep their header order.
func fromAcceptLanguage(header string) (Lang, bool) {
	if strings.TrimSpace(header) == "" {
		return Default, false
	}
	bestLang := Default
	bestQ := -1.0
	found := false
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		tag := part
		q := 1.0
		if semi := strings.Index(part, ";"); semi >= 0 {
			tag = strings.TrimSpace(part[:semi])
			q = parseQ(part[semi+1:])
		}
		l, ok := Normalize(tag)
		if !ok || q <= 0 || q > 1 {
			continue
		}
		// A strict comparison preserves header order among equal weights.
		if q > bestQ {
			bestQ, bestLang, found = q, l, true
		}
	}
	return bestLang, found
}

// parseQ extracts the q-value from an Accept-Language parameter segment like
// "q=0.8". Missing weights default to 1.0; malformed weights are rejected.
func parseQ(seg string) float64 {
	seg = strings.TrimSpace(seg)
	if !strings.HasPrefix(seg, "q=") {
		return 1.0
	}
	q, err := strconv.ParseFloat(strings.TrimSpace(seg[2:]), 64)
	if err != nil || math.IsNaN(q) || math.IsInf(q, 0) || q < 0 || q > 1 {
		return 0
	}
	return q
}

// Resolve uses the configured default for an omitted language.
func Resolve(l Lang) Lang {
	if Supported(l) {
		return l
	}
	return ServerDefault()
}

// First supports optional language arguments without breaking existing callers.
func First(langs []Lang) Lang {
	if len(langs) > 0 {
		return Resolve(langs[0])
	}
	return ServerDefault()
}

// Explicit reports a language attached to a run/request, without applying defaults.
func Explicit(ctx context.Context) (Lang, bool) {
	if ctx != nil {
		if lang, ok := ctx.Value(ctxKey{}).(Lang); ok && Supported(lang) {
			return lang, true
		}
	}
	return Default, false
}
