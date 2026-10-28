package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Autumn-27/artex/internal/dbtest"
	"github.com/Autumn-27/artex/selfupdate"
	"testing"

	"github.com/Autumn-27/artex/agent"
)

func TestMain(m *testing.M) {
	// The serve-failure child runs while its parent holds the shared lock.
	// Acquiring it again here would deadlock the parent waiting for that child.
	if os.Getenv("SCOPEWEAVER_TEST_SERVE_FAILURE_CHILD") == "1" {
		os.Exit(m.Run())
	}
	os.Exit(dbtest.Main(m))
}

func TestDatabaseSuiteOwnsSharedLock(t *testing.T) {
	conn, err := sql.Open("pgx", dbtest.DSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	c, err := conn.Conn(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var acquired bool
	if err := c.QueryRowContext(t.Context(), "SELECT pg_try_advisory_lock($1)", int64(7337741002)).Scan(&acquired); err != nil {
		t.Fatal(err)
	}
	if acquired {
		_, _ = c.ExecContext(t.Context(), "SELECT pg_advisory_unlock($1)", int64(7337741002))
		t.Fatal("cmd/artex suite does not hold the shared database advisory lock")
	}
}

func TestShutdownContextPreservesNamedCause(t *testing.T) {
	signalCtx, signalCancel := context.WithCancel(context.Background())
	ctx, shutdown := shutdownContext(signalCtx)
	defer shutdown(nil)
	signalCancel()
	<-ctx.Done()
	if code, _, _, ok := agent.AbortReason(ctx); !ok || code != "shutdown" {
		t.Fatalf("code=%q ok=%v, want shutdown", code, ok)
	}
}

func TestShutdownServerExitProtocol(t *testing.T) {
	for _, code := range []int{0, selfupdate.ExitRestart, 1} {
		called := false
		got := shutdownServer(code, &http.Server{}, func(ctx context.Context) error {
			called = true
			if _, ok := ctx.Deadline(); !ok || ctx.Err() != nil {
				t.Error("drain lacks fresh bounded context")
			}
			return nil
		})
		if !called || got != code {
			t.Fatalf("exit %d: got %d, drained=%v", code, got, called)
		}
	}
	for _, err := range []error{context.DeadlineExceeded, errors.New("owned final checkpoint failure")} {
		if got := shutdownServer(selfupdate.ExitRestart, &http.Server{}, func(context.Context) error { return err }); got != 1 {
			t.Fatalf("failed drain exit=%d", got)
		}
	}
}

func TestShutdownServerHTTPTimeoutStillDrains(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-release }))
	defer ts.Close()
	defer close(release)
	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		resp, err := ts.Client().Get(ts.URL)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}()
	<-started
	called := false
	got := shutdownServer(0, ts.Config, func(ctx context.Context) error {
		called = true
		if ctx.Err() != nil {
			t.Error("HTTP deadline leaked into chat drain")
		}
		select {
		case <-clientDone:
		case <-time.After(time.Second):
			t.Error("HTTP timeout did not close connection")
		}
		return nil
	})
	if got != 1 || !called {
		t.Fatalf("HTTP timeout exit=%d drained=%v", got, called)
	}
}

func TestMainServeFailureRunsShutdown(t *testing.T) {
	if os.Getenv("SCOPEWEAVER_TEST_SERVE_FAILURE_CHILD") == "1" {
		// The child starts with an empty config and validates the test DSN before
		// making it the application's DSN, preserving dbtest's live-DB guard.
		if err := os.Setenv("ARTEX_PG_DSN", dbtest.DSN(t)); err != nil {
			t.Fatal(err)
		}
		flag.CommandLine = flag.NewFlagSet("artex", flag.ExitOnError)
		os.Args = []string{"artex", "-addr", os.Getenv("SCOPEWEAVER_TEST_OCCUPIED_ADDR"), "-data", "data", "-proxy", ""}
		os.Exit(run())
	}
	dsn := dbtest.DSN(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestMainServeFailureRunsShutdown$")
	cmd.Dir = t.TempDir()
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		upper := strings.ToUpper(key)
		if strings.HasPrefix(upper, "ARTEX_") || strings.HasPrefix(upper, "SCOPEWEAVER_") || strings.HasPrefix(upper, "PG") || strings.Contains(upper, "API_KEY") {
			continue
		}
		cmd.Env = append(cmd.Env, entry)
	}
	cmd.Env = append(cmd.Env, "ARTEX_TEST_DSN="+dsn, "ARTEX_CONFIG="+filepath.Join(cmd.Dir, "config.json"), "ARTEX_SELFUPDATE_SMOKE=1", "SCOPEWEAVER_TEST_SERVE_FAILURE_CHILD=1", "SCOPEWEAVER_TEST_OCCUPIED_ADDR="+listener.Addr().String())
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 {
		t.Fatalf("serve failure did not exit 1: %v", err)
	}
	if !strings.Contains(string(out), "serve:") || !strings.Contains(string(out), "shutting down...") {
		t.Fatalf("serve failure bypassed cleanup: %s", out)
	}
	if strings.Contains(string(out), "database is closed") {
		t.Fatalf("serve cleanup raced database closure: %s", out)
	}
	t.Log("occupied listen port exited 1 through bounded shutdown")
}
