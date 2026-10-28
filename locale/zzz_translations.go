package locale

import (
	"embed"
	"encoding/json"
)

//go:embed zh-Hans.json zh-Hant.json ja.json es.json
var translatedCatalogs embed.FS

// Additional language catalogs are registered after the base EN/KO registrations. Adding
// a translated value never replaces the authoritative English or Korean text.
func init() {
	for _, lang := range []Lang{ZhHans, ZhHant, Ja, Es} {
		raw, err := translatedCatalogs.ReadFile(string(lang) + ".json")
		if err != nil {
			panic(err)
		}
		var entries map[string]string
		if err := json.Unmarshal(raw, &entries); err != nil {
			panic(err)
		}
		for key, text := range entries {
			RegisterAll(map[string]map[Lang]string{key: {lang: text}})
		}
	}
}
