package agent

import (
	"context"
	"encoding/json"

	"github.com/Autumn-27/norma/agentcore"
	"github.com/Autumn-27/norma/harness"
	"github.com/Autumn-27/norma/tool"
)

type callbackOwnerKey struct{}

// WithCallbackOwner lets a host own SDK callbacks independently of the prompt
// that starts them. A successful reservation must return its release function;
// false refuses callback execution. Ordinary prompt cancellation is unchanged.
func WithCallbackOwner(ctx context.Context, reserve func() (func(), bool)) context.Context {
	return context.WithValue(ctx, callbackOwnerKey{}, reserve)
}

// The SDK may return synthetic results before tool goroutines finish. Server
// contexts reserve those host callbacks until they return, without delaying the
// SDK's normal pause, cancellation, or duration-limit behavior.
func newOwnedSession(ctx context.Context, opts agentcore.Options) *agentcore.Session {
	reserve, _ := ctx.Value(callbackOwnerKey{}).(func() (func(), bool))
	if reserve == nil {
		return agentcore.NewSession(opts)
	}
	tools := opts.Tools
	if tools == nil {
		tools = tool.DefaultTools()
	}
	opts.Tools = make([]tool.CoreTool, len(tools))
	for i, t := range tools {
		// Deferred tools live in this same registry; ExecuteExtraTool resolves
		// these wrappers too, rather than bypassing their admission barrier.
		opts.Tools[i] = &ownedSessionTool{CoreTool: t, reserve: reserve}
	}
	if opts.Hooks != nil {
		hooks := &ownedSessionHooks{HookRunner: opts.Hooks, reserve: reserve}
		if firer, ok := opts.Hooks.(sessionHookFirer); ok {
			opts.Hooks = &ownedSessionFiringHooks{ownedSessionHooks: hooks, sessionHookFirer: firer}
		} else {
			opts.Hooks = hooks
		}
	}
	return agentcore.NewSession(opts)
}

type ownedSessionTool struct {
	tool.CoreTool
	reserve func() (func(), bool)
}

func (t *ownedSessionTool) Call(ctx context.Context, input json.RawMessage, tc *tool.ToolContext) (tool.Result, error) {
	release, ok := t.reserve()
	if !ok {
		return tool.Result{}, context.Canceled
	}
	defer release()
	return t.CoreTool.Call(ctx, input, tc)
}

func (t *ownedSessionTool) AcceptsRawInput() bool {
	if raw, ok := t.CoreTool.(interface{ AcceptsRawInput() bool }); ok {
		return raw.AcceptsRawInput()
	}
	return false
}

type ownedSessionHooks struct {
	harness.HookRunner
	reserve func() (func(), bool)
}

func (h *ownedSessionHooks) PreToolUse(ctx context.Context, name string, input []byte) (bool, string, []byte) {
	release, ok := h.reserve()
	if !ok {
		return true, context.Canceled.Error(), nil
	}
	defer release()
	return h.HookRunner.PreToolUse(ctx, name, input)
}

func (h *ownedSessionHooks) PostToolUse(ctx context.Context, name string, input, result []byte, isErr bool) {
	release, ok := h.reserve()
	if !ok {
		return
	}
	defer release()
	h.HookRunner.PostToolUse(ctx, name, input, result, isErr)
}

// Preserve agentcore's optional session-start and prompt-submission hooks.
type sessionHookFirer interface {
	FireSessionStart(context.Context)
	FireUserPromptSubmit(context.Context, string) string
}

type ownedSessionFiringHooks struct {
	*ownedSessionHooks
	sessionHookFirer
}
