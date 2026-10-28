# OpenCode SDK bridge

[English](README.md) · [한국어](README.ko.md) · [简体中文](README.zh-Hans.md) · [繁體中文](README.zh-Hant.md) · [日本語](README.ja.md) · [Español](README.es.md)

Opt-in, loopback-only bridge for **native OpenCode CLI/client/SDK 2.1.4** and the separately tested **legacy SDK/CLI 1.19.4** pair. ScopeWeaver keeps its own conversation, tools, approvals, scope checks, retries, and compaction. The bridge streams a single model step and captures tool requests; it never runs ScopeWeaver tools or OpenCode's built-in tools. Existing API and OpenCode Go providers do not require this bridge.

## Start

Use Bun **1.3.14**, the tested CI runtime. Newer versions have not been verified:

```sh
cd adapters/opencode-sdk
bun install --frozen-lockfile --ignore-scripts
export SCOPEWEAVER_OPENCODE_TOKEN="$(openssl rand -hex 32)"
bun run start
```

The bridge token must contain at least **32 non-whitespace characters**. The private native CLI server uses a separately generated per-call password (`OPENCODE_PASSWORD`) and HTTP Basic authentication. That password is distinct from the bridge token and upstream provider keys.

In the ScopeWeaver provider profile, select `opencode-sdk`, set the endpoint to `http://127.0.0.1:4097`, and enter that **bridge token**. It is not a cloud API key. Discover models, then select an exact `providerID/modelID`. Do not reuse an OpenCode Go/Console token as the bridge token.

The default signed-out runtime uses its own isolated catalog and public model discovery. Discovery makes public catalog GET requests, **never inference**. It exposes only currently listed zero-price OpenCode models that the pinned runtime recognizes. Catalog failures fail explicitly. A listed model does not prove available quota, eligibility, or successful inference; free models can change or disappear. No paid model or credential fallback is enabled by default.

**Prior verification on October 25, 2026 (KST):** `opencode/deepseek-v4.1-flash-free` completed a real ScopeWeaver conversation through the production native CLI/client 2.1.4 bridge. Three actual inference calls covered the UI connection test, a host tool request reviewed and approved before its file effect, and the final visible answer `FREE_OK`. The model came from the default zero-price catalog; the run used no API keys, account imports, model/response fixtures, spoofed identities or paid fallback. Native tools remained blocked and ScopeWeaver owned the host action.

Native inference uses the actual pinned CLI and official client APIs; inference-free discovery uses the embedded SDK. The earlier embedded capture-only SDK route received HTTP 403 `FreeTierError`. Legacy hosted free access remains unverified. Free models and service eligibility can change; this dated test establishes no numeric quota or continuing availability. `free_tier_restricted` reports a structured refusal without selecting a paid fallback.

`SCOPEWEAVER_OPENCODE_PORT` changes port 4097. `SCOPEWEAVER_OPENCODE_HOST` must remain `127.0.0.1`. Browser `Origin` requests and non-loopback URLs are rejected. Protect the token like a password. All API routes require it.

`SCOPEWEAVER_OPENCODE_TIMEOUT_MS` is an optional whole-call deadline, from **0 to 86,400,000 ms**. Default **0** leaves call duration under the host's cancellation control; there is no implicit two-minute model cutoff. Cancellation and shutdown still have a separate **1.5-second** hard-stop bound for an owned worker/process group. Public catalog discovery has a 10-second network deadline. Requests are limited to **16 MiB**, individual returned tool-argument JSON to **2,000,000 characters**, combined native model text/reasoning/arguments to **8,000,000 characters**, and worker output to **20,000,000 bytes**. Exceeding a bound fails explicitly and never truncates data.

## Dedicated provider configuration

For an explicit custom OpenAI-compatible model or a local fixture, set `SCOPEWEAVER_OPENCODE_PROVIDER_CONFIG` to a JSON file owned by the bridge operator. Keep it outside the checkout, or use the ignored name `provider-config.local.json`:

```json
{
  "providers": {
    "local": {
      "name": "Local model",
      "package": "@opencode/ai/providers/openai-compatible",
      "settings": {
        "baseURL": "http://127.0.0.1:8000/v1",
        "apiKey": "explicit-provider-key-if-needed"
      },
      "models": {
        "my-model": {
          "name": "My model",
          "capabilities": { "tools": true },
          "limit": { "context": 200000, "output": 32000 },
          "compatibility": { "maxTokensField": "max_tokens" }
        }
      }
    }
  }
}
```

