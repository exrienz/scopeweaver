import { createOpencodeClient } from "@opencode-ai/sdk/v2/client";
import { createRequire } from "node:module";
import { createServer } from "node:http";
import { spawn } from "node:child_process";
import { mkdir, writeFile, readFile, access, cp, lstat } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { randomUUID } from "node:crypto";
import { fileURLToPath } from "node:url";
import { BridgeError, fail, safeJSON } from "./protocol.mjs";
import { assertNoUnmanagedConfig } from "./native-cli.mjs";

const require = createRequire(import.meta.url);
const gateMarker = "SCOPEWEAVER_LEGACY_SECOND_REQUEST";
const canceled = () => new BridgeError("canceled", 499);
const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));
function nativeError(error) {
  const data = error?.data;
  if (data?.message === "SCOPEWEAVER_LEGACY_QUOTA") return new BridgeError("quota_exhausted", 402);
  if (data?.message === "SCOPEWEAVER_LEGACY_CONTEXT_LENGTH") return new BridgeError("context_length_exceeded", 400);
  if (data?.message === "SCOPEWEAVER_LEGACY_FREE_TIER") return new BridgeError("free_tier_restricted", 403);
  if (error?.name === "APIError" && data?.statusCode === 403 && typeof data.responseBody === "string" && data.responseBody.length < 100000) {
    try {
      const body = JSON.parse(data.responseBody);
      if ([body?.error?.code, body?.error?.type, body?.error?.name].includes("FreeTierError")) return new BridgeError("free_tier_restricted", 403);
    } catch {}
  }
  if (error?.name === "APIError" && [400, 413].includes(data?.statusCode) && typeof data.responseBody === "string" && data.responseBody.length < 100000) {
    try {
      const body = JSON.parse(data.responseBody), codes = [body?.error?.code, body?.error?.type];
      if (codes.some(code => ["context_length_exceeded", "model_context_window_exceeded", "context_window_exceeded", "prompt_too_long"].includes(code)))
        return new BridgeError("context_length_exceeded", 400);
    } catch {}
  }
  if (error?.name === "APIError" && [402, 429].includes(data?.statusCode) && typeof data.responseBody === "string" && data.responseBody.length < 100000) {
    try {
      const body = JSON.parse(data.responseBody);
      if (["insufficient_quota", "billing_hard_limit_reached", "billing_not_active", "credit_balance_exhausted", "quota_exceeded"].includes(body?.error?.code))
        return new BridgeError("quota_exhausted", 402);
    } catch {}
  }
  return new BridgeError(data?.message === gateMarker ? "second_model_request" : "sdk_failure", 502);
}

async function binary() {
  const platform = { darwin: "darwin", linux: "linux", win32: "windows" }[process.platform];
  if (!platform || !["arm64", "x64"].includes(process.arch)) fail("unsupported_runtime");
  // opencode-ai's postinstall is intentionally unnecessary (--ignore-scripts supported).
  let musl = false;
  if (process.platform === "linux" && !process.report?.getReport?.()?.header?.glibcVersionRuntime) {
    try { await access(`/lib/ld-musl-${process.arch === "arm64" ? "aarch64" : "x86_64"}.so.1`); musl = true; } catch {}
  }
  const base = `opencode-${platform}-${process.arch}`, suffix = musl ? "-musl" : "";
  const names = process.arch === "x64" ? [`${base}-baseline${suffix}`, `${base}${suffix}`] : [`${base}${suffix}`];
  let path;
  for (const name of names) {
    try { path = require.resolve(`${name}/bin/opencode${process.platform === "win32" ? ".exe" : ""}`); break; } catch {}
  }
  if (!path) fail("unsupported_runtime");
  if (JSON.parse(await readFile(resolve(dirname(path), "../package.json"), "utf8")).version !== "1.19.4") fail("unsupported_runtime");
  await access(path);
  return path;
}

function aliases(request) {
  const forward = new Map(), reverse = new Map();
  const names = [...request.tools.map(tool => tool.name), ...request.messages.flatMap(message => message.content.filter(b => b.type === "tool_use").map(b => b.name))];
  for (const name of names) if (!forward.has(name)) {
    let local = /^[A-Za-z_][A-Za-z0-9_-]{0,55}$/.test(name) ? name : `host_${forward.size}`;
    let suffix = 0;
    while (reverse.has(`capture_${local}`)) local = `host_${forward.size}_${++suffix}`;
    const alias = `capture_${local}`;
    forward.set(name, alias); reverse.set(alias, name);
  }
  return { forward, reverse };
}

