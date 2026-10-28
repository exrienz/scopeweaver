package intercept

import (
	"embed"
	"encoding/json"
	"github.com/Autumn-27/artex/locale"
)

//go:embed prompt_zh-Hans.json prompt_zh-Hant.json prompt_ja.json prompt_es.json
var translatedPromptFiles embed.FS

func init() {
	templates := map[string]string{"JudgeContextBoundary": JudgeContextBoundary, "JudgeOutputContract": JudgeOutputContract, "DefaultJudgePrompt": DefaultJudgePrompt}
	for _, lang := range []locale.Lang{locale.ZhHans, locale.ZhHant, locale.Ja, locale.Es} {
		raw, err := translatedPromptFiles.ReadFile("prompt_" + string(lang) + ".json")
		if err != nil {
			panic(err)
		}
		var translated map[string]string
		if err := json.Unmarshal(raw, &translated); err != nil {
			panic(err)
		}
		if len(translated) != len(templates) {
			panic(string(lang) + " judge prompt count mismatch")
		}
		for name, english := range templates {
			text, ok := translated[name]
			if !ok || text == "" {
				panic("missing " + string(lang) + " judge prompt: " + name)
			}
			locale.RegisterAll(map[string]map[locale.Lang]string{english: {lang: text}})
		}
	}
}
