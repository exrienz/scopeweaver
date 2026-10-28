import { test, expect, beforeAll, afterAll, spyOn } from "bun:test";
import http from "node:http";
import * as fs from "node:fs/promises";
import { mkdtemp, rm, readFile, mkdir, writeFile, symlink } from "node:fs/promises";
import * as childProcess from "node:child_process";
import { createHash } from "node:crypto";
import { fileURLToPath } from "node:url";
import { execFileSync } from "node:child_process";
import { tmpdir } from "node:os";
import { existsSync } from "node:fs";
import { completeLegacy, discoverLegacyModels } from "../src/legacy.mjs";
import legacyPlugin from "../src/legacy-plugin.mjs";
import { createBridge } from "../src/bridge.mjs";
import { parseRequest } from "../src/protocol.mjs";
import { unmanagedConfigPaths } from "../src/native-cli.mjs";
let dependencySnapshot;
async function dependencyDigest() {
  const root = fileURLToPath(new URL("../", import.meta.url)), paths = [];
  for await (const path of new Bun.Glob("**/package.json").scan({ cwd: `${root}/node_modules`, onlyFiles: true, followSymlinks: false })) paths.push(`node_modules/${path}`);
  paths.push("package.json", "bun.lock"); paths.sort();
  const hash = createHash("sha256");
  for (const path of paths) { hash.update(path); hash.update(await readFile(`${root}/${path}`)); }
  return hash.digest("hex");
}
beforeAll(async () => { dependencySnapshot = await dependencyDigest(); });
afterAll(async () => { expect(await dependencyDigest()).toBe(dependencySnapshot); });
function cliPIDs() {
  return new Set(execFileSync("ps", ["-axo", "pid=,command="], { encoding: "utf8" }).split("\n")
    .filter(line => /opencode-[^\s/]+\/bin\/opencode serve --hostname 127\.0\.0\.1/.test(line)).map(line => Number(line.trim().split(/\s+/)[0])));
}
function noNewCLI(before) { expect([...cliPIDs()].filter(pid => !before.has(pid))).toEqual([]); }
const tool = { name: "host.read", description: "Capture only", input_schema: { type: "object", properties: { path: { type: "string" } }, required: ["path"], additionalProperties: false } };
const request = (extra = {}) => ({ request_id: crypto.randomUUID(), session_id: "host-session", model: "fixture/test", system: ["HOST SYSTEM EXACT"], messages: [{ role: "user", content: [{ type: "text", text: "HOST USER EXACT" }] }], tools: [tool], max_tokens: 37, temperature: 0.2, stop: ["END"], thinking: "disabled", ...extra });

test("legacy refuses unmanaged configuration entries before launching the CLI", async () => {
  const root = await mkdtemp(`${tmpdir()}/legacy-config-policy-`), lstat = fs.lstat;
  const paths = unmanagedConfigPaths(), entries = [`${root}/managed.json`, `${root}/managed-dir`];
  await writeFile(entries[0], "{}"); await mkdir(entries[1]);
  if (process.platform !== "win32") {
    entries.push(`${root}/broken`); await symlink(`${root}/absent`, entries.at(-1));
  }
  // Redirect only system-path probes to disposable entries, never alter host config.
  let present, entry;
  const probe = spyOn(fs, "lstat").mockImplementation(path => {
    if (paths.includes(path)) return path === present ? lstat(entry) : Promise.reject(Object.assign(new Error(), { code: "ENOENT" }));
    return lstat(path);
  });
  const spawn = spyOn(childProcess, "spawn").mockImplementation(() => { throw new Error("Unexpected CLI launch"); });
  try {
    const context = { root, providerConfig: { providers: {} } };
    for (present of paths) for (entry of entries) {
      await expect(discoverLegacyModels(context)).rejects.toMatchObject({ code: "configuration_error", status: 400 });
      await expect(completeLegacy(request(), context)).rejects.toMatchObject({ code: "configuration_error", status: 400 });
    }
    expect(spawn).not.toHaveBeenCalled();
    expect(existsSync(`${root}/legacy`)).toBe(false);
  } finally {
    spawn.mockRestore(); probe.mockRestore(); await rm(root, { recursive: true, force: true });
  }
});

