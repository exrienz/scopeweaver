// Loaded only by the owned, isolated CLI 1.19.4 process. No host tools execute here.
import { readFileSync } from "node:fs";

export default async (_input, options) => {
  const state = JSON.parse(readFileSync(options.state, "utf8"));
  const attempts = new Map();
  let network = 0, quota = false, contextOverflow = false, freeTier = false, redirected = false;
  const blocked = () => redirected ? "SCOPEWEAVER_LEGACY_REDIRECT" : freeTier ? "SCOPEWEAVER_LEGACY_FREE_TIER" : contextOverflow ? "SCOPEWEAVER_LEGACY_CONTEXT_LENGTH" : quota ? "SCOPEWEAVER_LEGACY_QUOTA" : "SCOPEWEAVER_LEGACY_SECOND_REQUEST";
  return {
    config: async config => {
      config.provider ??= {};
      config.provider[state.provider] ??= {};
      config.provider[state.provider].options ??= {};
      // Public provider options support custom fetch. It gates actual network
      // attempts too, including AI SDK retries, and refuses HTTP redirects.
      config.provider[state.provider].options.fetch = async (input, init) => {
        if (++network !== 1) throw Error(blocked());
        const response = await fetch(input, { ...init, redirect: "error" }).catch(error => {
          // Preserve the first redirect failure if the CLI retries either gate.
          redirected = error?.code === "UnexpectedRedirect";
          throw error;
        });
        if ([400, 413, 402, 429, 403].includes(response.status)) {
          const reader = response.clone().body?.getReader();
          if (reader) {
            try {
              const decoder = new TextDecoder(); let text = "";
              while (text.length < 100000) { const chunk = await reader.read(); if (chunk.done) break; text += decoder.decode(chunk.value, { stream: true }); }
              if (text.length < 100000) {
                const body = JSON.parse(text);
                quota = [402, 429].includes(response.status) && ["insufficient_quota", "billing_hard_limit_reached", "billing_not_active", "credit_balance_exhausted", "quota_exceeded"].includes(body?.error?.code);
                contextOverflow = [400, 413].includes(response.status) && [body?.error?.code, body?.error?.type].some(code => ["context_length_exceeded", "model_context_window_exceeded", "context_window_exceeded", "prompt_too_long"].includes(code));
                freeTier = response.status === 403 && [body?.error?.code, body?.error?.type, body?.error?.name].includes("FreeTierError");
              }
            } catch {} finally { void reader.cancel().catch(() => {}); }
          }
        }
        // The CLI otherwise normalizes context errors and can discard the
        // structured HTTP evidence. Preserve only our proven typed marker.
        if (contextOverflow || freeTier) throw Error(blocked());
        return response;
      };
    },
    "tool.execute.before": async input => {
      if (!state.allowed.includes(input.tool)) throw Error("SCOPEWEAVER_LEGACY_NATIVE_TOOL");
    },
    "chat.params": async (input, output) => {
      const count = (attempts.get(input.sessionID) ?? 0) + 1;
      attempts.set(input.sessionID, count);
      if (count !== 1) throw Error(blocked());
      output.maxOutputTokens = state.request.max_tokens;
      if (state.request.temperature !== undefined) output.temperature = state.request.temperature;
      output.options = {
        ...output.options,
        ...(state.request.stop !== undefined ? { stop: state.request.stop } : {}),
        ...(state.request.thinking || state.request.thinking_type
          ? { thinking: { type: state.request.thinking || state.request.thinking_type } } : {}),
        ...(state.request.reasoning_effort ? { reasoningEffort: state.request.reasoning_effort } : {}),
      };
    },
    "chat.headers": async (_input, output) => {
      // OpenCode owns its internal session identity; upstream affinity remains host-owned.
      if (["opencode", "opencode-go"].includes(state.provider))
        output.headers["x-opencode-session"] = state.request.session_id ?? state.request.request_id;
    },
    "experimental.chat.system.transform": async (_input, output) => {
      output.system.splice(0, output.system.length, ...state.request.system);
    },
    "experimental.chat.messages.transform": async (_input, output) => {
      const sessionID = output.messages[0]?.info.sessionID;
      if (!sessionID) throw Error("SCOPEWEAVER_LEGACY_HISTORY");
      const history = state.history.map(message => ({
        info: { ...message.info, sessionID },
        parts: message.parts.map(part => ({ ...part, sessionID })),
      }));
      output.messages.splice(0, output.messages.length, ...history);
    },
  };
};
