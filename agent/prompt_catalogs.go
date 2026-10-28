package agent

import (
	"embed"
	"encoding/json"
	"github.com/Autumn-27/artex/locale"
)

//go:embed prompt_zh-Hans.json prompt_zh-Hant.json prompt_ja.json prompt_es.json
var translatedPromptFiles embed.FS

func init() {
	templates := map[string]string{
		"goalsDefaultTmpl":          goalsDefaultTmpl,
		"goalsScopeTail":            goalsScopeTail,
		"plannerDefaultTmpl":        plannerDefaultTmpl,
		"pentestDefaultTmpl":        pentestDefaultTmpl,
		"autoDefaultTmpl":           autoDefaultTmpl,
		"DefaultAssistantPrompt":    DefaultAssistantPrompt,
		"ReporterDefaultPrompt":     ReporterDefaultPrompt,
		"workerDefaultTmpl":         workerDefaultTmpl,
		"mainAgentDefaultTmpl":      mainAgentDefaultTmpl,
		"RetesterDefaultPrompt":     RetesterDefaultPrompt,
		"settleWrapUpPrompt":        settleWrapUpPrompt,
		"plannerWrapUpDefault":      plannerWrapUpDefault,
		"mainAgentWrapUpDefault":    mainAgentWrapUpDefault,
		"genericWrapUpDefault":      genericWrapUpDefault,
		"workerTaskTimeoutDefault":  workerTaskTimeoutDefault,
		"plannerTaskTimeoutDefault": plannerTaskTimeoutDefault,
	}
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
			panic(string(lang) + " agent prompt count mismatch")
		}
		for name, english := range templates {
			text, ok := translated[name]
			if !ok || text == "" {
				panic("missing " + string(lang) + " agent prompt: " + name)
			}
			locale.RegisterAll(map[string]map[locale.Lang]string{english: {lang: text}})
		}
	}
}
