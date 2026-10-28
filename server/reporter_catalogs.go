package server

import (
	"embed"
	"encoding/json"

	"github.com/Autumn-27/artex/locale"
)

//go:embed reporter_zh-Hant.json reporter_ja.json reporter_es.json
var translatedReporterFiles embed.FS

func init() {
	for _, lang := range []locale.Lang{locale.ZhHant, locale.Ja, locale.Es} {
		raw, err := translatedReporterFiles.ReadFile("reporter_" + string(lang) + ".json")
		if err != nil {
			panic(err)
		}
		var translated map[string]string
		if err := json.Unmarshal(raw, &translated); err != nil {
			panic(err)
		}
		if len(translated) != 1 || translated["reporterToolCallMessage"] == "" {
			panic("missing " + string(lang) + " reporter trigger")
		}
		locale.RegisterAll(map[string]map[locale.Lang]string{reporterToolCallMessage: {lang: translated["reporterToolCallMessage"]}})
	}
	locale.RegisterAll(map[string]map[locale.Lang]string{reporterToolCallMessage: {locale.ZhHans: "刚刚通过 report_finding 登记了一个漏洞。读取返回 JSON 中的 finding_id（独立发现记录）和 finding_node_id（探索节点）。用 get_finding_traffic(finding_id) 读取证据列表和 version。空列表也正常，不会阻碍撰写报告。若运行指令开启了自动绑定，请在读取前验证/绑定此发现的流量。节点详情使用 finding_node_id。最后通过 update_finding_report(finding_id=finding_node_id, report, evidence_version=实际读取的版本) 保存。不传 evidence_version，报告会一直显示为过期。不要混用两套 ID。"}})
}