function history(request, map, rawHistory) {
  // CLI 1.x carries structural JSON. Normal whitespace and string escapes
  // (including Go's HTML escaping) preserve values; unsafe numbers do not.
  // Native2 remains the lossless path for raw bytes and 64-bit IDs.
  for (const raw of Object.values(rawHistory)) {
    safeJSON(raw, "unsupported_input");
  }
  const results = new Map();
  for (const message of request.messages) for (const block of message.content)
    if (block.type === "tool_result") results.set(block.tool_use_id, block);
  const [providerID, ...modelParts] = request.model.split("/");
  const modelID = modelParts.join("/");
  return request.messages.map((message, index) => {
    const id = `msg_scopeweaver_${index}`, parts = [];
    for (const block of message.content) {
      const common = { id: `prt_scopeweaver_${index}_${parts.length}`, messageID: id };
      if (block.type === "text") parts.push({ ...common, type: "text", text: block.text });
      else if (block.type === "thinking") {
        // The pinned OpenAI-compatible converter preserves unsigned text only;
        // opaque provider signatures would be silently discarded, even if the
        // current request disables thinking. Reject before creating a runtime.
        if (block.signature !== undefined && block.signature !== "") fail("unsupported_input");
        if (request.thinking !== "disabled") parts.push({ ...common, type: "reasoning", text: block.thinking,
          time: { start: 0, end: 0 } });
      } else if (block.type === "tool_use") {
        const result = results.get(block.id); if (!result) fail("unsupported_input");
        const raw = rawHistory[block.id] ?? JSON.stringify(block.input);
        const input = safeJSON(raw, "unsupported_input");
        const text = result.content.map(part => part.text).join("\n");
        parts.push({ ...common, type: "tool", callID: block.id, tool: map.forward.get(block.name),
          state: result.is_error ? { status: "error", input, error: text, time: { start: 0, end: 0 } }
            : { status: "completed", input, output: text, title: "", metadata: {}, time: { start: 0, end: 0 } } });
      } else if (block.type !== "tool_result") fail("unsupported_input");
    }
    const info = message.role === "assistant"
      ? { id, role: "assistant", parentID: `msg_scopeweaver_${Math.max(0, index - 1)}`, modelID, providerID,
          agent: "capture", mode: "capture", path: { cwd: "/", root: "/" }, cost: 0,
          tokens: { input: 0, output: 0, reasoning: 0, cache: { read: 0, write: 0 } }, time: { created: 0, completed: 0 }, finish: "stop" }
      : { id, role: "user", time: { created: 0 }, agent: "capture", model: { providerID, modelID } };
    return { info, parts };
  }).filter(message => message.parts.length);
}

function providers(config) {
  if (!config) return {};
  const out = {};
  for (const [id, provider] of Object.entries(config.providers)) {
    // The trusted common configuration deliberately has a narrow legacy mapping.
    if (provider.package !== "@opencode/ai/providers/openai-compatible") fail("configuration_error");
    const models = {};
    for (const [modelID, model] of Object.entries(provider.models)) {
      if (model.compatibility?.maxTokensField && model.compatibility.maxTokensField !== "max_tokens") fail("unsupported_input");
      if (model.compatibility?.reasoningField || Object.keys(model.capabilities ?? {}).some(key => key !== "tools")) fail("unsupported_input");
      models[modelID] = { id: modelID, name: model.name ?? modelID, tool_call: model.capabilities?.tools ?? true, temperature: true,
        limit: { context: model.limit?.context ?? 200000, output: model.limit?.output ?? 32768 } };
    }
    out[id] = { name: provider.name ?? id, npm: "@ai-sdk/openai-compatible", options: provider.settings, models };
  }
  return out;
}

