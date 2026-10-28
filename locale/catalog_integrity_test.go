package locale

import (
	"regexp"
	"strings"
	"testing"
)

func TestProductionCatalogCompleteAndFormatCompatible(t *testing.T) {
	if len(catalog) < 1888 {
		t.Fatalf("production catalog unexpectedly small: %d", len(catalog))
	}
	keys := make([]string, 0, len(catalog))
	for key := range catalog {
		if key != "test.hello" {
			keys = append(keys, key)
		}
	}
	t.Logf("checking %d registered keys in %v", len(keys), Languages())
	if err := ValidateTranslations(keys); err != nil {
		t.Fatal(err)
	}
}

func TestFormatArgumentsRecognizesReorderingAndWidth(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{"%s %d %%", "1:s,2:d"},
		{"%[2]d %[1]s", "1:s,2:d"},
		{"%[3]*.[2]*[1]f", "1:f,2:width,3:width"},
	} {
		if got := strings.Join(formatArguments(tc.input), ","); got != tc.want {
			t.Errorf("%q: got %s, want %s", tc.input, got, tc.want)
		}
	}
}

// step_ids is the executable array argument of the trace tools, not prose to localize.
func stepTraceGuidancePreservesArrayArgument(text string) bool {
	return regexp.MustCompile(`\bstep_ids\b`).MatchString(text)
}

func TestBuiltinStepTraceGuidancePreservesArrayArgument(t *testing.T) {
	keys := []string{
		"Do not fetch the remaining steps if this is enough. If more detail is necessary, call again with the omitted step_ids.",
		"Locate step_ids using mode 1/2, then retrieve full output with mode 3. Thinking steps are excluded. Directly associated tasks' historical traces are supported with source_task_id/inherited=true and are read-only.",
		"step_ids from summary/search to retrieve in full, at most 5; extra IDs appear in omitted_step_ids",
	}
	for _, lang := range Languages() {
		for _, key := range keys {
			text, ok := Raw(lang, key)
			if !ok || !stepTraceGuidancePreservesArrayArgument(text) {
				t.Errorf("%s trace guidance lost step_ids array argument: %q", lang, key)
			}
		}
	}
	// Reproduce the reviewed singular-argument mutation without touching a live catalog.
	if stepTraceGuidancePreservesArrayArgument("Retrieve the omitted step_id; see omitted_step_ids.") {
		t.Fatal("singular step_id fixture incorrectly accepted as the step_ids argument")
	}
}
