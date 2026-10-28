package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Autumn-27/artex/agent"
	"github.com/Autumn-27/artex/chatstate"
	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/artex/internal/dbtest"
	"github.com/Autumn-27/artex/locale"
	"github.com/Autumn-27/norma/llm"
	actool "github.com/Autumn-27/norma/tool"
	"github.com/Autumn-27/norma/transcript"
	"github.com/jackc/pgx/v5"
)

type chatBrowserFixture struct {
	mu          sync.Mutex
	calls       int
	requests    []string
	hold        chan struct{}
	holdArmed   bool
	toolHold    chan struct{}
	toolStarted bool
	markerCalls int
	providers   []map[string]any
	snapshots   []map[string]any
	admissions  []map[string]any
}

func (f *chatBrowserFixture) release() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.hold != nil {
		close(f.hold)
		f.hold = nil
	}
	if f.toolHold != nil {
		close(f.toolHold)
		f.toolHold = nil
	}
}
func (f *chatBrowserFixture) model(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	provider := map[string]any{"ordinal": len(f.providers) + 1, "session": r.Header.Get("X-Chat-E2E-Session"), "arrival": time.Now().UTC().Format(time.RFC3339Nano)}
	f.providers = append(f.providers, provider)
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		provider["completion"] = time.Now().UTC().Format(time.RFC3339Nano)
		provider["context_error"] = fmt.Sprint(r.Context().Err())
	}()
	var in struct {
		Stream   bool             `json:"stream"`
		Messages []map[string]any `json:"messages"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil {
		http.Error(w, "bad fixture request", 400)
		return
	}
	raw, _ := json.Marshal(in.Messages)
	session := r.Header.Get("X-Chat-E2E-Session")
	main := strings.HasPrefix(session, "conv-") || strings.Contains(session, "-main")
	f.mu.Lock()
	f.calls++
	n := f.calls
	f.requests = append(f.requests, string(raw))
	provider["count"] = n
	provider["count_assignment"] = time.Now().UTC().Format(time.RFC3339Nano)
	provider["messages"] = json.RawMessage(raw)
	provider["stream"] = in.Stream
	held := f.hold
	if main && f.holdArmed {
		f.holdArmed = false
	} else {
		held = nil
	}
	f.mu.Unlock()
	if !main {
		held = nil
	}
	toolName := ""
	content := fmt.Sprintf("Fixture response %d", n)
	if strings.Contains(string(raw), "steer-fixture-trigger") {
		switch {
		case strings.Contains(string(raw), "[Chat steer"):
			content = "Correction accepted"
		case strings.Contains(string(raw), "fixture-wait-complete"):
			toolName = "FixtureMarker"
		default:
			toolName = "FixtureWait"
		}
	}
	if in.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		send := func(v any) {
			b, _ := json.Marshal(v)
			fmt.Fprintf(w, "data: %s\n\n", b)
			if flush, ok := w.(http.Flusher); ok {
				flush.Flush()
			}
		}
		delta := map[string]any{"role": "assistant", "content": content}
		if toolName != "" {
			delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("fixture-%d", n), "type": "function", "function": map[string]any{"name": toolName, "arguments": "{}"}}}}
		}
		send(map[string]any{"id": "fixture", "choices": []any{map[string]any{"index": 0, "delta": delta}}})
		if held != nil {
			select {
			case <-held:
			case <-r.Context().Done():
				return
			}
		}
		stop := "stop"
		if toolName != "" {
			stop = "tool_calls"
		}
		send(map[string]any{"id": "fixture", "choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": stop}}, "usage": map[string]int{"prompt_tokens": 80, "completion_tokens": 5, "total_tokens": 85}})
		fmt.Fprint(w, "data: [DONE]\n\n")
		return
	}
	if held != nil {
		select {
		case <-held:
		case <-r.Context().Done():
			return
		}
	}
	message := map[string]any{"role": "assistant", "content": content}
	stop := "stop"
	if toolName != "" {
		message["tool_calls"] = []any{map[string]any{"id": fmt.Sprintf("fixture-%d", n), "type": "function", "function": map[string]any{"name": toolName, "arguments": "{}"}}}
		stop = "tool_calls"
	}
	writeJSON(w, 200, map[string]any{"id": "fixture", "choices": []any{map[string]any{"message": message, "finish_reason": stop}}, "usage": map[string]int{"prompt_tokens": 80, "completion_tokens": 5, "total_tokens": 85}})
}

// Opt-in browser gate: real API + static production UI, isolated database and
// loopback-only model/tool fixtures. No production fixture routes are installed.
func TestChatControlsE2E(t *testing.T) {
	if os.Getenv("SCOPEWEAVER_CHAT_E2E") != "1" {
		t.Skip("opt in with SCOPEWEAVER_CHAT_E2E=1, a fresh scopeweaver_test_* DB and the web/out build")
	}
	dsn := dbtest.DSN(t)
	parsed, err := pgx.ParseConfig(dsn)
	if err != nil || !strings.HasPrefix(parsed.Database, "scopeweaver_test_") {
		t.Fatal("a disposable scopeweaver_test_* database is required")
	}
	for _, path := range []string{"../web/out/chat/index.html", "../e2e/node_modules/playwright/package.json"} {
		if _, err = os.Stat(path); err != nil {
			t.Fatalf("browser prerequisite %s: %v", path, err)
		}
	}
	for _, key := range []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "ARTEX_LLM_PROVIDER"} {
		t.Setenv(key, "")
	}
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	var profiles int
	if err = m.pg.QueryRow(`SELECT count(*) FROM llm_profiles`).Scan(&profiles); err != nil || profiles != 0 {
		t.Fatalf("fresh fixture must contain no profiles: %d %v", profiles, err)
	}
	if _, err = m.pg.Exec(`UPDATE mcp_servers SET enabled=false`); err != nil {
		t.Fatal(err)
	}
	fixture := &chatBrowserFixture{}
	// Test cleanup runs after the existing deferred app/model/listener teardown.
	// A missing final file (timeout, cancellation, write failure) is incomplete evidence.
	t.Cleanup(func() {
		dir, pathErr := filepath.Abs(filepath.Join("..", "dist", "chat-e2e"))
		if pathErr != nil {
			t.Errorf("diagnostic path: %v", pathErr)
			return
		}
		suite := "standard"
		if os.Getenv("CHAT_E2E_SUITE") == "draft" {
			suite = "draft"
		}
		fixture.mu.Lock()
		data, marshalErr := json.Marshal(map[string]any{"schema": 1, "suite": suite, "server_time": time.Now().UTC().Format(time.RFC3339Nano), "calls": fixture.calls, "provider_ordinal": len(fixture.providers), "providers": fixture.providers, "snapshots": fixture.snapshots, "admissions": fixture.admissions})
		fixture.mu.Unlock()
		if marshalErr != nil {
			t.Errorf("diagnostic marshal: %v", marshalErr)
			return
		}
		if dirErr := os.MkdirAll(dir, 0700); dirErr != nil {
			t.Errorf("diagnostic directory: %v", dirErr)
			return
		}
		file := filepath.Join(dir, "fixture-ledger-"+suite+".json")
		if writeErr := os.WriteFile(file, data, 0600); writeErr != nil {
			t.Errorf("diagnostic write %s: %v", file, writeErr)
			return
		}
		t.Logf("diagnostic final ledger: %s (%d bytes)", file, len(data))
	})
	model := httptest.NewServer(http.HandlerFunc(fixture.model))
	defer model.Close()
	defer fixture.release()
	profile := &db.LLMProfile{Name: "browser-loopback-fixture", Format: "openai", BaseURL: model.URL + "/v1", Model: "fixture-large", APIKey: "fixture", Streaming: true, ContextWindowK: 64, MaxTokens: 2048, SessionHeaderKey: "X-Chat-E2E-Session"}
	profileID, err := m.pg.SaveProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	profile.ID = profileID
	if err = m.pg.SetActiveProfile(profileID); err != nil {
		t.Fatal(err)
	}
	oldAugment, oldResolve, oldPrompt := agent.ToolAugment, agent.ToolResolve, agent.PromptOverride
	defer func() { agent.ToolAugment, agent.ToolResolve, agent.PromptOverride = oldAugment, oldResolve, oldPrompt }()
	var serverMu sync.Mutex
	var app *Server
	var stop context.CancelFunc
	start := func() {
		ctx, cancel := context.WithCancel(context.Background())
		next := New(ctx, m, m.dir, m.dir, m.dir)
		augment := agent.ToolAugment
		agent.ToolAugment = func(ctx context.Context, key string) ([]actool.CoreTool, agent.DeferredInfo, func()) {
			tools, def, cleanup := augment(ctx, key)
			if key == "auto" || key == "mainagent" {
				for _, name := range []string{"FixtureWait", "FixtureMarker"} {
					name := name
					tools = append(tools, actool.Build(actool.Spec{Name: name, Description: "Loopback browser test fixture", Schema: map[string]any{"type": "object"}, ReadOnly: func(json.RawMessage) bool { return true }, Concurrent: func(json.RawMessage) bool { return false }, Run: func(ctx context.Context, _ json.RawMessage, _ *actool.ToolContext) (actool.Result, error) {
						if name == "FixtureMarker" {
							fixture.mu.Lock()
							fixture.markerCalls++
							fixture.mu.Unlock()
							return actool.Text("marker executed"), nil
						}
						fixture.mu.Lock()
						fixture.toolStarted = true
						if fixture.toolHold == nil {
							fixture.toolHold = make(chan struct{})
						}
						hold := fixture.toolHold
						fixture.mu.Unlock()
						select {
						case <-hold:
							return actool.Text("fixture-wait-complete"), nil
						case <-ctx.Done():
							return actool.Result{}, ctx.Err()
						}
					}}))
				}
			}
			return tools, def, cleanup
		}
		serverMu.Lock()
		app, stop = next, cancel
		serverMu.Unlock()
	}
	start()
	defer func() {
		serverMu.Lock()
		cancel := stop
		s := app
		serverMu.Unlock()
		cancel()
		waitRetestIdle(t, s)
		s.archiveWG.Wait()
		if s.side != nil {
			<-s.side.done
		}
	}()
	const fixtureToken = "loopback-chat-browser-fixture"
	getApp := func() *Server { serverMu.Lock(); defer serverMu.Unlock(); return app }
	mux := http.NewServeMux()
	mux.HandleFunc("/__fixture/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Chat-Fixture") != fixtureToken {
			http.Error(w, "fixture auth", 401)
			return
		}
		s := getApp()
		action := strings.TrimPrefix(r.URL.Path, "/__fixture/")
		switch action {
		case "hold":
			fixture.mu.Lock()
			fixture.hold = make(chan struct{})
			fixture.holdArmed = true
			fixture.mu.Unlock()
		case "release":
			fixture.release()
		case "reset-tools":
			fixture.mu.Lock()
			fixture.toolStarted = false
			fixture.markerCalls = 0
			fixture.mu.Unlock()
		case "stats":
			var in struct {
				Diagnostic map[string]any `json:"diagnostic"`
			}
			_ = json.NewDecoder(r.Body).Decode(&in)
			fixture.mu.Lock()
			defer fixture.mu.Unlock()
			snapshot := map[string]any{"case": in.Diagnostic["case"], "operation": in.Diagnostic["operation"], "boundary": in.Diagnostic["boundary"], "read": in.Diagnostic["read"], "marker": in.Diagnostic["marker"], "calls": fixture.calls, "provider_ordinal": len(fixture.providers), "server_time": time.Now().UTC().Format(time.RFC3339Nano)}
			fixture.snapshots = append(fixture.snapshots, snapshot)
			writeJSON(w, 200, map[string]any{"calls": fixture.calls, "requests": append([]string{}, fixture.requests...), "tool_started": fixture.toolStarted, "marker_calls": fixture.markerCalls, "diagnostic": snapshot})
			return
		case "budget":
			var in struct {
				Window int `json:"window"`
			}
			if json.NewDecoder(r.Body).Decode(&in) != nil || (in.Window != 32 && in.Window != 64) {
				http.Error(w, "window", 400)
				return
			}
			profile.ContextWindowK = in.Window
			profile.Model = fmt.Sprintf("fixture-%d", in.Window)
			if _, err := m.pg.SaveProfile(profile); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			s.invalidateProfileAgents()
			s.reapplyActiveProfile()
		case "recover":
			var in struct {
				Input string `json:"input_id"`
			}
			if json.NewDecoder(r.Body).Decode(&in) != nil {
				http.Error(w, "input", 400)
				return
			}
			var ref chatstate.SessionRef
			if err := m.pg.QueryRow(`SELECT COALESCE(s.conversation_id,0),COALESCE(s.task_id,0),COALESCE(s.exploration_id,0),s.main_seg FROM chat_control_inputs i JOIN chat_control_sessions s USING(session_key) WHERE i.id=$1`, in.Input).Scan(&ref.ConversationID, &ref.TaskID, &ref.ExplorationID, &ref.MainSeg); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			run, err := m.pg.ClaimChatHistoryInput(r.Context(), ref.GateKey(), in.Input)
			if err != nil || run == nil {
				http.Error(w, fmt.Sprint("claim ", err), 500)
				return
			}
			store := transcript.NewStore(filepath.Join(m.dir, "transcripts"))
			records, err := store.Load(store.MainPath(run.TranscriptID))
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			last := ""
			if len(records) > 0 {
				last = records[len(records)-1].UUID
			}
			if err = m.pg.RecordChatTranscriptBaseline(r.Context(), run.ID, last); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			writer := store.ResumeWriter(run.TranscriptID, last)
			writer.RecordMessage(llm.UserText("lost-prompt-sentinel"), llm.Usage{})
			if err = writer.Err(); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			serverMu.Lock()
			cancel := stop
			serverMu.Unlock()
			cancel()
			s.archiveWG.Wait()
			if s.side != nil {
				<-s.side.done
			}
			start()
		case "approval":
			var in struct {
				Ref  chatstate.SessionRef `json:"ref"`
				Run  string               `json:"run"`
				Turn string               `json:"turn"`
			}
			if json.NewDecoder(r.Body).Decode(&in) != nil {
				http.Error(w, "approval", 400)
				return
			}
			var id int64
			if err := m.pg.QueryRow(`INSERT INTO intercept_pending(conversation_id,task_id,tool_name,status) VALUES($1,$2,'Bash','pending') RETURNING id`, nullableFixtureID(in.Ref.ConversationID), func() any {
				if in.Ref.TaskID == 0 {
					return nil
				}
				return strconv.FormatInt(in.Ref.TaskID, 10)
			}()).Scan(&id); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			a := db.Activity{Worker: "mainagent", Kind: "intercept_request", Summary: fmt.Sprintf("Tool Bash requests approval (#%d)", id), RunID: in.Run, TurnID: in.Turn}
			active := true
			a.ActiveAttempt = &active
			if in.Ref.ConversationID > 0 {
				_, err = m.pg.AppendConvActivity(in.Ref.ConversationID, a)
			} else {
				seg := in.Ref.MainSeg
				a.MainSeg = &seg
				_, err = m.pg.Exploration(in.Ref.ExplorationID).AppendActivity(a)
			}
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			writeJSON(w, 200, map[string]string{"id": strconv.FormatInt(id, 10)})
			return
		case "clear-approval":
			var in struct {
				ID string `json:"id"`
			}
			if json.NewDecoder(r.Body).Decode(&in) != nil {
				http.Error(w, "approval", 400)
				return
			}
			if _, err := m.pg.Exec(`DELETE FROM intercept_pending WHERE id=$1`, in.ID); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		default:
			http.NotFound(w, r)
			return
		}
		writeJSON(w, 200, map[string]bool{"ok": true})
	})
	files := http.FileServer(http.Dir("../web/out"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			// Observe existing control traffic without extra application/API requests.
			relevant := strings.Contains(r.URL.Path, "/controls") || r.URL.Path == "/api/chat" || (r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/api/conversations/") && strings.HasSuffix(r.URL.Path, "/messages")) || (r.Method != "GET" && (r.URL.Path == "/api/tasks" || r.URL.Path == "/api/conversations" || strings.HasSuffix(r.URL.Path, "/control") || r.URL.Path == "/api/exploration/main-session/new"))
			if !relevant {
				getApp().Handler().ServeHTTP(w, r)
				return
			}
			var body bytes.Buffer
			if r.Body != nil {
				r.Body = &chatDiagnosticBody{Reader: io.TeeReader(r.Body, &body), Closer: r.Body}
			}
			observed := &chatDiagnosticResponse{ResponseWriter: w}
			fixture.mu.Lock()
			admission := map[string]any{"ordinal": len(fixture.admissions) + 1, "arrival": time.Now().UTC().Format(time.RFC3339Nano), "method": r.Method, "path": r.URL.Path, "query": r.URL.RawQuery}
			fixture.admissions = append(fixture.admissions, admission)
			fixture.mu.Unlock()
			defer func() {
				fixture.mu.Lock()
				defer fixture.mu.Unlock()
				admission["completion"] = time.Now().UTC().Format(time.RFC3339Nano)
				admission["status"] = observed.status
				admission["request_body_read"] = body.String()
				admission["response_body"] = observed.body.String()
				admission["context_error"] = fmt.Sprint(r.Context().Err())
			}()
			getApp().Handler().ServeHTTP(observed, r)
		} else {
			files.ServeHTTP(w, r)
		}
	})
	listener := httptest.NewServer(mux)
	defer listener.Close()
	// Preserve the original three-minute budget per locale as the matrix grows.
	browserTimeout := 3 * time.Minute * time.Duration(len(locale.Languages()))
	ctx, cancel := context.WithTimeout(context.Background(), browserTimeout)
	defer cancel()
	t.Logf("chat browser gate budget: %s (%d locales)", browserTimeout, len(locale.Languages()))
	command := exec.CommandContext(ctx, "node", "../e2e/chat-controls-runner.mjs")
	artifacts, _ := filepath.Abs(filepath.Join("..", "dist", "chat-e2e"))
	if err = os.MkdirAll(artifacts, 0700); err != nil {
		t.Fatal(err)
	}
	command.Env = append(os.Environ(), "SCOPEWEAVER_E2E_ARTIFACTS="+artifacts, "SCOPEWEAVER_E2E_URL="+listener.URL, "SCOPEWEAVER_E2E_FIXTURE="+fixtureToken)
	output, err := command.CombinedOutput()
	t.Log(string(output))
	if err != nil {
		if ctx.Err() != nil {
			t.Fatalf("chat browser gate exceeded %s: %v", browserTimeout, ctx.Err())
		}
		t.Fatalf("chat browser gate failed: %v", err)
	}
}
func nullableFixtureID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

// Only the finite JSON control routes above use this writer; SSE routes bypass it.
type chatDiagnosticBody struct {
	io.Reader
	io.Closer
}
type chatDiagnosticResponse struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *chatDiagnosticResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}
func (w *chatDiagnosticResponse) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.body.Write(b[:n])
	return n, err
}
