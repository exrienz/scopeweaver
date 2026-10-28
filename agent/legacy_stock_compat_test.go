package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/locale"
	actool "github.com/Autumn-27/norma/tool"
)

// Upgrades of the built-in planner/worker templates must keep recognizing the exact
// prior stock bodies that existing installs seeded into agent_prompts — in every
// shipped language. Otherwise a stock install's stored prompt stops being translated
// and stops receiving the corrected guidance, silently becoming "custom" text.
// Fixtures: agent/testdata/legacy_stock_prompts.json (captured read-only from the
// authoritative baseline refs; source revision recorded inside).

func loadLegacyStockPrompts(t *testing.T) map[string]map[string]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "legacy_stock_prompts.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Planner map[string]string `json:"planner"`
		Worker  map[string]string `json:"worker"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, lang := range locale.Languages() {
		if fixture.Planner[string(lang)] == "" || fixture.Worker[string(lang)] == "" {
			t.Fatalf("missing prior planner/worker stock fixture for %s", lang)
		}
	}
	return map[string]map[string]string{"planner": fixture.Planner, "worker": fixture.Worker}
}

func stockDigest(text string) string {
	h := sha256.Sum256([]byte(text))
	return hex.EncodeToString(h[:])
}

// TestPriorStockPromptsStillRecognized: every prior stock planner/worker body in
// every shipped language canonicalizes to the current builtin (so renderSystem upgrades and
// translates it), while any byte-edited variant remains user content.
func TestPriorStockPromptsStillRecognized(t *testing.T) {
	current := map[string]string{"planner": plannerDefaultTmpl, "worker": workerDefaultTmpl}
	for agentKey, bodies := range loadLegacyStockPrompts(t) {
		for lang, body := range bodies {
			if got := CanonicalBuiltinPrompt(body); got != current[agentKey] {
				t.Errorf("%s %s prior stock prompt not recognized as current builtin (hash %s)", agentKey, lang, stockDigest(body))
			}
			if got := CanonicalBuiltinPrompt(body + " user edit"); got != body+" user edit" {
				t.Errorf("%s %s edited prior stock prompt was canonicalized", agentKey, lang)
			}
		}
	}
}

// TestPriorStockPromptRendersCurrentGuidance: an install whose DB still holds the
// prior stock body must render the CURRENT stock template (with the corrected
// guidance and language), not the stale stored bytes.
func TestPriorStockPromptRendersCurrentGuidance(t *testing.T) {
	old := PromptOverride
	defer func() { PromptOverride = old }()
	bodies := loadLegacyStockPrompts(t)
	for _, tc := range []struct{ key, lang, marker string }{
		{"planner", "en", "already recorded finding"},
		{"planner", "ko", "이미 기록된 발견"},
		{"planner", "zh-Hans", "已记录的发现"},
		{"worker", "en", "already recorded finding"},
		{"worker", "ko", "이미 기록된 발견"},
		{"worker", "zh-Hans", "已记录的发现"},
	} {
		PromptOverride = func(k string) (string, bool) {
			if k == tc.key {
				return bodies[tc.key][tc.lang], true
			}
			return "", false
		}
		def, vars := plannerDefaultTmpl, PlannerVars{Goal: "target"}
		if tc.key == "worker" {
			def = workerDefaultTmpl
		}
		lang := locale.Lang(tc.lang)
		got := renderSystem(tc.key, def, vars, lang)
		if !strings.Contains(got, tc.marker) {
			t.Errorf("%s %s: prior stock body did not render current guidance marker %q", tc.key, tc.lang, tc.marker)
		}
	}
}

// Compare actual runtime rendering with a fresh current template in every
// requested language. Historical source prose is fixture data, not a translation
// oracle; a byte edit must retain that exact user's prose in every output language.
func TestPriorStockPromptRendersEveryShippedLanguage(t *testing.T) {
	old := PromptOverride
	defer func() { PromptOverride = old }()
	bodies := loadLegacyStockPrompts(t)
	for key, def := range map[string]string{"planner": plannerDefaultTmpl, "worker": workerDefaultTmpl} {
		var vars any = PlannerVars{Goal: "target", Scope: "fixture scope", AssetSummary: "fixture asset", DataDir: "proof-data", Now: "fixed fixture time"}
		if key == "worker" {
			vars = WorkerVars{ProxyAddr: "http://127.0.0.1:1", WorkerName: "fixture-worker", DataDir: "proof-data", Now: "fixed fixture time"}
		}
		for _, sourceLang := range locale.Languages() {
			body := bodies[key][string(sourceLang)]
			if _, err := renderTmpl(body, vars); err != nil {
				t.Fatalf("%s/%s prior fixture cannot render with its role variables: %v", key, sourceLang, err)
			}
			for _, outputLang := range locale.Languages() {
				t.Run(key+"/"+string(sourceLang)+"/"+string(outputLang), func(t *testing.T) {
					PromptOverride = nil
					want := renderSystem(key, def, vars, outputLang)
					fresh, err := renderTmpl(BuiltinPromptText(def, outputLang), vars)
					if err != nil || want != fresh {
						t.Fatalf("fresh %s/%s runtime prompt did not render its current template: %v", key, outputLang, err)
					}
					PromptOverride = func(agentKey string) (string, bool) { return body, agentKey == key }
					if got := renderSystem(key, def, vars, outputLang); got != want {
						t.Errorf("prior %s stock did not render the fresh current %s template", sourceLang, outputLang)
					}
					edited := body + " user edit"
					PromptOverride = func(agentKey string) (string, bool) { return edited, agentKey == key }
					customWant, err := renderTmpl(edited, vars)
					if err != nil {
						t.Fatal(err)
					}
					if got := renderSystem(key, def, vars, outputLang); got != customWant {
						t.Fatal("byte-edited prior stock prompt was rewritten or translated")
					}
				})
			}
		}
	}
}

// Upgrades of factory tool schemas face a different persistence seam: wireTools
// seeds rows first-insert-only and ToolResolve overlays the stored schema, so a
// factory fix alone never reaches existing installs. Runtime resolution must
// recognize the exact prior-stock persisted schema and substitute the corrected
// current stock schema, while preserving customized schemas and never mutating the
// stored input. Fixture: agent/testdata/legacy_add_intent_schemas.json.

func loadLegacyAddIntentSchemas(t *testing.T) map[string]struct {
	hash   string
	schema map[string]any
} {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "legacy_add_intent_schemas.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		AddIntent map[string]struct {
			SchemaSHA256 string         `json:"schema_sha256"`
			InputSchema  map[string]any `json:"input_schema"`
		} `json:"add_intent"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, lang := range locale.Languages() {
		if fixture.AddIntent[string(lang)].InputSchema == nil {
			t.Fatalf("missing prior add_intent stock fixture for %s", lang)
		}
	}
	out := map[string]struct {
		hash   string
		schema map[string]any
	}{}
	for lang, snap := range fixture.AddIntent {
		out[lang] = struct {
			hash   string
			schema map[string]any
		}{snap.SchemaSHA256, snap.InputSchema}
	}
	return out
}

