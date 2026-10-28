package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Autumn-27/norma/agentcore"
	"github.com/Autumn-27/norma/compaction"
	"github.com/Autumn-27/norma/llm"
)

func TestNoaRetainsConfiguredModelContextWindow(t *testing.T) {
	for _, windowK := range []int{40, 128, 200, 250, 400, 1000} {
		t.Run(fmt.Sprintf("%dK", windowK), func(t *testing.T) {
			profile := Config{Format: llm.FormatOpenAI, ContextWindowK: windowK, MaxTokens: 8192}
			window := profile.CompactionWindow()
			if window != windowK*1000 {
				t.Fatalf("profile context = %d, want %d", window, windowK*1000)
			}
			opts := agentcore.Options{Compaction: compactionConfig(window), MaxTokens: profile.MaxTokens}
			applyRuntimePolicy(&opts, RuntimePolicy{ContextWindow: window})
			session := enableNoa(&opts, func() bool { return true }, t.TempDir(), "configured-window", nil)
			if session == nil {
				t.Fatal("Noa did not initialize")
			}
			if got := session.Config().ModelContextLimit; got != window {
				t.Fatalf("Noa context = %d, want configured %d", got, window)
			}
			if opts.Compaction != nil || opts.Compactor == nil {
				t.Fatal("Noa did not replace built-in compaction")
			}
			if opts.MaxTokens != 8192 {
				t.Fatalf("output limit changed: %d", opts.MaxTokens)
			}
		})
	}
}

func TestNoaContextWindowFallback(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  *compaction.Config
	}{
		{"absent", nil},
		{"unset", &compaction.Config{}},
		{"negative", &compaction.Config{ContextWindow: -1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := agentcore.Options{Compaction: tc.cfg}
			session := enableNoa(&opts, func() bool { return true }, t.TempDir(), "fallback", nil)
			if session == nil || session.Config().ModelContextLimit != 200000 {
				t.Fatalf("missing context must retain the 200K fallback: session=%v", session)
			}
		})
	}
}

func TestNoaDisabledPreservesConfiguredWindow(t *testing.T) {
	for _, enabled := range []func() bool{nil, func() bool { return false }} {
		root := t.TempDir()
		configured := compactionConfig(1000000)
		opts := agentcore.Options{Compaction: configured, MaxTokens: 4096}
		if session := enableNoa(&opts, enabled, root, "disabled", nil); session != nil {
			t.Fatal("disabled Noa initialized a session")
		}
		if opts.Compaction != configured || opts.Compactor != nil || len(opts.Tools) != 0 || opts.MaxTokens != 4096 {
			t.Fatal("disabled Noa changed existing context or output policy")
		}
		if _, err := os.Stat(filepath.Join(root, "noa")); !os.IsNotExist(err) {
			t.Fatalf("disabled Noa created archive state: %v", err)
		}
	}
}

func TestNoaStateRecoveryRetainsConfiguredWindow(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, "noa", "recovery")
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "state.json"), []byte("{invalid json"), 0600); err != nil {
		t.Fatal(err)
	}
	configured := compactionConfig(250000)
	opts := agentcore.Options{Compaction: configured, MaxTokens: 4096}
	warning := ""
	session := enableNoa(&opts, func() bool { return true }, root, "recovery", func(s string) { warning = s })
	if session == nil || warning == "" {
		t.Fatal("damaged state must report its recovery warning and initialize Noa")
	}
	if session.Config().ModelContextLimit != 250000 || opts.MaxTokens != 4096 {
		t.Fatal("state recovery changed the configured context or output limit")
	}
}
