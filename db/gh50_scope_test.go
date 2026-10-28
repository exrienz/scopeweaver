package db

import (
	"fmt"
	"testing"
)

func TestGH50EvaluateAssetGateDomainLabels(t *testing.T) {
	for _, tc := range []struct {
		pattern, domain string
		allow, want     bool
	}{
		{".gov", "x.gov", false, false},
		{".gov", "x.gov.uk", false, false},
		{".gov", "a.government.com", false, true},
		{" .GOV. ", "X.GOV.UK", false, false},
		{"...", "example.com", false, true},
		{"juice-shop.internal", "juice-shop.internal", true, true},
		{"juice-shop.internal", "a.juice-shop.internal", true, true},
		{"juice-shop.internal", "juice-shop.internal2", true, false},
		{"juice-shop.internal", "juice-shop.internal.attacker.com", true, false},
		{" .JUICE-SHOP.INTERNAL. ", "A.JUICE-SHOP.INTERNAL", true, true},
	} {
		t.Run(fmt.Sprintf("%s/%s/allow=%v", tc.pattern, tc.domain, tc.allow), func(t *testing.T) {
			rules := []AssetInterceptRule{{Kind: "fuzzy_domain", Pattern: tc.pattern, Enabled: true}}
			var block, allow []AssetInterceptRule
			if tc.allow {
				allow = rules
			} else {
				block = rules
			}
			if got := EvaluateAssetGate(block, allow, []string{tc.domain}, nil, nil); got.Allowed != tc.want {
				t.Fatalf("allowed=%v want=%v: %+v", got.Allowed, tc.want, got)
			}
		})
	}
	// Membership in the block set wins even when Action claims otherwise.
	r := AssetInterceptRule{Kind: "fuzzy_domain", Pattern: "internal", Enabled: true, Action: "allow"}
	if EvaluateAssetGate([]AssetInterceptRule{r}, []AssetInterceptRule{r}, []string{"a.internal"}, nil, nil).Allowed {
		t.Fatal("block set must take precedence regardless of Action")
	}
}

func TestGH50AgentScopeGate(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	companyID, _, err := d.Companies().UpsertCompany("gh50-scope-company", "")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Companies().DeleteCompany(companyID)
	for _, tc := range []struct {
		name, kind, value, allowKind, allow, blockKind, block, source string
		want                                                          bool
	}{
		{name: "outside", kind: "subdomain", value: "outside.example", allowKind: "fuzzy_domain", allow: "example.com"},
		{name: "normalized root", kind: "root_domain", value: "a.example.com", allowKind: "fuzzy_domain", allow: "a.example.com"},
		{name: "allowed CIDR", kind: "cidr", value: "10.1.0.0/16", allowKind: "cidr", allow: "10.0.0.0/8", want: true},
		{name: "last outside", kind: "cidr", value: "10.1.0.0/16", allowKind: "cidr", allow: "10.1.0.0/24"},
		{name: "first outside", kind: "cidr", value: "10.1.0.0/16", allowKind: "cidr", allow: "10.1.255.0/24"},
		{name: "interior block", kind: "cidr", value: "10.1.0.0/16", allowKind: "cidr", allow: "10.0.0.0/8", blockKind: "cidr", block: "10.1.2.0/24", want: true},
		{name: "first blocked", kind: "cidr", value: "10.1.0.0/16", blockKind: "exact_ip", block: "10.1.0.0"},
		{name: "last blocked", kind: "cidr", value: "10.1.0.0/16", blockKind: "exact_ip", block: "10.1.255.255"},
		{name: "IPv6 last outside", kind: "cidr", value: "2001:db8::/64", allowKind: "cidr", allow: "2001:db8::/120"},
		{name: "IPv6 allowed", kind: "cidr", value: "2001:db8::/64", allowKind: "cidr", allow: "2001:db8::/32", want: true},
		{name: "IP allowed", kind: "ip", value: "10.1.2.3:443", allowKind: "exact_ip", allow: "10.1.2.3", want: true},
		{name: "company", kind: "company", value: fmt.Sprint(companyID), allowKind: "fuzzy_domain", allow: "example.com"},
		{name: "keyword", kind: "keyword", value: "Scope Company", allowKind: "fuzzy_domain", allow: "example.com"},
		{name: "icp", kind: "icp", value: "京ICP备123456号", allowKind: "fuzzy_domain", allow: "example.com"},
		{name: "global block without allow", kind: "subdomain", value: "x.gov"},
		{name: "unblocked without allow", kind: "subdomain", value: "other.example", want: true},
		{name: "keyword without allow", kind: "keyword", value: "Scope Company", want: true},
		{name: "manual bypass", kind: "subdomain", value: "x.gov", allowKind: "fuzzy_domain", allow: "example.com", source: "manual", want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			task, err := d.CreateTaskWithOptions("gh50 scope", "local fixture", TaskCreateOptions{})
			if err != nil {
				t.Fatal(err)
			}
			defer d.DeleteTask(task.ID)
			as := d.Assets()
			if tc.allow != "" {
				if _, err := as.CreateTaskInterceptRule(task.ID, "allow", tc.allowKind, tc.allow, "", true); err != nil {
					t.Fatal(err)
				}
			}
			if tc.block != "" {
				if _, err := as.CreateTaskInterceptRule(task.ID, "block", tc.blockKind, tc.block, "", true); err != nil {
					t.Fatal(err)
				}
			}
			_, err = as.AddAgentScope(task.ID, tc.kind, tc.value, "fixture", tc.source)
			if (err == nil) != tc.want {
				t.Errorf("error=%v, want accepted=%v", err, tc.want)
			}
			rows, e := as.ListTaskScope(task.ID)
			if e != nil {
				t.Fatal(e)
			}
			want := 0
			if tc.want {
				want = 1
			}
			if len(rows) != want {
				t.Errorf("scope rows=%+v want count=%d", rows, want)
			}
		})
	}
}