func currentAddIntent(t *testing.T, lang locale.Lang) actool.CoreTool {
	t.Helper()
	for _, tool := range NewToolSet(nil, "", lang).PlannerTools() {
		if tool.Name() == "add_intent" {
			return tool
		}
	}
	t.Fatal("add_intent missing")
	return nil
}

// TestLegacyStockAddIntentSchemaUpgradesAtRuntime: each persisted prior-stock
// language schema resolves to the corrected current stock schema of the requested
// language; the persisted input map is never mutated; the stored bytes are not
// rewritten (pure function over copies).
func TestLegacyStockAddIntentSchemaUpgradesAtRuntime(t *testing.T) {
	for langName, snap := range loadLegacyAddIntentSchemas(t) {
		// Fixture integrity: the recorded hash must match the fixture schema bytes.
		encoded, err := json.Marshal(snap.schema)
		if err != nil {
			t.Fatal(err)
		}
		if d := stockDigest(string(encoded)); d != snap.hash {
			t.Fatalf("%s fixture hash mismatch: %s", langName, d)
		}
		persisted := cloneToolSchema(snap.schema)
		for _, resolveLang := range locale.Languages() {
			current := currentAddIntent(t, resolveLang)
			resolved := ResolveLegacyStockSchema("add_intent", persisted, current.InputSchema())
			want, _ := json.Marshal(current.InputSchema())
			got, _ := json.Marshal(resolved)
			if string(got) != string(want) {
				t.Errorf("persisted %s stock schema did not resolve to current %s stock schema", langName, resolveLang)
			}
		}
		again, _ := json.Marshal(persisted)
		if string(again) != string(encoded) {
			t.Fatalf("%s persisted schema mutated by resolution", langName)
		}
	}
}

