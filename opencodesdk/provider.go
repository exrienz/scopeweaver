// Package opencodesdk speaks the versioned, capture-only OpenCode bridge protocol.
// The bridge never executes host tools: Norma remains their sole executor.
package opencodesdk

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"github.com/Autumn-27/artex/locale"
	"io"
	"iter"
	"mime"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/Autumn-27/artex/llmnet"
	"github.com/Autumn-27/norma/llm"
	"github.com/Autumn-27/norma/transcript"
	"github.com/google/uuid"
)

const Format = "opencode-sdk"
const DefaultBaseURL = "http://127.0.0.1:4097"
const maxBody = 32 << 20
const maxLine = 8 << 20

// NormalizeBaseURL accepts only explicit loopback bridges and preserves path/port identity.
func NormalizeBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = DefaultBaseURL
	}
	u, err := url.Parse(raw)
	if err != nil || u.Opaque != "" || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return "", locale.NewError("bridge URL must be a loopback HTTP(S) URL without credentials, query or fragment")
	}
	host := strings.ToLower(u.Hostname())
	if host != "localhost" {
		ip, e := netip.ParseAddr(host)
		if e != nil || !ip.IsLoopback() {
			return "", locale.NewError("bridge URL must use localhost or a literal loopback address")
		}
		host = ip.String()
	}
	port := u.Port()
	if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
		port = ""
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	u.Host = host
	if port != "" {
		u.Host += ":" + port
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = strings.TrimRight(u.RawPath, "/")
	return u.String(), nil
}

func SameEndpoint(a, b string) bool {
	aa, e := NormalizeBaseURL(a)
	bb, f := NormalizeBaseURL(b)
	return e == nil && f == nil && aa == bb
}

// BridgeError retains the authenticated bridge's classification instead of guessing from diagnostic text.
type BridgeError struct {
	Code    string
	Status  int
	Message string
}

func (e *BridgeError) Error() string {
	if diagnostic, ok := bridgeDiagnostics[e.Code]; ok {
		// Norma's overflow classifier requires the context_length machine marker.
		return e.Code + ": " + diagnostic.Error()
	}
	return e.Message
}

// Only protocol-owned codes select translations; bridge-supplied prose is evidence, never a template.
func (e *BridgeError) MessageForLanguage(lang locale.Lang) string {
	if diagnostic, ok := bridgeDiagnostics[e.Code]; ok {
		return locale.ErrorMessage(lang, diagnostic)
	}
	return e.Message
}

// Keep this bounded catalog aligned with adapters/opencode-sdk/src/protocol.mjs.
var bridgeDiagnostics = map[string]error{
	"unauthorized":            locale.NewError("Bridge authentication required."),
	"forbidden":               locale.NewError("Only non-browser loopback requests are accepted."),
	"invalid_request":         locale.NewError("Invalid completion request."),
	"unsupported_input":       locale.NewError("This input cannot be represented safely by the SDK."),
	"unsupported_runtime":     locale.NewError("Unsupported SDK/runtime pair or platform. Use a tested pinned runtime."),
	"unknown_model":           locale.NewError("The selected model is unavailable in this bridge."),
	"duplicate_request":       locale.NewError("This request ID is already active."),
	"canceled":                locale.NewError("The request was canceled."),
	"timeout":                 locale.NewError("The SDK request timed out."),
	"sdk_failure":             locale.NewError("The SDK request failed. Check the selected model and provider configuration."),
	"protocol_error":          locale.NewError("The SDK response could not be represented safely."),
	"configuration_error":     locale.NewError("Invalid bridge provider configuration."),
	"second_model_request":    locale.NewError("An unexpected additional SDK model request was blocked."),
	"quota_exhausted":         locale.NewError("The upstream provider reported exhausted quota or billing credit."),
	"context_length_exceeded": locale.NewError("The upstream model reported that the context length was exceeded."),
	"free_tier_restricted":    locale.NewError("The hosted free tier is restricted for this SDK runtime."),
	"not_found":               locale.NewError("Route not found."),
	"body_too_large":          locale.NewError("Request body exceeds the bridge limit."),
}

