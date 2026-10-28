package report

import (
	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/locale"
	"strings"
	"testing"
	"time"
)

func TestReportLanguagesPreserveUserContent(t *testing.T) {
	f := &db.DBFinding{ID: 1, Name: "用户原文 한국어", Summary: "<raw> 用户摘要", Evidence: "GET /中文 HTTP/1.1", Report: "# 用户报告", Severity: "high", Status: "pending"}
	en := FindingsMarkdown([]*db.DBFinding{f}, time.Unix(0, 0), locale.En)
	ko := FindingsMarkdown([]*db.DBFinding{f}, time.Unix(0, 0), locale.Ko)
	if !strings.Contains(en, "# Vulnerability findings report") || !strings.Contains(ko, "# 취약점 발견 보고서") {
		t.Fatalf("missing translated headings: %q / %q", en, ko)
	}
	for _, content := range []string{f.Name, f.Summary, f.Evidence, f.Report} {
		if !strings.Contains(en, content) || !strings.Contains(ko, content) {
			t.Fatalf("stored content changed: %q", content)
		}
	}
	if strings.Contains(en, "%!") || strings.Contains(ko, "%!") {
		t.Fatal("invalid report interpolation")
	}
	if !strings.Contains(string(FindingsCSV([]*db.DBFinding{f}, locale.Ko)), "트래픽 증거 수") {
		t.Fatal("CSV headings not localized")
	}
}

func TestEmptyReportLanguages(t *testing.T) {
	for _, tc := range []struct {
		l    locale.Lang
		want string
	}{{locale.En, "No vulnerabilities were confirmed."}, {locale.Ko, "확인된 취약점이 없습니다."}} {
		text := Markdown(Input{Language: tc.l, GeneratedAt: time.Unix(0, 0), Goal: "保留目标"})
		if !strings.Contains(text, tc.want) || !strings.Contains(text, "保留目标") {
			t.Fatalf("wrong language or altered goal: %q", text)
		}
	}
}

func TestChineseReportsLocalizeLabelsAndPreserveEvidence(t *testing.T) {
	raw := "用户原文 한국어 <raw> 100%"
	f := &db.DBFinding{ID: 7, Name: raw, Summary: raw, Evidence: raw, Report: raw, Severity: "high", Status: "pending"}
	text := FindingsMarkdown([]*db.DBFinding{f}, time.Unix(0, 0), locale.ZhHans)
	for _, want := range []string{raw, "漏洞", "高", "待确认"} {
		if !strings.Contains(text, want) {
			t.Fatalf("Chinese report missing %q", want)
		}
	}
	if strings.Contains(text, "%!") {
		t.Fatal("invalid Chinese report interpolation")
	}
	csv := string(FindingsCSV([]*db.DBFinding{f}, locale.ZhHans))
	if !strings.Contains(csv, "high") || !strings.Contains(csv, "pending") || !strings.Contains(csv, raw) {
		t.Fatal("CSV machine values/raw evidence changed")
	}
	if severityLabel(locale.ZhHans, "custom") != "CUSTOM" || statusLabel(locale.ZhHans, "custom") != "custom" {
		t.Fatal("unknown report fallback changed")
	}
}

func TestAllShippedReportLanguagesPreserveEvidenceAndMachineValues(t *testing.T) {
	raw := "original evidence 原文 / 원문 <raw> 100% {unchanged}"
	f := &db.DBFinding{ID: 7, Name: raw, Summary: raw, Evidence: raw, Report: raw, Severity: "high", Status: "pending"}
	for _, lang := range locale.Languages() {
		t.Run(string(lang), func(t *testing.T) {
			text := FindingsMarkdown([]*db.DBFinding{f}, time.Unix(0, 0), lang)
			heading, ok := locale.Raw(lang, "# Vulnerability findings report\n\n")
			if !ok || !strings.HasPrefix(text, heading) || !strings.Contains(text, raw) || strings.Contains(text, "%!") {
				t.Fatal("report localization or content preservation failed")
			}
			csv := string(FindingsCSV([]*db.DBFinding{f}, lang))
			if !strings.Contains(csv, "high") || !strings.Contains(csv, "pending") || !strings.Contains(csv, raw) {
				t.Fatal("CSV machine values or raw evidence changed")
			}
			if severityLabel(lang, "custom") != "CUSTOM" || statusLabel(lang, "custom") != "custom" {
				t.Fatal("unknown report value fallback changed")
			}
		})
	}
}

func TestFindingReportInfoAndEmptySeverityLocalization(t *testing.T) {
	for _, lang := range []struct {
		id   locale.Lang
		info string
	}{{locale.En, "INFO"}, {locale.Ko, "정보"}, {locale.ZhHans, "信息"}} {
		for _, severity := range []string{"info", "", "custom"} {
			t.Run(string(lang.id)+"/"+severity, func(t *testing.T) {
				raw := "原始证据 / 원문"
				finding := &db.DBFinding{ID: 23, Name: raw, Summary: raw, Evidence: raw, Severity: severity, Status: "pending"}
				want := lang.info
				if severity == "custom" {
					want = "CUSTOM"
				}
				for name, text := range map[string]string{
					"findings": FindingsMarkdown([]*db.DBFinding{finding}, time.Unix(0, 0), lang.id),
					"finding":  SingleFindingMarkdown(finding, time.Unix(0, 0), lang.id),
				} {
					if !strings.Contains(text, "["+want+"] "+raw) {
						t.Errorf("%s report heading lacks [%s]: %s", name, want, text)
					}
					if !strings.Contains(text, raw) || finding.Severity != severity {
						t.Fatal("raw finding content/severity changed")
					}
				}
			})
		}
		// Raw helper callers historically retain the empty/unknown fallback; report
		// renderers choose info for an empty stored severity before asking for a label.
		if severityLabel(lang.id, "") != "" || severityLabel(lang.id, "custom") != "CUSTOM" {
			t.Errorf("%s raw fallback changed", lang.id)
		}
	}
}