async function fixture(mode, callback) {
  const root = await mkdtemp(`${tmpdir()}/scopeweaver-test-legacy-`), requests = [], events = [];
  let markStarted; const started = new Promise(resolve => { markStarted = resolve; });
  const server = http.createServer(async (req, res) => {
    let text = ""; for await (const chunk of req) text += chunk;
    if (req.url === "/v1/models") { res.writeHead(200, { "content-type": "application/json" }); res.end('{"data":[{"id":"test"}]}'); return; }
    const body = JSON.parse(text); requests.push(body); markStarted();
    if (mode === "redirect" && req.url !== "/escaped") { res.writeHead(307, { location: `http://127.0.0.1:${server.address().port}/escaped` }); res.end(); return; }
    if ([500, 429, 402].includes(mode)) { res.writeHead(mode, { "content-type": "application/json" }); res.end(mode === 402 ? '{"error":{"code":"insufficient_quota"}}' : '{"error":{"message":"fixture failure"}}'); return; }
    if (mode === "quota429") { res.writeHead(429, { "content-type": "application/json" }); res.end('{"error":{"code":"insufficient_quota"}}'); return; }
    if (["context400", "context413", "generic400"].includes(mode)) {
      res.writeHead(mode === "context413" ? 413 : 400, { "content-type": "application/json" });
      res.end(JSON.stringify({ error: mode === "context400" ? { code: "context_length_exceeded" } : mode === "context413" ? { type: "prompt_too_long" } : { code: "invalid_request_error", message: "context length exceeded" } })); return;
    }
    if (["freetier403", "generic403"].includes(mode)) {
      res.writeHead(403, { "content-type": "application/json" });
      res.end(JSON.stringify({ error: mode === "freetier403" ? { name: "FreeTierError", data: { message: "fixture restriction" } } : { code: "forbidden", message: "FreeTierError" } })); return;
    }
    if (mode === "hang") return;
    res.writeHead(200, { "content-type": "text/event-stream" });
    const emit = (delta, finish = null, usage) => res.write(`data: ${JSON.stringify({ id: "fixture", object: "chat.completion.chunk", created: 1, model: "test", choices: [{ index: 0, delta, finish_reason: finish }], ...(usage ? { usage } : {}) })}\n\n`);
    emit({ role: "assistant", content: "native text" });
    if (mode !== "text" && (requests.length === 1 || mode === "reuse")) {
      const name = body.tools?.[0]?.function?.name;
      const count = mode === "two" ? 2 : 1;
      for (let i = 0; i < count; i++) emit({ tool_calls: [{ index: i, id: `native-call-${i}`, type: "function", function: { name: mode === "evil" ? "bash" : name, arguments: mode === "unsafe" ? '{"id":9223372036854775807}' : mode === "evil" ? JSON.stringify({ command: `touch '${root}/NATIVE_BASH_RAN'` }) : JSON.stringify({ path: `file-${i}.txt` }) } }] });
      emit({}, "tool_calls", { prompt_tokens: 101, completion_tokens: 13, total_tokens: 114, prompt_tokens_details: { cached_tokens: 5 } });
    } else emit({}, "stop", { prompt_tokens: 101, completion_tokens: 13, total_tokens: 114, prompt_tokens_details: { cached_tokens: 5 } });
    res.end("data: [DONE]\n\n");
  });
  await mkdir(`${root}/.opencode/plugins`, { recursive: true });
  await writeFile(`${root}/.opencode/plugins/hostile.mjs`, `import {writeFileSync} from 'node:fs';export default async()=>{writeFileSync(${JSON.stringify(`${root}/PROJECT_PLUGIN_RAN`)},'bad');return {}}`);
  await new Promise(resolve => server.listen(0, "127.0.0.1", resolve));
  const providerConfig = { providers: { fixture: { name: "Fixture", package: "@opencode/ai/providers/openai-compatible", settings: { baseURL: `http://127.0.0.1:${server.address().port}/v1`, apiKey: "fixture-no-secret" }, models: { test: { name: "Fixture model", limit: { context: 200000, output: 777 }, compatibility: { maxTokensField: "max_tokens" } } } } } };
  try { return await callback({ root, providerConfig, requests, events, started, emit: e => events.push(e), signal: AbortSignal.timeout(45000) }); }
  finally { server.closeAllConnections(); await new Promise(resolve => server.close(resolve)); const loaded = existsSync(`${root}/PROJECT_PLUGIN_RAN`); await rm(root, { recursive: true, force: true }); expect(loaded).toBe(false); }
}

