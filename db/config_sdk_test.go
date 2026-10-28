package db

import (
	"fmt"
	"testing"
)

func TestSDKProfileCredentialTransitions(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	cases := []struct{ name, oldFormat, oldURL, newFormat, newURL, key, want string }{
		{"SDK same endpoint", "opencode-sdk", "http://localhost:4097/a", "opencode-sdk", "http://localhost:4097/a/", "", "old-token"},
		{"SDK default port casing", "opencode-sdk", "HTTP://LOCALHOST:80/a/", "opencode-sdk", "http://localhost/a", "", "old-token"},
		{"SDK different path", "opencode-sdk", "http://localhost:4097/a", "opencode-sdk", "http://localhost:4097/b", "", ""},
		{"SDK different host alias", "opencode-sdk", "http://localhost:4097", "opencode-sdk", "http://127.0.0.1:4097", "", ""},
		{"SDK different port", "opencode-sdk", "http://localhost:4097", "opencode-sdk", "http://localhost:4098", "", ""},
		{"SDK to cloud", "opencode-sdk", "http://localhost:4097", "openai", "https://api.openai.com/v1", "", ""},
		{"cloud to SDK", "openai", "https://api.openai.com/v1", "opencode-sdk", "http://localhost:4097", "", ""},
		{"SDK explicit replacement", "opencode-sdk", "http://localhost:4097/a", "opencode-sdk", "http://localhost:4097/b", "new-token", "new-token"},
		{"cloud unchanged behavior", "openai", "https://one.example/v1", "openai", "https://two.example/v1", "", "old-token"},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &LLMProfile{Name: fmt.Sprintf("sdk-token-transition-%d", i), Format: tc.oldFormat, BaseURL: tc.oldURL, Model: "opencode/test", APIKey: "old-token", Streaming: true}
			id, e := d.SaveProfile(p)
			if e != nil {
				t.Fatal(e)
			}
			defer d.Exec(`DELETE FROM llm_profiles WHERE id=$1`, id)
			p.ID = id
			p.Format = tc.newFormat
			p.BaseURL = tc.newURL
			p.APIKey = tc.key
			p.APIKeyHint = "stale-hint"
			if _, e = d.SaveProfile(p); e != nil {
				t.Fatal(e)
			}
			got, e := d.ProfileByID(id)
			if e != nil {
				t.Fatal(e)
			}
			var savedHint string
			if e = d.QueryRow(`SELECT COALESCE(api_key_hint,'') FROM llm_profiles WHERE id=$1`, id).Scan(&savedHint); e != nil {
				t.Fatal(e)
			}
			wantHint := ""
			if tc.want != "" {
				wantHint = "…" + tc.want[len(tc.want)-4:]
			}
			if got.APIKey != tc.want || savedHint != wantHint || got.Format != tc.newFormat || got.BaseURL != tc.newURL {
				t.Fatalf("saved=%+v want key=%q hint=%q", got, tc.want, wantHint)
			}
		})
	}
}

func TestSDKFormatUpgradePreservesExistingProfiles(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	old := &LLMProfile{Name: "sdk-upgrade-existing", Format: "openai", Model: "existing-model", BaseURL: "https://existing.example/v1", APIKey: "existing-key"}
	id, err := d.SaveProfile(old)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Exec(`DELETE FROM llm_profiles WHERE id=$1`, id)
	// Recreate the previous release's constraint only inside this disposable test database.
	if _, err = d.Exec(`ALTER TABLE llm_profiles DROP CONSTRAINT llm_profiles_format_check; ALTER TABLE llm_profiles ADD CONSTRAINT llm_profiles_format_check CHECK (format IN ('openai','anthropic','openai-responses','ollama'))`); err != nil {
		t.Fatal(err)
	}
	sdk := &LLMProfile{Name: "sdk-upgrade-new", Format: "opencode-sdk", Model: "opencode/test", BaseURL: "http://localhost:4097", APIKey: "sdk-token"}
	if _, err = d.SaveProfile(sdk); err == nil {
		t.Fatal("previous schema unexpectedly accepted SDK format")
	}
	upgraded, err := Open(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Close()
	sdkID, err := upgraded.SaveProfile(sdk)
	if err != nil {
		t.Fatal(err)
	}
	defer upgraded.Exec(`DELETE FROM llm_profiles WHERE id=$1`, sdkID)
	got, err := upgraded.ProfileByID(id)
	if err != nil || got.APIKey != old.APIKey || got.Format != old.Format || got.Model != old.Model || got.BaseURL != old.BaseURL {
		t.Fatalf("existing profile changed: %+v err=%v", got, err)
	}
}
