package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/locale"
	"github.com/Autumn-27/norma/llm"
)

func gh50Tools(t *testing.T, d *db.DB, lang locale.Lang) (*ToolSet, *db.Task) {
	t.Helper()
	task, err := d.CreateTaskWithOptions("gh50 backend", "isolated fixture", db.TaskCreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.DeleteTask(task.ID) })
	tools := NewToolSet(d.Exploration(task.ExplorationID), "planner", lang)
	tools.SetTaskID(task.ID)
	tools.SetAssetStore(d.Assets(), d.Companies())
	return tools, task
}

func TestGH50InsertAssetsAllowSuffix(t *testing.T) {
	d := testDB(t)
	defer d.Close()
	tools, task := gh50Tools(t, d, locale.En)
	as := d.Assets()
	if _, err := as.CreateTaskInterceptRule(task.ID, "allow", "fuzzy_domain", "juice-shop.internal", "", true); err != nil {
		t.Fatal(err)
	}
	defer d.Exec(`DELETE FROM assets WHERE domain IN ('juice-shop.internal2','a.juice-shop.internal','juice-shop.internal')`)
	for _, tc := range []struct {
		kind, domain string
		want         bool
	}{{"root_domain", "juice-shop.internal2", false}, {"subdomain", "a.juice-shop.internal", true}} {
		t.Run(tc.domain, func(t *testing.T) {
			out := callInsertAssets(t, tools, map[string]any{"assets": []any{map[string]any{"type": tc.kind, "domain": tc.domain}}})
			var n int
			if err := d.QueryRow(`SELECT count(*) FROM assets WHERE domain=$1`, tc.domain).Scan(&n); err != nil {
				t.Fatal(err)
			}
			want := 0
			if tc.want {
				want = 1
			}
			if n != want {
				t.Errorf("asset rows=%d want=%d output=%v", n, want, out)
			}
			rows, err := as.ListTaskScope(task.ID)
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != want {
				t.Errorf("scope rows=%+v want count=%d", rows, want)
			}
		})
	}
}

func TestGH50AddTaskScopeToolLocale(t *testing.T) {
	d := testDB(t)
	defer d.Close()
	for _, tc := range []struct {
		lang locale.Lang
		text string
	}{{locale.En, "allowlist"}, {locale.Ko, "허용 범위"}, {locale.ZhHans, "白名单"}} {
		t.Run(string(tc.lang), func(t *testing.T) {
			tools, task := gh50Tools(t, d, tc.lang)
			if _, err := d.Assets().CreateTaskInterceptRule(task.ID, "allow", "fuzzy_domain", "example.org", "", true); err != nil {
				t.Fatal(err)
			}
			ctx := locale.WithLang(context.Background(), tc.lang)
			out, err := tools.addTaskScope().Call(ctx, json.RawMessage(`{"kind":"subdomain","value":"blocked.example","reason":"fixture"}`), nil)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.Flatten(), tc.text) {
				t.Errorf("missing localized rejection %q: %s", tc.text, out.Flatten())
			}
			rows, err := d.Assets().ListTaskScope(task.ID)
			if err != nil || len(rows) != 0 {
				t.Errorf("rejection persisted rows=%+v err=%v", rows, err)
			}
		})
	}
}

func TestGH50AddIntentPersistedGate(t *testing.T) {
	d := testDB(t)
	defer d.Close()
	for _, tc := range []struct {
		name, domain, root, allow string
		want                      bool
	}{
		{"suffix collision", "a.juice-shop.internal2", "juice-shop.internal2", "juice-shop.internal", false},
		{"subdomain", "api.juice-shop.internal", "juice-shop.internal", "juice-shop.internal", true},
		{"government label", "x.government.com", "government.com", "government.com", true},
		{"root only", "", "juice-shop.internal", "juice-shop.internal", true},
		{"root only collision", "", "juice-shop.internal2", "juice-shop.internal", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tools, task := gh50Tools(t, d, locale.En)
			if _, err := d.Assets().CreateTaskInterceptRule(task.ID, "allow", "fuzzy_domain", tc.allow, "", true); err != nil {
				t.Fatal(err)
			}
			var id int64
			if err := d.QueryRow(`INSERT INTO assets(type,domain,root_domain) VALUES ('subdomain',NULLIF($1,''),$2) RETURNING id`, tc.domain, tc.root).Scan(&id); err != nil {
				t.Fatal(err)
			}
			defer d.Assets().DeleteByIDs([]int64{id})
			raw, _ := json.Marshal(id)
			_, err := tools.addOneIntent(intentItem{Summary: "gh50 " + tc.name, AssetIDs: []json.RawMessage{raw}})
			if (err == nil) != tc.want {
				t.Errorf("error=%v want accepted=%v", err, tc.want)
			}
			var n int
			if err := d.QueryRow(`SELECT count(*) FROM exploration_nodes WHERE exploration_id=$1 AND kind='intent'`, task.ExplorationID).Scan(&n); err != nil {
				t.Fatal(err)
			}
			want := 0
			if tc.want {
				want = 1
			}
			if n != want {
				t.Errorf("intent rows=%d want=%d", n, want)
			}
		})
	}
}