// assertSchemaPreserved fails on nil (a regression resolving non-nil persisted input
// to nil would silently fall back to factory metadata in DecorateTool) and on any
// byte change; each preservation path is asserted unconditionally.
func assertSchemaPreserved(t *testing.T, label string, persisted map[string]any, resolve func() map[string]any) {
	t.Helper()
	before, _ := json.Marshal(persisted)
	resolved := resolve()
	if resolved == nil {
		t.Fatalf("%s: resolution returned nil for non-nil persisted schema", label)
	}
	after, _ := json.Marshal(resolved)
	if string(after) != string(before) {
		t.Fatalf("%s: schema was rewritten\npersisted: %s\nresolved:  %s", label, before, after)
	}
}

// TestCustomizedAddIntentSchemaIsPreserved: any schema that differs from prior
// stock — page edits of a description, an added property, a changed default — must
// survive runtime resolution untouched, each verified independently rather than via
// one combined marker.
func TestCustomizedAddIntentSchemaIsPreserved(t *testing.T) {
	for sourceLang, snap := range loadLegacyAddIntentSchemas(t) {
		for _, outputLang := range locale.Languages() {
			t.Run(sourceLang+"/"+string(outputLang), func(t *testing.T) {
				current := currentAddIntent(t, outputLang)

				t.Run("custom description only", func(t *testing.T) {
					edited := cloneToolSchema(snap.schema)
					edited["properties"].(map[string]any)["summary"].(map[string]any)["description"] = "user-customized summary wording"
					assertSchemaPreserved(t, "custom description", edited, func() map[string]any {
						return ResolveLegacyStockSchema("add_intent", edited, current.InputSchema())
					})
				})
				t.Run("custom property and default only", func(t *testing.T) {
					edited := cloneToolSchema(snap.schema)
					edited["x-custom"] = "keep me"
					edited["properties"].(map[string]any)["priority"].(map[string]any)["default"] = 7
					assertSchemaPreserved(t, "custom property/default", edited, func() map[string]any {
						return ResolveLegacyStockSchema("add_intent", edited, current.InputSchema())
					})
				})
				t.Run("unregistered tool", func(t *testing.T) {
					other := cloneToolSchema(snap.schema)
					assertSchemaPreserved(t, "unregistered tool", other, func() map[string]any {
						return ResolveLegacyStockSchema("node_detail", other, current.InputSchema())
					})
				})
				t.Run("current stock schema", func(t *testing.T) {
					fresh := cloneToolSchema(currentAddIntent(t, locale.En).InputSchema())
					assertSchemaPreserved(t, "current stock", fresh, func() map[string]any {
						return ResolveLegacyStockSchema("add_intent", fresh, currentAddIntent(t, locale.En).InputSchema())
					})
				})

			})
		}
	}
}

// TestDecoratedLegacyStockAddIntentLocalizes: the full runtime composition used by
// server ToolResolve (DecorateTool after stock-schema resolution, then builtin
// localization) must behave exactly like a fresh install for stock rows and keep
// user edits for customized rows.
func TestDecoratedLegacyStockAddIntentLocalizes(t *testing.T) {
	for sourceLang, snap := range loadLegacyAddIntentSchemas(t) {
		for _, outputLang := range locale.Languages() {
			t.Run(sourceLang+"/"+string(outputLang), func(t *testing.T) {
				persisted := cloneToolSchema(snap.schema)
				current := currentAddIntent(t, locale.En)
				resolved := ResolveLegacyStockSchema("add_intent", persisted, current.InputSchema())
				prepared := DecorateTool(current, current.Description(), resolved)
				localized := localizeBuiltinTools([]actool.CoreTool{prepared}, outputLang)[0]
				want := currentAddIntent(t, outputLang)
				wantSchema, _ := json.Marshal(want.InputSchema())
				gotSchema, _ := json.Marshal(localized.InputSchema())
				if string(gotSchema) != string(wantSchema) {
					t.Fatal("stock row did not localize like a fresh install after schema resolution")
				}
				if localized.Description() != want.Description() {
					t.Fatal("stock row description did not localize like a fresh install")
				}

			})
		}
	}
}