Select `local/my-model`. This file replaces default discovery for that bridge. It does not load OpenCode's user/project config, `.env` files, plugins, MCP servers, or saved credentials. Only the explicit `apiKey` is provided to that provider. A custom remote endpoint can charge for inference; configuration does not make it free. Arbitrary SDK packages and extra configuration keys are rejected.

## Runtime boundary

Every call runs in a separate Bun worker with an allowlisted environment and owned temporary home/config/data paths. Native owns a private official CLI server, an actual CLI `run`, its temporary database, and a one-use authenticated loopback egress transport. The official client supplies discovery, session events, and interruption. No worker inherits cloud credentials, user auth files, proxies, or preload settings. Pinned optional platform binaries are used directly; `--ignore-scripts` remains valid and no global install is needed. Do not omit optional dependencies.

Native retains OpenCode's genuine tool definitions and original implementations. Every native permission is ASK and CLI `--auto` is absent. A public `execute.before` policy hook independently refuses every non-host tool before execution, including generated `execute`; a proposed native call fails the bridge request and never reaches ScopeWeaver. Only registered host capture functions may suspend and become host tool requests. Other agents are removed. Collision-free `sw_tool_N` aliases prevent host tools from replacing native definitions; descriptions retain exact original names and descriptions, and responses restore original host names.

Host system segments use the public custom-agent system setting verbatim, joined by two newlines. OpenCode's generated model/environment/tool context is additionally retained. A final truthful policy part tells the model that only functions marked `Host tool:` are enabled. Host conversation roles, tool IDs/results, and per-call limits/settings use the public context hook. This preserves the host's messages while adding native runtime context; it does not promise a system array containing only the host segments. Model-specific thinking/effort fields are forwarded; unsupported provider settings fail explicitly. A smaller declared model output limit caps the requested limit. Native usage components and cache counts remain separate.

The worker blocks any second model request. Its private egress transport forwards the genuine generated request to the original upstream URL with redirects disabled; it does not choose a different endpoint, model, identity, or credential. Host calls become visible only after interruption, session deletion, and owned process cleanup. Disconnect, DELETE, timeout, and shutdown terminate the POSIX worker process group, with a 1.5-second hard stop. Both native and legacy inference are rejected on Windows because that process-tree cleanup boundary is not implemented.

The CLI public surface cannot disable all system/MDM configuration discovery. Native refuses to start when an unmanaged source exists: `/Library/Application Support/opencode` or an `ai.opencode.managed.plist` under the user's/global `/Library/Managed Preferences` on macOS, or `/etc/opencode` on Linux. Inaccessible sources and broken symlinks also fail closed with `configuration_error`. It never reads or changes those sources.

In native mode, raw tool-argument JSON, including 64-bit integers, is preserved in completion, stream responses, and replayed host tool history. The parsed JavaScript projection is never executed. Unsafe numbers outside authoritative raw tool arguments, unsupported content, unknown native tool calls, duplicate tool IDs, and malformed responses fail closed. Tool schemas support inline definitions, local JSON Pointer `$ref`/`$defs`, and unions; remote references, `$id`, `$dynamicRef`, and unresolved references are rejected. Native initialization still loads its own built-in provider/catalog code; this is an application boundary, **not an operating-system sandbox**. A loopback bridge can use a remote model, so it does not satisfy ScopeWeaver's local-only inference policy.

## Runtime versions and legacy limits

| `SCOPEWEAVER_OPENCODE_RUNTIME` | Pinned pair | Capability `runtime` |
| --- | --- | --- |
| `native` (default) | official CLI, `@opencode/client`, and SDK **2.1.4** | `native` |
| `legacy` | `@opencode-ai/sdk/v2` and CLI **1.19.4** | `legacy` |

Set `SCOPEWEAVER_OPENCODE_RUNTIME=legacy` before starting to choose the older API. Both pairs were tested locally on **macOS arm64**; CI is configured to run their actual-runtime fixtures on Linux. Other SDK versions are rejected before inference. This is not a universal cross-version or platform guarantee.