test("legacy actual CLI captures two tools, streams, gates the second step and forwards frozen settings", async () => {
  await fixture("two", async context => {
    const result = await completeLegacy(request({ reasoning_effort: "low" }), context);
    expect(context.requests).toHaveLength(1);
    expect(context.requests[0]).toMatchObject({ max_tokens: 37, temperature: 0.2, stop: ["END"], thinking: { type: "disabled" }, reasoning_effort: "low" });
    expect(context.requests[0].tools.map(t => t.function.name)).toEqual(["capture_host_0"]);
    expect(context.requests[0].tools[0].function.description).toContain(tool.name);
    expect(context.requests[0].tools[0].function.description).toContain(tool.description);
    expect(JSON.stringify(context.requests[0].messages)).toContain("HOST USER EXACT");
    expect(context.requests[0].messages.filter(message => message.role === "system").map(message => message.content).join("\n")).toBe("HOST SYSTEM EXACT");
    expect(result.stop_reason).toBe("tool_use");
    expect(result.message.content.filter(b => b.type === "tool_use").map(b => b.raw)).toEqual(['{"path":"file-0.txt"}', '{"path":"file-1.txt"}']);
    expect(result.usage).toEqual({ input_tokens: 96, output_tokens: 13, cache_read_tokens: 5, cache_write_tokens: 0 });
    expect(context.events.filter(e => e.type === "text_delta").map(e => e.text).join("")).toBe("native text");
    expect(context.events.at(-1).type).toBe("message_stop");
  });
}, 30000);

test("legacy discovers configured real models without inference and completes a tool-free turn", async () => {
  await fixture("text", async context => {
    expect(await discoverLegacyModels(context)).toEqual([{ id: "fixture/test", name: "Fixture model" }]);
    expect(context.requests).toHaveLength(0);
    const result = await completeLegacy(request({ tools: [] }), { ...context, root: `${context.root}/complete` });
    expect(result.stop_reason).toBe("end_turn"); expect(context.requests).toHaveLength(1);
    expect(context.requests[0].tools ?? []).toHaveLength(0);
  });
}, 30000);

test("legacy preserves host call/result IDs and safe JSON history while rejecting unsafe numbers", async () => {
  await fixture("text", async context => {
    const messages = [{ role: "user", content: [{ type: "text", text: "old user" }] }, { role: "assistant", content: [{ type: "tool_use", id: "host-call", name: tool.name, input: { path: "old" } }] }, { role: "user", content: [{ type: "tool_result", tool_use_id: "host-call", is_error: true, content: [{ type: "text", text: "HOST REAL FAILURE" }] }] }];
    await completeLegacy(request({ messages }), { ...context, rawHistory: { "host-call": '{"path":"old"}' } });
    const wire = JSON.stringify(context.requests[0].messages);
    expect(wire).toContain("host-call"); expect(wire).toContain("HOST REAL FAILURE");
    await completeLegacy(request({ messages }), { ...context, root: `${context.root}/whitespace`, rawHistory: { "host-call": '{ "path": "old" }' } });
    const call = context.requests[1].messages.flatMap(message => message.tool_calls ?? []).find(call => call.id === "host-call");
    expect(JSON.parse(call.function.arguments)).toEqual({ path: "old" });
    await expect(completeLegacy(request({ messages }), { ...context, root: `${context.root}/int64`, rawHistory: { "host-call": '{"id":9223372036854775807}' } })).rejects.toMatchObject({ code: "unsupported_input" });
    expect(context.requests).toHaveLength(2);
  });
}, 30000);

