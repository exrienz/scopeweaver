package agent

import (
	"github.com/Autumn-27/artex/locale"
	"log"
	"path/filepath"

	"github.com/Autumn-27/norma/agentcore"
	"github.com/Autumn-27/norma/noaadapter"
)

// noaWarn returns a diagnostics sink tagging non-fatal noa messages with the
// session, routed through the package logger (agents have no per-instance one).
func noaWarn(session string) func(string) {
	return func(msg string) { log.Printf("[noa] %s: %s", session, msg) }
}

// noa is norma v0.4.0's model-driven context compaction, exposed as an experimental
// setting. It is mutually exclusive with built-in compaction. noaadapter.Enable
// installs the Compactor, Compress tool, and three persistent prompt blocks.
// Without Enable, ordinary compaction remains active. Each agent's noaEnabledFn
// is read once per run, so changes affect subsequent runs without rebuilding agents.

// enableNoa adds noa to opts when its resolver enables it. archiveRoot is the
// persistent global workDir; archives share <workDir>/noa instead of task/intent dirs.
// The globally unique sessionID names each archive subdirectory without collisions.
//
// As an experiment, setup failure must not interrupt tasks: warn and retain built-in compaction.
// On success clear opts.Compaction to avoid the warning about two context managers.
func enableNoa(opts *agentcore.Options, enabled func() bool, archiveRoot, sessionID string, onWarn func(string)) *noaadapter.Session {
	if enabled == nil || !enabled() {
		return nil
	}
	if _, local := opts.Compactor.(*localCompactor); local {
		if onWarn != nil {
			onWarn("Native Ollama contexts up to 64K retain the 72% compaction policy; experimental noa is skipped")
		}
		return nil
	}
	if opts.OnWarn == nil {
		opts.OnWarn = onWarn
	}
	// Keep the profile's resolved context window when replacing its compactor.
	// MaxTokens limits output separately; an absent window keeps the SDK fallback.
	window := 0
	if opts.Compaction != nil && opts.Compaction.ContextWindow > 0 {
		window = opts.Compaction.ContextWindow
	}
	session, err := noaadapter.EnableWithSession(opts, noaadapter.Options{
		ArchiveBaseDir:    filepath.Join(archiveRoot, "noa"),
		SessionID:         sessionID,
		OnWarn:            onWarn,
		ModelContextLimit: window,
	})
	if err != nil {
		if onWarn != nil {
			onWarn(locale.Text(locale.ServerDefault(), "Could not enable noa compaction; falling back to built-in compaction: ") + err.Error())
		}
		return nil
	}
	// Compactor supersedes Compaction; explicitly clear the latter to avoid repeated warnings.
	opts.Compaction = nil
	return session
}