func TestGH50ScopeReadback(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	source, err := d.CreateTaskWithOptions("gh50 source", "fixture", TaskCreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer d.DeleteTask(source.ID)
	task, err := d.CreateTaskWithOptions("gh50 current", "fixture", TaskCreateOptions{SourceTaskIDs: []int64{source.ID}})
	if err != nil {
		t.Fatal(err)
	}
	defer d.DeleteTask(task.ID)
	as := d.Assets()
	for _, entry := range []struct {
		task   int64
		domain string
	}{{source.ID, "source.example"}, {task.ID, "manual.example"}} {
		if _, err := as.AddAgentScope(entry.task, "subdomain", entry.domain, "retained", "manual"); err != nil {
			t.Fatal(err)
		}
	}
	ids := []int64{}
	defer func() { _, _ = as.DeleteByIDs(ids) }()
	for _, domain := range []string{"a.example.org", "b.blocked.example", "anchored.example"} {
		var id int64
		root, _ := RootDomain(domain)
		err := d.QueryRow(`INSERT INTO assets(type,domain,root_domain) VALUES ('subdomain',$1,$2) RETURNING id`, domain, root).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if _, err := d.Exploration(task.ExplorationID).AddNode(KindFact, map[string]any{"summary": "anchored"}, 0, "confirmed", "worker", []int64{ids[2]}); err != nil {
		t.Fatal(err)
	}
	if _, err := as.CreateTaskInterceptRule(task.ID, "allow", "fuzzy_domain", "example.org", "", true); err != nil {
		t.Fatal(err)
	}
	assertAssets := func(want bool) {
		t.Helper()
		rows, err := as.QueryDSLInScope("", "subdomain", task.ID, 100, 0)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[int64]bool{}
		for _, a := range rows {
			seen[a.ID] = true
		}
		if seen[ids[0]] != want || seen[ids[1]] || seen[ids[2]] {
			t.Errorf("scope query=%v want accepted present=%v, rejected/anchor absent", seen, want)
		}
	}
	assertAssets(false)
	if _, err := as.AddAgentScope(task.ID, "root_domain", "example.org", "fixture", "agent"); err != nil {
		t.Fatal(err)
	}
	assertAssets(true)
	if _, err := as.AddAgentScope(task.ID, "root_domain", "blocked.example", "fixture", "agent"); err == nil {
		t.Error("rejected scope saved")
	}
	assertAssets(true)
	rows, err := as.ListTaskScopeWithSources(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("retained manual, source, agent rows=%+v", rows)
	}
	seen := map[string]bool{}
	for _, row := range rows {
		seen[fmt.Sprintf("%d/%s/%s", row.TaskID, row.Source, row.Domain)] = true
	}
	for _, key := range []string{fmt.Sprintf("%d/manual/manual.example", task.ID), fmt.Sprintf("%d/manual/source.example", source.ID), fmt.Sprintf("%d/agent/example.org", task.ID)} {
		if !seen[key] {
			t.Errorf("missing provenance %s: %+v", key, rows)
		}
	}
	cov, err := as.TaskCoverageWithSources(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	scoped, err := as.QueryDSLInScope("", "", task.ID, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if cov.Denominator != len(scoped)+1 || cov.Tested != 1 {
		t.Errorf("accepted scope plus independent fact anchor: %+v", cov)
	}
}