const safeHistoryEncodings = [
  // Exact input object captured from product QA's real Go encoding/json request.
  ["Go HTML escaping", String.raw`{"command":"printf 'gh49-allow-proof\\n' \u003e\u003e /tmp/scopeweaver-gh49-product-qa/app-legacy/allow.txt"}`, "printf 'gh49-allow-proof\\n' >> /tmp/scopeweaver-gh49-product-qa/app-legacy/allow.txt"],
  ["whitespace and escaped Unicode", String.raw`{
    "command" : "printf '\ud55c\uad6d\\n' \u003e\u003e /tmp/legacy-unicode.txt"
  }`, "printf '한국\\n' >> /tmp/legacy-unicode.txt"],
];
for (const [name, raw, command] of safeHistoryEncodings) test(`legacy continues actual host history with ${name}`, async () => {
  await fixture("text", async context => {
    const input = { command }, callID = "real-go-bash-call";
    const messages = [
      { role: "user", content: [{ type: "text", text: "run the allowed command" }] },
      { role: "assistant", content: [{ type: "tool_use", id: callID, name: "Bash", input }] },
      { role: "user", content: [{ type: "tool_result", tool_use_id: callID, content: [{ type: "text", text: "REAL HOST RESULT" }] }] },
    ];
    const serialized = JSON.stringify(request({ messages, tools: [] })).replace(JSON.stringify(input), raw);
    const parsed = parseRequest(serialized);
    expect(parsed.rawHistory[callID]).toBe(raw);
    const result = await completeLegacy(parsed.request, { ...context, rawHistory: parsed.rawHistory });
    expect(result.stop_reason).toBe("end_turn"); expect(context.requests).toHaveLength(1);
    const call = context.requests[0].messages.flatMap(message => message.tool_calls ?? []).find(call => call.id === callID);
    expect(JSON.parse(call.function.arguments)).toEqual(input);
    expect(JSON.parse(call.function.arguments).command).toBe(command);
    expect(JSON.stringify(context.requests[0].messages)).toContain("REAL HOST RESULT");
  });
}, 30000);

test("legacy rejects unsafe Go rawHistory numbers with zero model HTTP requests", async () => {
  await fixture("text", async context => {
    const input = { id: 1 }, callID = "go-int64-call";
    const messages = [
      { role: "user", content: [{ type: "text", text: "old user" }] },
      { role: "assistant", content: [{ type: "tool_use", id: callID, name: tool.name, input }] },
      { role: "user", content: [{ type: "tool_result", tool_use_id: callID, content: [{ type: "text", text: "old result" }] }] },
    ];
    const serialized = JSON.stringify(request({ messages })).replace(JSON.stringify(input), '{"id":9223372036854775807}');
    const parsed = parseRequest(serialized);
    await expect(completeLegacy(parsed.request, { ...context, rawHistory: parsed.rawHistory })).rejects.toMatchObject({ code: "unsupported_input" });
    expect(context.requests).toHaveLength(0); expect(existsSync(`${context.root}/legacy`)).toBe(false);
  });
}, 30000);

for (const thinking of ["enabled", "disabled"]) test(`legacy rejects opaque reasoning history before inference with thinking ${thinking}`, async () => {
  await fixture("text", async context => {
    const messages = [
      { role: "user", content: [{ type: "text", text: "old user" }] },
      { role: "assistant", content: [{ type: "thinking", thinking: "old thought", signature: "opaque-signature" }, { type: "text", text: "old answer" }] },
      { role: "user", content: [{ type: "text", text: "continue" }] },
    ];
    let error;
    try { await completeLegacy(request({ messages, tools: [], thinking }), context); } catch (caught) { error = caught; }
    expect(context.requests).toHaveLength(0);
    expect(existsSync(`${context.root}/legacy`)).toBe(false);
    expect(error).toMatchObject({ code: "unsupported_input" });
  });
}, 30000);

test("legacy keeps unsigned reasoning history supported through the actual converter", async () => {
  await fixture("text", async context => {
    const messages = [
      { role: "user", content: [{ type: "text", text: "old user" }] },
      { role: "assistant", content: [{ type: "thinking", thinking: "UNSIGNED HISTORY" }, { type: "text", text: "old answer" }] },
      { role: "user", content: [{ type: "text", text: "continue" }] },
    ];
    const result = await completeLegacy(request({ messages, tools: [], thinking: "enabled" }), context);
    expect(result.stop_reason).toBe("end_turn"); expect(context.requests).toHaveLength(1);
    expect(context.requests[0].messages.some(message => message.role === "assistant" && message.reasoning_content === "UNSIGNED HISTORY")).toBe(true);
  });
}, 30000);

test("legacy reuses a native call ID across two host turns without reusing host IDs", async () => {
  await fixture("reuse", async context => {
    const first = await completeLegacy(request(), context);
    const call = first.message.content.find(block => block.type === "tool_use");
    const next = request({ messages: [
      ...request().messages,
      { role: "assistant", content: [{ type: "tool_use", id: call.id, name: call.name, input: JSON.parse(call.raw) }] },
      { role: "user", content: [{ type: "tool_result", tool_use_id: call.id, is_error: false, content: [{ type: "text", text: "REAL HOST RESULT" }] }] },
    ] });
    const second = await completeLegacy(next, { ...context, root: `${context.root}/second`, rawHistory: { [call.id]: call.raw } });
    const nextCall = second.message.content.find(block => block.type === "tool_use");
    expect(nextCall.id).not.toBe(call.id); expect(context.requests).toHaveLength(2);
    expect(JSON.stringify(context.requests[1].messages)).toContain(call.id);
    expect(JSON.stringify(context.requests[1].messages)).toContain("REAL HOST RESULT");
  });
}, 45000);