async function start({ root, providerConfig, request, map, rawHistory = {}, signal }) {
  await assertNoUnmanagedConfig();
  if (signal?.aborted) throw canceled();
  const runtime = `${root}/legacy`;
  for (const dir of ["home", "config", "data", "state", "cache", "work"]) await mkdir(`${runtime}/${dir}`, { recursive: true });
  // The CLI normally bootstraps its plugin package on startup. Give it a real,
  // private package tree and matching lock; never a writable link to our tree.
  const pluginRoot = resolve(dirname(fileURLToPath(import.meta.resolve("@opencode-ai/plugin"))), "..");
  if (JSON.parse(await readFile(`${pluginRoot}/package.json`, "utf8")).version !== "1.19.4") fail("unsupported_runtime");
  for (const dir of [`${runtime}/config`, `${runtime}/config/opencode`]) {
    await mkdir(dir, { recursive: true });
    const dependencies = { "@opencode-ai/plugin": "1.19.4" };
    await writeFile(`${dir}/package.json`, JSON.stringify({ dependencies }));
    await writeFile(`${dir}/package-lock.json`, JSON.stringify({ lockfileVersion: 3, packages: { "": { dependencies } } }));
    try { if ((await lstat(`${dir}/node_modules`)).isSymbolicLink()) fail("configuration_error"); }
    catch (error) { if (error.code !== "ENOENT") throw error; }
    await mkdir(`${dir}/node_modules/@opencode-ai`, { recursive: true });
    await cp(pluginRoot, `${dir}/node_modules/@opencode-ai/plugin`, { recursive: true,
      filter: source => !source.slice(pluginRoot.length).split(/[\\/]/).includes("node_modules") });
  }
  const configured = providers(providerConfig), captures = new Map();
  let captureServer, processHandle, stopped = false, captureListening = false, captureClosing;
  const stopNow = () => {
    if (processHandle && processHandle.exitCode === null && processHandle.signalCode === null) processHandle.kill("SIGTERM");
    if (captureServer && captureListening && !captureClosing) {
      captureServer.closeAllConnections(); captureClosing = new Promise(resolve => captureServer.close(() => resolve()));
    }
  };
  signal?.addEventListener("abort", stopNow, { once: true });
  const close = async () => {
    if (stopped) return; stopped = true;
    signal?.removeEventListener("abort", stopNow);
    stopNow();
    const exited = () => processHandle.exitCode !== null || processHandle.signalCode !== null;
    if (processHandle && !exited()) {
      processHandle.kill("SIGTERM");
      await Promise.race([new Promise(resolve => processHandle.once("exit", resolve)), sleep(1500)]);
      if (!exited()) {
        processHandle.kill("SIGKILL");
        await Promise.race([new Promise(resolve => processHandle.once("exit", resolve)), sleep(1500)]);
        if (!exited()) fail("sdk_failure", 502);
      }
    }
    if (captureClosing) await captureClosing;
  };
  try {
    const cfg = { autoupdate: false, share: "disabled", snapshot: false, compaction: { auto: false }, plugin: [],
      lsp: false, formatter: false, permission: { "*": "deny" }, tools: { "*": false },
      agent: { capture: { mode: "primary", permission: { "*": "deny" }, tools: { "*": false } }, title: { disable: true }, summary: { disable: true }, compaction: { disable: true } },
      default_agent: "capture", provider: configured };
    if (request) {
      const separator = request.model.indexOf("/"), provider = request.model.slice(0, separator), model = request.model.slice(separator + 1);
      cfg.model = request.model; cfg.small_model = request.model;
      if (providerConfig && !configured[provider]?.models[model]) fail("unknown_model");
      if (request.tools.length && configured[provider]?.models[model]?.tool_call === false) fail("unsupported_input");
      const max = configured[provider]?.models[model]?.limit.output;
      if (max && request.max_tokens > max) request = { ...request, max_tokens: max };
      if (configured[provider]) configured[provider].models[model].limit.output = request.max_tokens;
      const statePath = `${runtime}/state.json`;
      await writeFile(statePath, JSON.stringify({ request, provider, allowed: request.tools.map(tool => map.forward.get(tool.name)), history: history(request, map, rawHistory) }), { mode: 0o600 });
      cfg.plugin = [[new URL("./legacy-plugin.mjs", import.meta.url).href, { state: statePath }]];
      const token = randomUUID(), schemas = request.tools.map(tool => ({ name: map.forward.get(tool.name).slice("capture_".length), description: `ScopeWeaver host tool ${tool.name}. Invoke this advertised function when host instructions refer to ${tool.name}.\n${tool.description}`, inputSchema: tool.input_schema }));
      captureServer = createServer(async (req, res) => {
        if (req.headers.authorization !== `Bearer ${token}`) { res.writeHead(401); res.end(); return; }
        if (req.method !== "POST") { res.writeHead(req.method === "DELETE" ? 200 : 405); res.end(); return; }
        try {
          let text = ""; for await (const chunk of req) { text += chunk; if (text.length > 2000000) throw Error(); }
          const call = safeJSON(text, "protocol_error");
          if (call.id === undefined) { res.writeHead(202); res.end(); return; }
          let result;
          if (call.method === "initialize") result = { protocolVersion: call.params.protocolVersion, capabilities: { tools: {} }, serverInfo: { name: "scopeweaver-capture", version: "1" } };
          else if (call.method === "tools/list") result = { tools: schemas };
          else if (call.method === "tools/call") {
            if (!schemas.some(schema => schema.name === call.params.name)) throw Error();
            const key = `capture_${call.params.name}`, values = captures.get(key) ?? [];
            values.push(JSON.stringify(call.params.arguments)); captures.set(key, values);
            result = { content: [{ type: "text", text: "deferred_to_host" }] };
          } else if (call.method === "ping") result = {};
          else { res.writeHead(400); res.end(); return; }
          res.writeHead(200, { "content-type": "application/json" }); res.end(JSON.stringify({ jsonrpc: "2.0", id: call.id, result }));
        } catch { res.writeHead(400); res.end(); }
      });
      await new Promise(resolve => captureServer.listen(0, "127.0.0.1", resolve));
      captureListening = true;
      if (signal?.aborted) throw canceled();
      cfg.mcp = { capture: { type: "remote", url: `http://127.0.0.1:${captureServer.address().port}/mcp`, headers: { Authorization: `Bearer ${token}` }, oauth: false, enabled: true, timeout: 10000 } };
      for (const tool of request.tools) {
        const name = map.forward.get(tool.name);
        cfg.permission[name] = "allow"; cfg.tools[name] = true;
        cfg.agent.capture.permission[name] = "allow"; cfg.agent.capture.tools[name] = true;
      }
    }
    await writeFile(`${runtime}/opencode.json`, JSON.stringify(cfg), { mode: 0o600 });
    const password = randomUUID();
    const env = { PATH: `${process.execPath.slice(0, process.execPath.lastIndexOf("/"))}:/usr/bin:/bin`, HOME: `${runtime}/home`,
      XDG_CONFIG_HOME: `${runtime}/config`, XDG_DATA_HOME: `${runtime}/data`, XDG_STATE_HOME: `${runtime}/state`, XDG_CACHE_HOME: `${runtime}/cache`,
      OPENCODE_CONFIG: `${runtime}/opencode.json`, OPENCODE_CONFIG_DIR: `${runtime}/config`, OPENCODE_DISABLE_DEFAULT_PLUGINS: "true",
      OPENCODE_DISABLE_AUTOUPDATE: "true", OPENCODE_DISABLE_MODELS_FETCH: "true", OPENCODE_DISABLE_LSP_DOWNLOAD: "true", OPENCODE_DISABLE_PROJECT_CONFIG: "true", OPENCODE_SERVER_PASSWORD: password, CI: "1" };
    // CLI treats --port 0 as its default port; reserve then release a real ephemeral port.
    const reservation = createServer(); await new Promise(resolve => reservation.listen(0, "127.0.0.1", resolve));
    const port = reservation.address().port; await new Promise(resolve => reservation.close(resolve));
    const executable = await binary();
    if (signal?.aborted) throw canceled();
    processHandle = spawn(executable, ["serve", "--hostname", "127.0.0.1", "--port", String(port)], { cwd: `${runtime}/work`, env, stdio: ["ignore", "pipe", "pipe"] });
    const url = await new Promise((resolve, reject) => {
      let text = "";
      const timer = setTimeout(() => reject(new BridgeError("sdk_failure", 502)), 20000);
      const abort = () => { clearTimeout(timer); reject(canceled()); };
      signal?.addEventListener("abort", abort, { once: true });
      const data = chunk => {
        text = (text + chunk).slice(-4000);
        if (text.includes(`http://127.0.0.1:${port}`)) { clearTimeout(timer); signal?.removeEventListener("abort", abort); resolve(`http://127.0.0.1:${port}`); }
      };
      processHandle.stdout.on("data", data); processHandle.stderr.on("data", data);
      processHandle.once("error", () => { clearTimeout(timer); signal?.removeEventListener("abort", abort); reject(new BridgeError("unsupported_runtime")); });
      processHandle.once("exit", () => { clearTimeout(timer); signal?.removeEventListener("abort", abort); reject(new BridgeError("sdk_failure", 502)); });
    });
    return { client: createOpencodeClient({ baseUrl: url, directory: `${runtime}/work`, headers: { Authorization: `Basic ${Buffer.from(`opencode:${password}`).toString("base64")}` }, throwOnError: true }), directory: `${runtime}/work`, close, stopNow, captures, request };
  } catch (error) { await close(); throw error; }
}

