// Z.ai documentation verified 2026-10-03. This helper changes form fields only.
// https://docs.z.ai/guides/llm/glm-5.3
// https://zcode.z.ai/en/docs/configuration
// Ollama templates target the native /api/chat path (https://docs.ollama.com/api), verified 2026-10-15.
// Cloud economy templates follow provider docs checked 2026-10-16; sources are listed in
// .sdlc/work/gh-24-deepseek-token-consumption/refs/providers.md.
export const LLM_PROFILE_PRESET_IDS = [
  "zai-general",
  "zai-coding-reference",
  "deepseek",
  "claude",
  "openai",
  "qwen",
  "kimi",
  "openrouter",
  "opencode-go",
  "opencode-sdk",
  "ollama-host",
  "ollama-docker-host",
  "ollama-compose",
] as const;
export type LLMProfilePresetId = (typeof LLM_PROFILE_PRESET_IDS)[number];

export interface ZaiPresetFields {
  format: "openai";
  model: string;
  base_url: string;
  thinking_type: "enabled";
  reasoning_effort: "low";
  context_window_k: number;
  max_tokens: number;
  max_tokens_field: "";
  // Optional required session header; when a preset defines it, the form fills it.
  session_header_key?: string;
}

// Opt-in economy defaults: low effort and a 16384-token output cap. context_window_k 0 keeps
// the 200K compaction default. Empty thinking/effort values omit the request field.
// The opencode-go gateway template deliberately selects effort max at the user's request;
// every other cloud template keeps the measured economy low default.
export interface CloudPresetFields {
  format: "openai" | "anthropic" | "openai-responses";
  model: string;
  base_url: string;
  thinking_type: "" | "disabled";
  reasoning_effort: "" | "low" | "max";
  context_window_k: number;
  max_tokens: number;
  // Empty sends max_tokens in openai format; anthropic and openai-responses use fixed fields.
  max_tokens_field: "" | "max_completion_tokens";
  // Optional required session header; when a preset defines it, the form fills it.
  session_header_key?: string;
}

export interface OllamaPresetFields {
  format: "ollama";
  model: string;
  base_url: string;
  // Native templates clear any unsaved key so a cloud credential is never sent to a local endpoint.
  api_key: "";
  // Empty values omit the fields: Ollama decides thinking per model.
  thinking_type: "";
  reasoning_effort: "";
  // Binary K for Ollama: 32 sends options.num_ctx = 32768.
  context_window_k: number;
  max_tokens: number;
  max_tokens_field: "";
  keep_alive: string;
  llm_max_concurrent: number;
  // Ollama templates never set a session header; an entered one is kept.
  session_header_key?: string;
}

export interface OpenCodeSDKPresetFields {
  format: "opencode-sdk";
  model: "";
  base_url: string;
  api_key: "";
  proxy: "";
  session_header_key: "";
  streaming: true;
  thinking_type: "";
  reasoning_effort: "";
  context_window_k: number;
  max_tokens: number;
  max_tokens_field: "";
}
export type LLMProfilePresetFields = ZaiPresetFields | CloudPresetFields | OllamaPresetFields | OpenCodeSDKPresetFields;

const OLLAMA_TARGETS: Record<string, { base_url: string }> = {
  "ollama-host": { base_url: "http://127.0.0.1:11434" },
  "ollama-docker-host": { base_url: "http://host.docker.internal:11434" },
  "ollama-compose": { base_url: "http://ollama:11434" },
};

type CloudTarget = Pick<CloudPresetFields, "format" | "base_url" | "model" | "thinking_type" | "reasoning_effort" | "max_tokens_field" | "session_header_key">;