for (const status of [500, 429]) test(`legacy ${status} cannot retry a second upstream network request`, async () => {
  await fixture(status, async context => {
    await expect(completeLegacy(request(), context)).rejects.toMatchObject({ code: "second_model_request" });
    expect(context.requests).toHaveLength(1); expect(context.events.some(e => e.type === "tool_use_start")).toBe(false);
  });
}, 30000);

test("legacy structured 402 quota remains distinct from generic 429", async () => {
  await fixture(402, async context => {
    await expect(completeLegacy(request(), context)).rejects.toMatchObject({ code: "quota_exhausted" });
    expect(context.requests).toHaveLength(1);
  });
}, 30000);

test("legacy retains structured 429 quota through the native retry boundary", async () => {
  await fixture("quota429", async context => {
    await expect(completeLegacy(request(), context)).rejects.toMatchObject({ code: "quota_exhausted" });
    expect(context.requests).toHaveLength(1);
  });
}, 30000);

for (const mode of ["context400", "context413"]) test(`legacy preserves structured ${mode} for host compaction`, async () => {
  await fixture(mode, async context => {
    await expect(completeLegacy(request(), context)).rejects.toMatchObject({ code: "context_length_exceeded", status: 400 });
    expect(context.requests).toHaveLength(1); expect(context.events.some(event => event.type === "tool_use_start")).toBe(false);
  });
}, 30000);

test("legacy does not classify generic 400 prose as context overflow", async () => {
  await fixture("generic400", async context => {
    await expect(completeLegacy(request(), context)).rejects.toMatchObject({ code: "sdk_failure" });
    expect(context.requests).toHaveLength(1);
  });
}, 30000);

test("legacy preserves structured FreeTierError as an explicit runtime restriction", async () => {
  await fixture("freetier403", async context => {
    await expect(completeLegacy(request(), context)).rejects.toMatchObject({ code: "free_tier_restricted", status: 403 });
    expect(context.requests).toHaveLength(1); expect(context.events.some(event => event.type === "tool_use_start")).toBe(false);
  });
}, 30000);

test("legacy does not classify generic 403 prose as a free tier restriction", async () => {
  await fixture("generic403", async context => {
    await expect(completeLegacy(request(), context)).rejects.toMatchObject({ code: "sdk_failure" });
    expect(context.requests).toHaveLength(1);
  });
}, 30000);

test("legacy retains redirect failure through both pre-network retry gates", async () => {
  const root = await mkdtemp(`${tmpdir()}/scopeweaver-test-legacy-redirect-`), paths = [];
  const server = http.createServer((req, res) => {
    paths.push(req.url);
    res.writeHead(307, { location: `http://127.0.0.1:${server.address().port}/escaped` }); res.end();
  });
  await new Promise(resolve => server.listen(0, "127.0.0.1", resolve));
  try {
    const state = `${root}/state.json`;
    await writeFile(state, JSON.stringify({ provider: "fixture", request: request(), allowed: [], history: [] }));
    const hooks = await legacyPlugin({}, { state }), config = {};
    await hooks.config(config);
    await hooks["chat.params"]({ sessionID: "redirect" }, { options: {} });
    const fetch = config.provider.fixture.options.fetch;
    await expect(fetch(`http://127.0.0.1:${server.address().port}/model`)).rejects.toMatchObject({ code: "UnexpectedRedirect" });
    await expect(hooks["chat.params"]({ sessionID: "redirect" }, { options: {} })).rejects.toThrow("SCOPEWEAVER_LEGACY_REDIRECT");
    await expect(fetch(`http://127.0.0.1:${server.address().port}/model`)).rejects.toThrow("SCOPEWEAVER_LEGACY_REDIRECT");
    expect(paths).toEqual(["/model"]);
  } finally {
    server.closeAllConnections(); await new Promise(resolve => server.close(resolve));
    await rm(root, { recursive: true, force: true });
  }
});

