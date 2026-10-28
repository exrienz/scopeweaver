package locale

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSharedMatchingVectors(t *testing.T) {
	raw, err := matchingVectors.ReadFile("matching-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Vectors []struct {
			Input    string `json:"input"`
			Expected string `json:"expected"`
		} `json:"vectors"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, vector := range fixture.Vectors {
		got, ok := Normalize(vector.Input)
		if vector.Expected == "" {
			if ok {
				t.Errorf("%q unexpectedly resolved to %q", vector.Input, got)
			}
			continue
		}
		if !ok || string(got) != vector.Expected {
			t.Errorf("%q: got (%q,%v), want %q", vector.Input, got, ok, vector.Expected)
		}
	}
}

func TestDefinitionDrivenAdditionalLanguages(t *testing.T) {
	defs := append(Definitions(), Definition{ID: "pt", Aliases: []string{"pt", "pt-BR"}})
	for input, want := range map[string]Lang{"pt-br": "pt", "pt-pt": "pt", "zh-hant-cn": "zh-Hant", "zh-tw": "zh-Hant", "zh-hans-tw": ZhHans} {
		got, ok := matchDefinitions(input, strings.Split(input, "-"), defs)
		if !ok || got != want {
			t.Errorf("definition-driven match %s = (%s,%v), want %s", input, got, ok, want)
		}
	}
	if _, ok := Normalize("en;legacy=1"); !ok {
		t.Fatal("historical Go semicolon acceptance lost")
	}
	// Browser parsing historically rejected this malformed primary token; shared
	// vectors cover common accepted inputs, and both preserve their own boundary.
	copy := Definitions()
	copy[0].Aliases[0] = "mutated"
	if Definitions()[0].Aliases[0] == "mutated" {
		t.Fatal("declaration aliases leaked mutable state")
	}
}

func TestRegisterAllRetainsAdditionalLanguagesAndMerges(t *testing.T) {
	const key = "test.keyed"
	defer delete(catalog, key)
	RegisterAll(map[string]map[Lang]string{key: {En: "Hello %s", Ko: "안녕 %s", ZhHans: "你好 %s", ZhHant: "您好 %s", Ja: "こんにちは %s", Es: "Hola %s", "pt": "Olá %s"}})
	Register(key, "Hi %s", "")
	RegisterAll(map[string]map[Lang]string{key: {Ko: ""}})
	if got, _ := Raw(Ko, key); got != "안녕 %s" {
		t.Fatal("empty Korean merge erased existing translation")
	}
	if got, ok := Raw(ZhHans, key); !ok || got != "你好 %s" {
		t.Fatalf("additional language lost: %q", got)
	}
	if got, _ := Raw("ja", key); got != "こんにちは %s" {
		t.Fatal("unshipped keyed language discarded")
	}
	if got := T("unknown", key, "原文"); got != "Hi 原文" {
		t.Fatalf("English fallback changed: %s", got)
	}
	if err := ValidateTranslations([]string{key}); err != nil {
		t.Fatal(err)
	}
	RegisterAll(map[string]map[Lang]string{key: {ZhHans: "你好 %d"}})
	if err := ValidateTranslations([]string{key}); err == nil {
		t.Fatal("incompatible translation passed")
	}
	RegisterAll(map[string]map[Lang]string{key: {ZhHans: ""}})
	if err := ValidateTranslations([]string{key}); err == nil {
		t.Fatal("empty translation passed")
	}
	delete(catalog[key], ZhHans)
	if err := ValidateTranslations([]string{key}); err == nil {
		t.Fatal("missing translation passed via fallback")
	}
}
