package db

import (
	"fmt"
	"github.com/Autumn-27/artex/locale"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestTaskLanguageAtomicCreation(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	marker := fmt.Sprintf("locale_atomic_%d", time.Now().UnixNano())
	// Fail only this test task's metadata write, within an isolated test database.
	// The trigger does not affect other settings or tasks and is removed on return.
	function := marker + "_fn"
	trigger := marker + "_tr"
	query := fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF NEW.key LIKE 'task_language.%%' AND EXISTS(SELECT 1 FROM tasks WHERE description='%s' AND NEW.key='task_language.'||id::text) THEN
 RAISE EXCEPTION 'injected task language failure'; END IF; RETURN NEW; END $$`, function, marker)
	if _, err = d.Exec(query); err != nil {
		t.Fatal(err)
	}
	defer d.Exec("DROP FUNCTION " + function + "()")
	if _, err = d.Exec("CREATE TRIGGER " + trigger + " BEFORE INSERT ON settings FOR EACH ROW EXECUTE FUNCTION " + function + "()"); err != nil {
		t.Fatal(err)
	}
	defer d.Exec("DROP TRIGGER " + trigger + " ON settings")
	task, err := d.CreateTaskWithOptions(marker, "no network target", TaskCreateOptions{Language: locale.Ko})
	if err == nil || task != nil || !strings.Contains(err.Error(), "injected task language failure") {
		t.Fatalf("expected atomic metadata failure: task=%v err=%v", task, err)
	}
	for _, table := range []string{"tasks", "explorations"} {
		var count int
		if err = d.QueryRow("SELECT count(*) FROM "+table+" WHERE description=$1", marker).Scan(&count); err != nil || count != 0 {
			t.Fatalf("partial %s remained: count=%d err=%v", table, count, err)
		}
	}
}

func TestTaskLanguageSavedAndRemovedWithTask(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	task, err := d.CreateTaskWithOptions("language persistence", "no network target", TaskCreateOptions{Language: locale.Ko})
	if err != nil {
		t.Fatal(err)
	}
	defer d.DeleteTask(task.ID)
	key := "task_language." + strconv.FormatInt(task.ID, 10)
	value, ok, err := d.GetSetting(key)
	if err != nil || !ok || value != "ko" {
		t.Fatalf("language missing: %q %v %v", value, ok, err)
	}
	if err = d.DeleteTask(task.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok, err = d.GetSetting(key); err != nil || ok {
		t.Fatalf("language remained after task deletion: %v %v", ok, err)
	}
}

func TestShippedTaskLanguagePersistenceAndCanonicalValidation(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	raw := "用户原文 한국어"
	for _, language := range locale.Languages() {
		t.Run(string(language), func(t *testing.T) {
			task, err := d.CreateTaskWithOptions(raw, raw, TaskCreateOptions{Language: language})
			if err != nil {
				t.Fatal(err)
			}
			defer d.DeleteTask(task.ID)
			value, ok, err := d.GetSetting("task_language." + strconv.FormatInt(task.ID, 10))
			if err != nil || !ok || value != string(language) {
				t.Fatalf("language %s persistence: %q %v %v", language, value, ok, err)
			}
			if task.Description != raw || task.Goal != raw {
				t.Fatal("user task fields modified")
			}
		})
	}
	for _, input := range []locale.Lang{"zh-CN", "zh-TW", "zh-hant", "ja-JP", "es-MX", "fr", ""} {
		created, err := d.CreateTaskWithOptions("canonical language fixture", raw, TaskCreateOptions{Language: input})
		if input == "" {
			if err != nil {
				t.Fatal(err)
			}
			defer d.DeleteTask(created.ID)
			if _, exists, err := d.GetSetting("task_language." + strconv.FormatInt(created.ID, 10)); err != nil || exists {
				t.Fatal("legacy missing language field changed")
			}
			continue
		}
		if err == nil || created != nil {
			t.Fatalf("noncanonical %q accepted", input)
		}
		if got := locale.ErrorMessage(locale.ZhHans, err); !strings.Contains(got, "语言必须为 "+locale.SupportedLanguageNames()+" 之一") {
			t.Fatalf("validation stale/English: %s", got)
		}
	}
}