test("legacy refuses model redirects before the redirected request leaves the CLI", async () => {
  await fixture("redirect", async context => {
    await expect(completeLegacy(request(), context)).rejects.toMatchObject({ code: "sdk_failure" });
    expect(context.requests).toHaveLength(1);
    expect(context.events.some(event => event.type === "tool_use_start")).toBe(false);
  });
}, 30000);

test("legacy rejects an unadvertised native bash call without publishing host tools", async () => {
  await fixture("evil", async context => {
    await expect(completeLegacy(request(), context)).rejects.toMatchObject({ code: "protocol_error" });
    expect(context.requests).toHaveLength(1); expect(context.events.some(e => e.type === "tool_use_start")).toBe(false);
    expect(existsSync(`${context.root}/NATIVE_BASH_RAN`)).toBe(false);
  });
}, 30000);

test("legacy cancellation aborts owned inference and concurrent peer remains isolated", async () => {
  await Promise.all([fixture("hang", async context => {
    const controller = new AbortController();
    const completion = completeLegacy(request(), { ...context, signal: controller.signal });
    await context.started;
    controller.abort(); await expect(completion).rejects.toMatchObject({ code: "canceled" });
    expect(context.requests).toHaveLength(1);
  }), fixture("two", async context => {
    const result = await completeLegacy(request({ max_tokens: 19 }), context);
    expect(result.stop_reason).toBe("tool_use"); expect(context.requests[0].max_tokens).toBe(19); expect(context.requests).toHaveLength(1);
  })]);
}, 60000);

test("legacy rejects unsafe numeric output before delivering tool calls", async () => {
  await fixture("unsafe", async context => {
    await expect(completeLegacy(request(), context)).rejects.toMatchObject({ code: "protocol_error" });
    expect(context.events.some(e => e.type === "tool_use_start")).toBe(false); expect(context.requests).toHaveLength(1);
  });
}, 30000);

test("legacy bridge preserves local schema refs and retains the host tool contract", async () => {
  await fixture("two", async context => {
    const token = "legacy-ref-token-01234567890123456789";
    const bridge = createBridge({ token, runtime: "legacy", providerConfig: context.providerConfig });
    try {
      const schema = { type: "object", $defs: { path: { type: "string" } }, properties: { path: { $ref: "#/$defs/path" } }, required: ["path"], additionalProperties: false };
      const response = await bridge.fetch(new Request("http://127.0.0.1:4097/v1/complete", { method: "POST", headers: { Authorization: `Bearer ${token}`, "content-type": "application/json" }, body: JSON.stringify(request({ tools: [{ ...tool, input_schema: schema }] })) }));
      expect(response.status).toBe(200);
      const result = await response.json();
      expect(result.message.content.filter(block => block.type === "tool_use")).toHaveLength(2);
      expect(context.requests[0].tools[0].function.parameters).toMatchObject(schema);
      expect(context.requests).toHaveLength(1);
    } finally { await bridge.close(); }
  });
}, 30000);

test("legacy cancellation during startup cannot create an orphan CLI", async () => {
  await fixture("hang", async context => {
    const before = cliPIDs(), controller = new AbortController();
    const pending = completeLegacy(request(), { ...context, signal: controller.signal });
    setTimeout(() => controller.abort(), 20);
    await expect(pending).rejects.toMatchObject({ code: "canceled" });
    noNewCLI(before); expect(context.requests).toHaveLength(0);
  });
}, 30000);

for (const mode of ["disconnect", "shutdown"]) test(`legacy bridge ${mode} stops a blocked model and owned CLI`, async () => {
  await fixture("hang", async context => {
    const before = cliPIDs(), token = "legacy-test-token-01234567890123456789";
    const bridge = createBridge({ token, runtime: "legacy", providerConfig: context.providerConfig });
    const response = await bridge.fetch(new Request("http://127.0.0.1:4097/v1/stream", { method: "POST", headers: { Authorization: `Bearer ${token}`, "content-type": "application/json" }, body: JSON.stringify(request()) }));
    expect(response.status).toBe(200);
    const reader = response.body.getReader();
    try {
      await reader.read(); await context.started;
      expect([...cliPIDs()].filter(pid => !before.has(pid)).length).toBeGreaterThan(0);
      if (mode === "disconnect") await reader.cancel();
      await bridge.close();
      if (mode === "shutdown") await reader.cancel();
      noNewCLI(before); expect(context.requests).toHaveLength(1);
    } finally { await reader.cancel().catch(() => {}); await bridge.close(); }
  });
}, 30000);