Legacy uses the matching official optional platform binary directly. `--ignore-scripts` therefore remains valid: it does not invoke the uninitialized wrapper or a global `opencode` installation. Do not omit optional dependencies. Its owned CLI and loopback MCP capture server are terminated with the request worker's POSIX process group. **Legacy is rejected on Windows**, where that cleanup boundary is not implemented. Each CLI call uses its own private plugin package copy and lock, with no writable link into shared dependencies.

Legacy integers must have an absolute value no greater than **9,007,199,254,740,991**.

Legacy supports structurally valid, safely representable JSON/history. Whitespace and equivalent HTML/Unicode escapes may normalize; it does not promise lexical JSON fidelity. Unsafe parsed integers, opaque/signed reasoning history, and unsupported provider overrides fail explicitly; choose native mode for raw 64-bit argument fidelity. Legacy custom providers use the same explicit OpenAI-compatible configuration above, but only `max_tokens` compatibility, supported tool capability fields, and context/output limits are mapped. Unsupported `reasoningField` and capability overrides are rejected. Existing API/Go profiles remain available for inputs outside either SDK mode's tested boundary.

## Protocol and checks

Protocol **`scopeweaver-opencode-sdk`, version 1** has authenticated routes: `GET /v1/capabilities`, `GET /v1/models`, `POST /v1/complete`, `POST /v1/stream`, `DELETE /v1/requests/{request_id}`. Streaming uses NDJSON with actual native text/thinking deltas, captured tool-use blocks, terminal usage, and exactly one `message_stop` on success. Errors have bounded machine codes and redact provider messages. Explicit structured upstream billing/quota exhaustion is returned as `quota_exhausted` (HTTP 402 for complete); generic rate limits, auth errors, and 5xx responses are never relabeled as quota. Structured model context overflow is returned as `context_length_exceeded` (HTTP 400), preserving the host's compaction signal; generic 400 responses and message keywords are not enough. Structured `FreeTierError` is returned as `free_tier_restricted` (HTTP 403), separate from quota and generic forbidden responses. The bridge does not select a fallback model; the host's configured policy decides what happens next.

Example completion request (the stream endpoint accepts the same body):

```json
{
  "request_id": "unique-call-uuid",
  "session_id": "stable-host-conversation-uuid",
  "model": "local/my-model",
  "system": ["Host system instructions"],
  "messages": [{ "role": "user", "content": [{ "type": "text", "text": "Hello" }] }],
  "tools": [],
  "max_tokens": 64,
  "thinking": "disabled"
}
```

Optional fields are `session_id`, `temperature`, `stop` (strings), `thinking`, `thinking_type`, and `reasoning_effort`. Host content blocks are `text`, assistant `thinking`/`tool_use`, and user `tool_result` with text content. Tool declarations use `name`, `description`, and `input_schema`. Complete responses contain `message`, `stop_reason` (`end_turn`, `tool_use`, or `max_tokens`), and `usage` with separate input/output/cache-read/cache-write token counts. See [validation and serialization](src/protocol.mjs) and [transport](src/bridge.mjs) for the public wire contract and limits.

For `opencode-go`, `session_id` keeps `x-opencode-session` stable across fresh per-call SDK sessions; otherwise `request_id` is used. `opencode` preserves its genuine runtime-generated session and identity headers. Generic providers receive no gateway-specific session header. The genuine CLI User-Agent is preserved.

```sh
bun run check
bun test --timeout 30000
```

Tests drive the **actual pinned CLI/client/SDK pair** against local SSE model fixtures. They verify inference-free discovery, host history/settings, capture safety, native streaming/usage, raw numeric arguments, hostile native calls, concurrent isolation, and cancellation. They do not prove real hosted free-model entitlement. See [한국어](README.ko.md).

## ScopeWeaver profile and downgrade boundary

A blank token retains a saved credential only for the same SDK endpoint. Changing endpoint or format must not reuse the saved token; `localhost`, `127.0.0.1`, and `::1` are distinct credential scopes. The SDK template clears unsaved cloud credentials, proxy and custom session header, leaves the model blank, and does not discover, infer, save or activate. Use **Load models** explicitly; **Test connection** performs inference only when clicked.

The profile database `CHECK` constraint adds `opencode-sdk`. Older binaries cannot start while SDK profiles remain. Before downgrading, back up the database and use the current version to convert SDK profiles to an older supported format or remove them. Prior migration verification used disposable databases; it did not migrate a live database. See the [provider guide](../../docs/llm-providers.md#opencode-sdk-bridge).