func (e *BridgeError) QuotaExhausted() bool {
	return e.Code == "quota_exhausted" && (e.Status == http.StatusPaymentRequired || e.Status == 0)
}

type Options struct {
	Client *http.Client
	// LocalOnly is evaluated before every operation, so cached profiles follow policy changes.
	LocalOnly       func() bool
	ThinkingType    string
	ReasoningEffort string
}

type Provider struct {
	base, token, model string
	client             *http.Client
	opts               Options
}

func New(base, token, model string, opts Options) (*Provider, error) {
	normalized, err := NormalizeBaseURL(base)
	if err != nil {
		return nil, err
	}
	token = strings.TrimSpace(token)
	if token == "" || strings.IndexFunc(token, unicode.IsControl) >= 0 {
		return nil, locale.NewError("a separate OpenCode bridge token is required")
	}
	client := opts.Client
	if client == nil {
		client, err = llmnet.NewClient("", true, nil)
		if err != nil {
			return nil, err
		}
	}
	clone := *client
	clone.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	p := &Provider{base: normalized, token: token, model: strings.TrimSpace(model), client: &clone, opts: opts}
	if err := p.policy(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Provider) policy() error {
	if p.opts.LocalOnly != nil && p.opts.LocalOnly() {
		return locale.NewError("local_llm_only: OpenCode SDK downstream inference cannot be restricted by the loopback bridge")
	}
	return nil
}
func (p *Provider) clean(s string) string {
	s = strings.ReplaceAll(s, p.token, "[redacted]")
	quoted, _ := json.Marshal(p.token)
	if len(quoted) > 2 {
		s = strings.ReplaceAll(s, string(quoted[1:len(quoted)-1]), "[redacted]")
	}
	return s
}
func (p *Provider) safeError(err error) error {
	if err != nil && p.clean(err.Error()) != err.Error() {
		return locale.Errorf("OpenCode SDK request failed: %w", &BridgeError{Code: "transport_error", Message: p.clean(err.Error())})
	}
	return err
}
func (p *Provider) request(ctx context.Context, method, path string, body any) (*http.Response, error) {
	if method != http.MethodDelete {
		if err := p.policy(); err != nil {
			return nil, err
		}
	}
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		if len(b) > maxBody {
			return nil, locale.NewError("OpenCode SDK request exceeds 32 MiB")
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, p.base+path, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Accept", "application/json")
	if path == "/v1/stream" {
		req.Header.Set("Accept", "application/x-ndjson")
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := p.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, locale.Errorf("OpenCode SDK request failed: %w", &BridgeError{Code: "transport_error", Message: p.clean(err.Error())})
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		var wire struct {
			Error struct{ Code, Message string } `json:"error"`
		}
		_ = json.Unmarshal(b, &wire)
		detail := wire.Error.Code + ": " + wire.Error.Message
		if wire.Error.Code == "" {
			detail = http.StatusText(resp.StatusCode)
		}
		return nil, locale.Errorf("OpenCode SDK HTTP %d: %w", resp.StatusCode, &BridgeError{Code: wire.Error.Code, Status: resp.StatusCode, Message: p.clean(detail)})
	}
	return resp, nil
}

func decode(r io.Reader, into any) error {
	b, err := io.ReadAll(io.LimitReader(r, maxBody+1))
	if err != nil {
		return err
	}
	if len(b) > maxBody {
		return locale.NewError("OpenCode SDK response exceeds 32 MiB")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	if err = d.Decode(into); err != nil {
		return locale.Errorf("invalid OpenCode SDK JSON: %w", err)
	}
	var extra any
	if err = d.Decode(&extra); err != io.EOF {
		return locale.NewError("invalid OpenCode SDK trailing JSON")
	}
	return nil
}

type Capabilities struct {
	Protocol      string `json:"protocol"`
	Version       int    `json:"version"`
	SDKVersion    string `json:"sdk_version"`
	Runtime       string `json:"runtime"`
	ToolExecution string `json:"tool_execution"`
	Streaming     bool   `json:"streaming"`
}

func (p *Provider) Capabilities(ctx context.Context) (Capabilities, error) {
	var c Capabilities
	if err := p.policy(); err != nil {
		return c, p.safeError(err)
	}
	resp, err := p.request(ctx, "GET", "/v1/capabilities", nil)
	if err != nil {
		return c, p.safeError(err)
	}
	defer resp.Body.Close()
	if err = decode(resp.Body, &c); err != nil {
		return c, p.safeError(err)
	}
	supported := (c.Runtime == "native" && c.SDKVersion == "2.1.4") || (c.Runtime == "legacy" && c.SDKVersion == "1.19.4")
	if c.Protocol != "scopeweaver-opencode-sdk" || c.Version != 1 || c.ToolExecution != "capture-only" || !c.Streaming || !supported {
		return c, locale.NewError("unsupported OpenCode SDK bridge protocol, runtime version or capture-only capabilities")
	}
	return c, nil
}

type Model struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ValidModel requires an explicit provider plus a nonempty model path; nested vendor/model IDs are preserved.
func ValidModel(model string) bool {
	a, b, ok := strings.Cut(model, "/")
	if !ok || a == "" || b == "" || strings.IndexFunc(model, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 || strings.ContainsAny(model, "?#\\") {
		return false
	}
	for _, part := range strings.Split(b, "/") {
		if part == "" {
			return false
		}
	}
	return true
}

func (p *Provider) Models(ctx context.Context) ([]Model, error) {
	if _, err := p.Capabilities(ctx); err != nil {
		return nil, err
	}
	resp, err := p.request(ctx, "GET", "/v1/models", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var body struct {
		Models []Model `json:"models"`
	}
	if err = decode(resp.Body, &body); err != nil {
		return nil, p.safeError(err)
	}
	for _, m := range body.Models {
		if !ValidModel(m.ID) {
			return nil, locale.NewError("invalid OpenCode SDK provider/model ID")
		}
	}
	if body.Models == nil {
		body.Models = []Model{}
	}
	return body.Models, nil
}

type completionInput struct {
	RequestID       string           `json:"request_id"`
	SessionID       string           `json:"session_id,omitempty"`
	Model           string           `json:"model"`
	System          []string         `json:"system"`
	Messages        []llm.Message    `json:"messages"`
	Tools           []llm.ToolSchema `json:"tools"`
	MaxTokens       int              `json:"max_tokens"`
	Temperature     *float64         `json:"temperature,omitempty"`
	Stop            []string         `json:"stop,omitempty"`
	Thinking        string           `json:"thinking,omitempty"`
	ThinkingType    string           `json:"thinking_type,omitempty"`
	ReasoningEffort string           `json:"reasoning_effort,omitempty"`
}

func (p *Provider) input(ctx context.Context, req llm.CompletionRequest) (completionInput, error) {
	if !ValidModel(p.model) {
		return completionInput{}, locale.NewError("OpenCode SDK model must be providerID/modelID")
	}
	if req.MaxTokens < 0 {
		return completionInput{}, locale.NewError("OpenCode SDK max_tokens must be nonnegative")
	}
	if _, err := p.Capabilities(ctx); err != nil {
		return completionInput{}, err
	}
	tools := req.Tools
	if tools == nil {
		tools = []llm.ToolSchema{}
	}
	messages := req.Messages
	if messages == nil {
		messages = []llm.Message{}
	}
	system := req.System
	if system == nil {
		system = []string{}
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 32768
	}
	return completionInput{RequestID: uuid.NewString(), SessionID: transcript.SessionIDFrom(ctx), Model: p.model, System: system, Messages: messages, Tools: tools, MaxTokens: maxTokens, Temperature: req.Temperature, Stop: req.Stop, Thinking: req.Thinking, ThinkingType: p.opts.ThinkingType, ReasoningEffort: p.opts.ReasoningEffort}, nil
}
func (p *Provider) cleanup(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp, err := p.request(ctx, "DELETE", "/v1/requests/"+id, nil)
	if err == nil {
		resp.Body.Close()
	}
}
func validUsage(u llm.Usage) bool {
	maxInt := int(^uint(0) >> 1)
	return u.InputTokens >= 0 && u.OutputTokens >= 0 && u.CacheReadTokens >= 0 && u.CacheWriteTokens >= 0 && u.CacheReadTokens <= maxInt-u.CacheWriteTokens && u.InputTokens <= maxInt-u.CacheReadTokens-u.CacheWriteTokens
}

// Native usage excludes cache from input; Norma's canonical total includes both cached subsets.
func canonicalUsage(u llm.Usage) llm.Usage {
	u.InputTokens += u.CacheReadTokens + u.CacheWriteTokens
	return u
}

func validStop(s string) bool { return s == "end_turn" || s == "tool_use" || s == "max_tokens" }
func toolNames(req llm.CompletionRequest) map[string]bool {
	m := map[string]bool{}
	for _, t := range req.Tools {
		m[t.Name] = true
	}
	return m
}
func validateMessage(m llm.Message, stop string, usage llm.Usage, names map[string]bool) error {
	if m.Role != llm.RoleAssistant || !validStop(stop) || !validUsage(usage) {
		return locale.NewError("invalid OpenCode SDK message, stop reason or usage")
	}
	ids := map[string]bool{}
	calls := 0
	for _, b := range m.Content {
		switch b.Type {
		case llm.BlockText, llm.BlockThinking:
		case llm.BlockToolUse:
			if b.ID == "" || ids[b.ID] || !names[b.Name] {
				return locale.NewError("invalid or unknown OpenCode SDK host tool call")
			}
			ids[b.ID] = true
			calls++
			if len(b.Input) == 0 || !json.Valid(b.Input) {
				return locale.NewError("invalid OpenCode SDK tool arguments")
			}
		default:
			return locale.NewError("unsupported OpenCode SDK content block")
		}
	}
	if (calls > 0) != (stop == "tool_use") {
		return locale.NewError("OpenCode SDK tool stop reason does not match tool calls")
	}
	return nil
}
func (p *Provider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
	in, err := p.input(ctx, req)
	if err != nil {
		return llm.Message{}, "", llm.Usage{}, p.safeError(err)
	}
	defer p.cleanup(in.RequestID)
	resp, err := p.request(ctx, "POST", "/v1/complete", in)
	if err != nil {
		return llm.Message{}, "", llm.Usage{}, p.safeError(err)
	}
	defer resp.Body.Close()
	var body struct {
		Message llm.Message `json:"message"`
		Stop    string      `json:"stop_reason"`
		Usage   *llm.Usage  `json:"usage"`
	}
	if err = decode(resp.Body, &body); err == nil {
		if body.Usage == nil {
			err = locale.NewError("OpenCode SDK response has no usage")
		} else {
			err = validateMessage(body.Message, body.Stop, *body.Usage, toolNames(req))
		}
	}
	if err != nil {
		return llm.Message{}, "", llm.Usage{}, p.safeError(err)
	}
	return body.Message, body.Stop, canonicalUsage(*body.Usage), nil
}

type wireEvent struct {
	Type       llm.StreamEventType `json:"type"`
	Text       string              `json:"text"`
	ToolID     string              `json:"tool_id"`
	ToolName   string              `json:"tool_name"`
	StopReason string              `json:"stop_reason"`
	Usage      *llm.Usage          `json:"usage"`
	Code       string              `json:"code"`
	Message    string              `json:"message"`
}

func (p *Provider) Stream(ctx context.Context, req llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
	return func(yield func(llm.StreamEvent, error) bool) {
		fail := func(e error) { yield(llm.StreamEvent{}, p.safeError(e)) }
		in, err := p.input(ctx, req)
		if err != nil {
			fail(err)
			return
		}
		defer p.cleanup(in.RequestID)
		resp, err := p.request(ctx, "POST", "/v1/stream", in)
		if err != nil {
			fail(err)
			return
		}
		defer resp.Body.Close()
		media, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
		if media != "application/x-ndjson" {
			fail(locale.NewError("OpenCode SDK stream must use application/x-ndjson"))
			return
		}
		scanner := bufio.NewScanner(io.LimitReader(resp.Body, maxBody+1))
		scanner.Buffer(make([]byte, 64<<10), maxLine)
		started, stopped, terminal, hasUsage := false, false, false, false
		size := 0
		tools := map[string]bool{}
		names := toolNames(req)
		calls := 0
		activeTool := ""
		inputs := map[string]string{}
		for scanner.Scan() {
			line := scanner.Bytes()
			size += len(line) + 1
			if size > maxBody {
				fail(locale.NewError("OpenCode SDK stream exceeds 32 MiB"))
				return
			}
			if len(bytes.TrimSpace(line)) == 0 {
				continue
			}
			var ev wireEvent
			if err = json.Unmarshal(line, &ev); err != nil {
				fail(locale.Errorf("invalid OpenCode SDK stream JSON: %w", err))
				return
			}
			if stopped {
				fail(locale.NewError("OpenCode SDK event after message_stop"))
				return
			}
			if ev.Type == "error" {
				fail(locale.Errorf("OpenCode SDK %s: %w", p.clean(ev.Code), &BridgeError{Code: ev.Code, Message: p.clean(ev.Message)}))
				return
			}
			if terminal && ev.Type != llm.SEMessageStop {
				fail(locale.NewError("OpenCode SDK event after terminal delta"))
				return
			}
			if !started && ev.Type != llm.SEMessageStart {
				fail(locale.NewError("OpenCode SDK stream missing message_start"))
				return
			}
			if ev.Usage != nil && !validUsage(*ev.Usage) {
				fail(locale.NewError("invalid OpenCode SDK usage"))
				return
			}
			switch ev.Type {
			case llm.SEMessageStart:
				if started {
					fail(locale.NewError("duplicate OpenCode SDK message_start"))
					return
				}
				started = true
			case llm.SETextDelta, llm.SEThinkingDelta, llm.SEThinkingSignature:
			case llm.SEToolUseStart:
				if ev.ToolID == "" || tools[ev.ToolID] || !names[ev.ToolName] {
					fail(locale.NewError("invalid or unknown OpenCode SDK host tool call"))
					return
				}
				tools[ev.ToolID] = true
				activeTool = ev.ToolID
				calls++
			case llm.SEToolInputJSON:
				if activeTool == "" {
					fail(locale.NewError("OpenCode SDK tool input without tool start"))
					return
				}
				inputs[activeTool] += ev.Text
			case llm.SEMessageDelta:
				if terminal || !validStop(ev.StopReason) || (calls > 0) != (ev.StopReason == "tool_use") {
					fail(locale.NewError("invalid OpenCode SDK terminal delta"))
					return
				}
				for _, input := range inputs {
					if !json.Valid([]byte(input)) {
						fail(locale.NewError("invalid OpenCode SDK tool input JSON"))
						return
					}
				}
				for id := range tools {
					if _, ok := inputs[id]; !ok {
						fail(locale.NewError("missing OpenCode SDK tool input"))
						return
					}
				}
				terminal = true
				hasUsage = ev.Usage != nil
			case llm.SEMessageStop:
				if !terminal || !hasUsage {
					fail(locale.NewError("OpenCode SDK stream missing terminal usage"))
					return
				}
				stopped = true
			default:
				fail(locale.NewError("unknown OpenCode SDK event type"))
				return
			}
			out := llm.StreamEvent{Type: ev.Type, Text: ev.Text, ToolID: ev.ToolID, ToolName: ev.ToolName, StopReason: ev.StopReason}
			if ev.Usage != nil {
				out.Usage = canonicalUsage(*ev.Usage)
			}
			if !yield(out, nil) {
				return
			}
		}
		if err = scanner.Err(); err != nil {
			if ctx.Err() != nil {
				err = ctx.Err()
			}
			fail(locale.Errorf("OpenCode SDK stream failed: %w", err))
			return
		}
		if !stopped {
			fail(locale.NewError("OpenCode SDK stream ended without message_stop"))
		}
	}
}
