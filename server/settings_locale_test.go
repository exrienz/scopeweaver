package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/internal/dbtest"
	"github.com/Autumn-27/artex/locale"
)

func TestShippedSettingsPersistenceAndValidation(t *testing.T) {
	dbtest.DSN(t) // skip unless ARTEX_TEST_DSN points at a throwaway DB
	old := locale.ServerDefault()
	defer locale.SetServerDefault(old)
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	previousValue, previousExists, err := m.pg.GetSetting(settingLanguage)
	if err != nil {
		t.Fatal(err)
	}
	previousDefault := locale.ServerDefault()
	// The package shares one disposable DB. Restore both preference layers before
	// closing this manager, including the absence of a pre-existing settings row.
	t.Run("scoped-preference-update", func(t *testing.T) {
		defer func() {
			if previousExists {
				err = m.pg.SetSetting(settingLanguage, previousValue)
			} else {
				_, err = m.pg.Exec("DELETE FROM settings WHERE key=$1", settingLanguage)
			}
			if err != nil {
				t.Errorf("restore durable language preference: %v", err)
			}
			locale.SetServerDefault(previousDefault)
		}()
		s := &Server{m: m}
		handler := withLocale(http.HandlerFunc(s.putSettings))
		for _, language := range locale.Languages() {
			for _, tc := range []struct {
				body   string
				status int
			}{
				{`{"language":"` + string(language) + `"}`, 200}, {`{}`, 200}, {`{"language":null}`, 200},
				{`{"language":""}`, 400}, {`{"language":"zh-CN"}`, 400}, {`{"language":"zh-TW"}`, 400}, {`{"language":"zh-hant"}`, 400},
				{`{"language":"ja-JP"}`, 400}, {`{"language":"es-MX"}`, 400}, {`{"language":"fr"}`, 400},
			} {
				r := httptest.NewRequest("PUT", "/api/settings?lang=zh-Hans", strings.NewReader(tc.body))
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, r)
				if w.Code != tc.status {
					t.Fatalf("%s: %d %s", tc.body, w.Code, w.Body)
				}
				stored, ok, err := m.pg.GetSetting(settingLanguage)
				if err != nil || !ok || stored != string(language) || locale.ServerDefault() != language {
					t.Fatalf("preference changed: %q %v %v", stored, ok, err)
				}
				if tc.status == 400 && !strings.Contains(w.Body.String(), "语言必须为 "+locale.SupportedLanguageNames()+" 之一") {
					t.Fatalf("invalid localized response: %s", w.Body)
				}
			}
			locale.SetServerDefault(locale.En)
			t.Setenv("SCOPEWEAVER_LANGUAGE", "ko")
			if err := loadLanguage(m.pg); err != nil {
				t.Fatal(err)
			}
			if locale.ServerDefault() != language {
				t.Fatal("durable language preference did not win over env after reload")
			}
		}
	})
	stored, exists, err := m.pg.GetSetting(settingLanguage)
	if err != nil || stored != previousValue || exists != previousExists || locale.ServerDefault() != previousDefault {
		t.Fatalf("locale fixture leaked state: stored=%q exists=%v default=%s err=%v", stored, exists, locale.ServerDefault(), err)
	}
	// Reload the same DB as subsequent package tests do; restoring only the atomic
	// default would pass an immediate read but load the leaked zh-Hans row here.
	next, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	if locale.ServerDefault() != previousDefault {
		t.Fatal("subsequent manager reloaded a leaked language preference")
	}
}