export async function discoverLegacyModels({ root, providerConfig }) {
  const runtime = await start({ root, providerConfig });
  try {
    const data = (await runtime.client.provider.list()).data;
    let publicIDs;
    if (!providerConfig) {
      const response = await fetch("https://opencode.ai/inference/v1/models", { redirect: "error", signal: AbortSignal.timeout(10000) });
      if (!response.ok) fail("sdk_failure", 502);
      const catalog = safeJSON(await response.text(), "sdk_failure"); publicIDs = new Set(catalog.data.map(model => model.id));
    }
    const models = data.all.flatMap(provider => Object.values(provider.models).filter(model => providerConfig
      ? Object.hasOwn(providerConfig.providers, provider.id) && Object.hasOwn(providerConfig.providers[provider.id].models, model.id)
      : provider.id === "opencode" && publicIDs.has(model.id) && model.cost?.input === 0 && model.cost?.output === 0)
      .map(model => ({ id: `${provider.id}/${model.id}`, name: model.name ?? model.id })));
    if (!models.length) fail("unknown_model");
    return models.sort((a, b) => a.id.localeCompare(b.id));
  } finally { await runtime.close(); }
}

export async function completeLegacy(request, { root, providerConfig, rawHistory = {}, signal, emit = () => {} }) {
  const map = aliases(request);
  history(request, map, rawHistory); // all representability failures precede SDK inference
  if (!providerConfig && !(await discoverLegacyModels({ root: `${root}/catalog` })).some(model => model.id === request.model)) fail("unknown_model");
  const runtime = await start({ root, providerConfig, request, map, rawHistory, signal });
  const { client, captures, directory } = runtime;
  const abortEvents = new AbortController(), blocks = new Map(), content = [], calls = new Map();
  let session, consumer, error, firstUsage, finish, ended = false, bytes = 0, wake;
  const notify = () => { wake?.(); wake = undefined; };
  const abort = () => { runtime.stopNow(); error = canceled(); notify(); };
  signal?.addEventListener("abort", abort, { once: true });
  const cleanupSession = async () => {
    if (!session) return;
    await Promise.race([(async () => {
      await client.session.abort({ sessionID: session.id, directory });
      await client.session.delete({ sessionID: session.id, directory });
    })(), sleep(1500)]);
    session = undefined;
  };
  try {
    if (signal?.aborted) throw canceled();
    const sub = await client.event.subscribe({ directory }, { signal: abortEvents.signal });
    consumer = (async () => {
      try { for await (const event of sub.stream) {
        if (event.properties?.sessionID !== session?.id && event.properties?.part?.sessionID !== session?.id) continue;
        const part = event.properties?.part;
        if (event.type === "message.part.updated" && part) {
          if (["text", "reasoning"].includes(part.type) && !blocks.has(part.id)) {
            const block = part.type === "text" ? { type: "text", text: "" } : { type: "thinking", thinking: "" };
            blocks.set(part.id, block); content.push(block);
          } else if (part.type === "tool") {
            if (!map.reverse.has(part.tool)) fail("protocol_error");
            if (!calls.has(part.callID)) {
              if (calls.size >= 128) fail("protocol_error");
              const call = { type: "tool_use", id: `sdk_${request.request_id}_${calls.size}`, name: map.reverse.get(part.tool) }; calls.set(part.callID, { call, nativeName: part.tool }); content.push(call);
            } else if (calls.get(part.callID).nativeName !== part.tool) fail("protocol_error");
            if (part.state.status === "error") fail("protocol_error");
            if (part.state.status === "completed") {
              const raw = JSON.stringify(part.state.input); safeJSON(raw, "protocol_error");
              const call = calls.get(part.callID).call;
              if (part.state.output !== "deferred_to_host" || (call.raw !== undefined && call.raw !== raw)) fail("protocol_error");
              if (call.raw === undefined) {
                const values = captures.get(part.tool), index = values?.indexOf(raw) ?? -1;
                if (index < 0 || raw.length > 2000000) fail("protocol_error");
                bytes += raw.length; if (bytes > 8000000) fail("protocol_error");
                values.splice(index, 1); call.raw = raw;
              }
            }
          } else if (part.type === "step-finish" && !firstUsage) {
            firstUsage = part.tokens; finish = part.reason;
          }
        } else if (event.type === "message.part.delta") {
          const delta = event.properties;
          if (delta.field !== "text") continue;
          const block = blocks.get(delta.partID); if (!block) fail("protocol_error");
          bytes += delta.delta.length; if (bytes > 8000000) fail("protocol_error");
          block[block.type === "text" ? "text" : "thinking"] += delta.delta;
          emit({ type: block.type === "text" ? "text_delta" : "thinking_delta", text: delta.delta });
        } else if (event.type === "session.error") {
          if (event.properties.error?.data?.message === gateMarker && firstUsage && calls.size) ended = true;
          else throw nativeError(event.properties.error);
        } else if (event.type === "session.status" && event.properties.status.type === "idle" && firstUsage && !calls.size) ended = true;
        notify();
      } } catch (e) { if (!abortEvents.signal.aborted) { error = e instanceof BridgeError ? e : new BridgeError("sdk_failure", 502); notify(); } }
    })();
    const permissions = [{ permission: "*", pattern: "*", action: "deny" }, ...request.tools.map(tool => ({ permission: map.forward.get(tool.name), pattern: "*", action: "allow" }))];
    session = (await client.session.create({ directory, title: "ScopeWeaver capture", permission: permissions })).data;
    emit({ type: "message_start" });
    const split = request.model.indexOf("/");
    await client.session.promptAsync({ sessionID: session.id, directory, model: { providerID: request.model.slice(0, split), modelID: request.model.slice(split + 1) }, agent: "capture", parts: [{ type: "text", text: "ScopeWeaver host turn" }] });
    while (!ended && !error) await new Promise(resolve => { wake = resolve; if (ended || error || signal?.aborted) notify(); });
    if (error) throw error;
    if (!firstUsage || !content.length || [...calls.values()].some(value => !value.call.raw)) fail("protocol_error");
    const usage = { input_tokens: firstUsage.input, output_tokens: firstUsage.output, cache_read_tokens: firstUsage.cache?.read ?? 0, cache_write_tokens: firstUsage.cache?.write ?? 0 };
    if (!Object.values(usage).every(value => Number.isSafeInteger(value) && value >= 0)) fail("protocol_error");
    const stop_reason = calls.size ? "tool_use" : ["length", "max-tokens"].includes(finish) ? "max_tokens" : finish === "stop" ? "end_turn" : undefined;
    if (!stop_reason) fail("protocol_error");
    await cleanupSession();
    abortEvents.abort(); await runtime.close();
    if (signal?.aborted) throw canceled();
    for (const { call } of calls.values()) { emit({ type: "tool_use_start", tool_id: call.id, tool_name: call.name }); emit({ type: "tool_input_delta", text: call.raw }); }
    emit({ type: "message_delta", stop_reason, usage }); emit({ type: "message_stop" });
    return { message: { role: "assistant", content }, stop_reason, usage };
  } finally {
    signal?.removeEventListener("abort", abort); abortEvents.abort();
    if (session) runtime.stopNow();
    try { await cleanupSession(); } catch {}
    await runtime.close(); if (consumer) await Promise.race([consumer, sleep(500)]);
  }
}