const CLOUD_TARGETS: Record<string, CloudTarget> = {
  deepseek: { format: "openai", base_url: "https://api.deepseek.com", model: "deepseek-flash", thinking_type: "", reasoning_effort: "low", max_tokens_field: "" },
  // Haiku allows disabled thinking at effort low; omitting it means adaptive thinking,
  // whose thinking blocks the pinned SDK drops when replaying tool history.
  claude: { format: "anthropic", base_url: "https://api.anthropic.com", model: "claude-haiku-5-5", thinking_type: "disabled", reasoning_effort: "low", max_tokens_field: "" },
  openai: { format: "openai-responses", base_url: "https://api.openai.com/v1", model: "gpt-6-luna", thinking_type: "", reasoning_effort: "low", max_tokens_field: "" },
  qwen: { format: "openai", base_url: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1", model: "qwen3.8-flash", thinking_type: "", reasoning_effort: "low", max_tokens_field: "max_completion_tokens" },
  kimi: { format: "openai", base_url: "https://api.moonshot.ai/v1", model: "kimi-k3", thinking_type: "", reasoning_effort: "low", max_tokens_field: "max_completion_tokens" },
  // OpenRouter lists reasoning_effort among this model's supported parameters.
  openrouter: { format: "openai", base_url: "https://openrouter.ai/api/v1", model: "deepseek/deepseek-v4.1-flash", thinking_type: "", reasoning_effort: "low", max_tokens_field: "" },
  // OpenCode Go (opencode subscription gateway, provider id "opencode-go" in opencode 1.18.5):
  // @ai-sdk/openai-compatible at https://opencode.ai/zen/go/v1, OPENCODE_API_KEY. GLM-5.3-class
  // models accept effort low/high/max; this template deliberately fills max at the user's request.
  // The gateway rejects requests without x-opencode-session (HTTP 400 MissingSessionID), so the
  // template fills the required session header.
  "opencode-go": { format: "openai", base_url: "https://opencode.ai/zen/go/v1", model: "glm-5.3-flash", thinking_type: "", reasoning_effort: "max", max_tokens_field: "", session_header_key: "x-opencode-session" },
};

export function isCloudEconomyPreset(id: string): boolean {
  return Object.hasOwn(CLOUD_TARGETS, id);
}

export function isOllamaPreset(id: string): boolean {
  return Object.hasOwn(OLLAMA_TARGETS, id);
}

export function getLLMProfilePreset(id: string): Readonly<LLMProfilePresetFields> {
  if (!(LLM_PROFILE_PRESET_IDS as readonly string[]).includes(id)) {
    throw new RangeError(`Unknown LLM profile preset: ${id}`);
  }
  if (id === "opencode-sdk") {
    return {
      format: "opencode-sdk", base_url: "http://127.0.0.1:4097", model: "", api_key: "", proxy: "", session_header_key: "", streaming: true,
      thinking_type: "", reasoning_effort: "", context_window_k: 0, max_tokens: 0, max_tokens_field: "",
    };
  }
  if (isOllamaPreset(id)) {
    return {
      format: "ollama",
      ...OLLAMA_TARGETS[id],
      // The form discovers an installed model from this server; never assume a tag is installed.
      model: "",
      api_key: "",
      thinking_type: "",
      reasoning_effort: "",
      context_window_k: 32,
      max_tokens: 4096,
      max_tokens_field: "",
      keep_alive: "30m",
      llm_max_concurrent: 1,
    };
  }
  if (isCloudEconomyPreset(id)) {
    return { ...CLOUD_TARGETS[id], context_window_k: 0, max_tokens: 16384 };
  }
  return {
    format: "openai",
    model: "glm-5.3",
    base_url: id === "zai-general" ? "https://api.z.ai/api/paas/v4" : "https://api.z.ai/api/coding/paas/v4",
    thinking_type: "enabled",
    // GH-24 measurement (GLM Coding Plan, 3 runs each): low effort used under half the tokens
    // of max with the same finding. The output cap and 200K window keep large tasks bounded.
    reasoning_effort: "low",
    context_window_k: 0,
    max_tokens: 16384,
    max_tokens_field: "",
  };
}

// Fields not defined by a template survive unchanged.
// Cloud templates keep entered credentials; Ollama and SDK templates clear them.
// The SDK bridge also clears unsupported proxy/session headers and enables native streaming.
// The original draft is not mutated, and no request or persistence occurs here.
export function applyLLMProfilePreset<T extends object>(draft: T, id: string): T & LLMProfilePresetFields {
  const preset = getLLMProfilePreset(id);
  if ("format" in draft && draft.format === "opencode-sdk" && preset.format !== "opencode-sdk") {
    return { ...draft, ...preset, api_key: "" };
  }
  return { ...draft, ...preset };
}

// A saved bridge token belongs only to that SDK endpoint, never to another format or host alias.
export function canReuseSDKBridgeToken(
  profile: { format: string; base_url?: string } | null | undefined,
  format: string,
  baseUrl: string,
): boolean {
  if (profile?.format !== "opencode-sdk" || format !== "opencode-sdk") return false;
  const endpoint = (value: string) => {
    const raw = value.trim() || "http://127.0.0.1:4097";
    const parts = /^https?:\/\/(\[[^\]]+\]|[^/:?#]+)(?::\d+)?(\/[^?#]*)?$/.exec(raw);
    if (!parts || raw.includes("\\")) return "";
    const url = new URL(raw);
    if (!["http:", "https:"].includes(url.protocol) || url.username || url.password || url.search || url.hash) return "";
    const host = parts[1].toLowerCase();
    if (!host.startsWith("[") && host !== url.hostname) return "";
    if (url.hostname !== "localhost" && url.hostname !== "[::1]" && !/^127\./.test(url.hostname)) return "";
    // Preserve path identity: the Go endpoint normalizer does not resolve dot segments.
    return url.origin + (parts[2] ?? "").replace(/\/+$/, "");
  };
  try {
    const previous = endpoint(profile.base_url ?? "");
    return previous !== "" && previous === endpoint(baseUrl);
  } catch {
    return false;
  }
}

// Choose only from the installed list, preferring confirmed native tool support.
export function selectInstalledOllamaModel(
  models: readonly string[],
  details: readonly { name?: string; model?: string; capabilities?: string[]; error?: string }[] = [],
): string {
  return models.find((model) => details.some((d) =>
    (d.name === model || d.model === model) && !d.error && d.capabilities?.includes("tools"),
  )) ?? models[0] ?? "";
}