func TestGH50WorkerScopeRecheck(t *testing.T) {
	d := testDB(t)
	defer d.Close()
	for _, mode := range []string{"block", "allow", "global read error", "task read error", "coverage disabled"} {
		t.Run(mode, func(t *testing.T) {
			_, task := gh50Tools(t, d, locale.En)
			as := d.Assets()
			ids := []int64{}
			for _, domain := range []string{"a.gh50-worker.example", "b.gh50-worker.example"} {
				id, err := as.UpsertSubdomain(db.UpsertSubdomainReq{Domain: domain})
				if err != nil {
					t.Fatal(err)
				}
				ids = append(ids, id)
			}
			defer as.DeleteByIDs(ids)
			ts := d.Exploration(task.ExplorationID)
			intentID, err := ts.AddNode("intent", map[string]any{"summary": "local worker fixture", "asset_ids": ids}, 0, "running", "planner", ids)
			if err != nil {
				t.Fatal(err)
			}
			intent, err := ts.GetNode(intentID)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "block" {
				if _, err := as.CreateTaskInterceptRule(task.ID, "block", "exact_domain", "a.gh50-worker.example", "", true); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "allow" {
				if _, err := as.CreateTaskInterceptRule(task.ID, "allow", "exact_domain", "b.gh50-worker.example", "", true); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "coverage disabled" {
				if _, err := d.Exec(`UPDATE tasks SET coverage_enabled=false WHERE id=$1`, task.ID); err != nil {
					t.Fatal(err)
				}
			}
			if strings.Contains(mode, "read error") {
				table := "asset_intercept_rules"
				if mode == "task read error" {
					table = "task_intercept_rules"
				}
				if _, err := d.Exec(`ALTER TABLE ` + table + ` RENAME TO gh50_hidden_rules`); err != nil {
					t.Fatal(err)
				}
				defer func() {
					if _, err := d.Exec(`ALTER TABLE gh50_hidden_rules RENAME TO ` + table); err != nil {
						t.Error(err)
					}
				}()
			}
			provider := captureUsageProvider{stream: func(_ context.Context, yield func(llm.StreamEvent, error) bool) {
				yield(llm.StreamEvent{Type: llm.SETextDelta, Text: "done"}, nil)
				yield(llm.StreamEvent{Type: llm.SEMessageDelta, StopReason: "end_turn"}, nil)
			}}
			w := NewWorker(provider, "fixture", t.TempDir(), nil, 0, 3)
			if _, _, err := w.Execute(context.Background(), "gh50-worker", task.ID, as, ts, intent, nil, nil, nil, nil); err != nil {
				t.Fatal(err)
			}
			rows, err := as.ListTaskScope(task.ID)
			if err != nil {
				t.Fatal(err)
			}
			want := 0
			if mode == "block" || mode == "allow" {
				want = 1
			}
			if len(rows) != want {
				t.Errorf("scope rows=%+v want count=%d", rows, want)
			}
			if want == 1 && len(rows) == 1 && rows[0].Domain != "b.gh50-worker.example" {
				t.Errorf("wrong asset scoped: %+v", rows)
			}
		})
	}
}

func TestGH50ScopeRuleReadFailure(t *testing.T) {
	d := testDB(t)
	defer d.Close()
	for _, table := range []string{"asset_intercept_rules", "task_intercept_rules"} {
		t.Run(table, func(t *testing.T) {
			tools, task := gh50Tools(t, d, locale.Ko)
			if _, err := d.Exec(`ALTER TABLE ` + table + ` RENAME TO gh50_hidden_rules`); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if _, err := d.Exec(`ALTER TABLE gh50_hidden_rules RENAME TO ` + table); err != nil {
					t.Error(err)
				}
			}()
			out := callReadJSON(t, tools.addTaskScope(), `{"kind":"subdomain","value":"safe.example","reason":"fixture"}`)
			if !strings.Contains(fmt.Sprint(out), "errors") {
				t.Errorf("rule read failure must reject: %v", out)
			}
			rows, err := d.Assets().ListTaskScope(task.ID)
			if err != nil || len(rows) != 0 {
				t.Errorf("failure wrote scope rows=%+v err=%v", rows, err)
			}
		})
	}
}

func TestGH50ScopeWording(t *testing.T) {
	for _, tc := range []struct {
		lang              locale.Lang
		boundary, minimum string
	}{
		{locale.En, "not a network boundary", "never expand"},
		{locale.Ko, "네트워크 경계가 아", "확장하지 마세요"},
		{locale.ZhHans, "不是网络边界", "绝不自行扩大"},
	} {
		t.Run(string(tc.lang), func(t *testing.T) {
			for name, text := range map[string]string{
				"prompt": BuiltinPromptText(goalsScopeTail, tc.lang),
				"tool":   NewToolSet(nil, "", tc.lang).addTaskScope().Description(),
			} {
				if !strings.Contains(text, tc.boundary) || !strings.Contains(text, "Interception") || !strings.Contains(strings.ToLower(text), tc.minimum) {
					t.Errorf("%s does not explain scope and minimum scope in %s: %s", name, tc.lang, text)
				}
			}
		})
	}
}
