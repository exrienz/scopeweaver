import assert from "node:assert/strict";
import { createServer } from "node:http";
import { once } from "node:events";
import test from "node:test";
import { z } from "zod";
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StdioClientTransport } from "@modelcontextprotocol/sdk/client/stdio.js";
import { cpSync, mkdtempSync, rmSync, symlinkSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";
import { configFromEnv, ScopeWeaverClient } from "../src/client.js";
import { MESSAGES, SUPPORTED_LANGUAGES, translator } from "../src/messages.js";
import { createTools, executeTool } from "../src/tools.js";
import extension from "../pi-extension.js";

async function fixture(t, handler) {
  const server = createServer(handler);
  server.listen(0, "127.0.0.1");
  await once(server, "listening");
  t.after(() => { server.closeAllConnections(); server.close(); });
  return `http://127.0.0.1:${server.address().port}`;
}

test("configuration rejects unsafe origins and invalid options without printing credentials", () => {
  for (const url of ["http://example.com", "https://secret@example.com", "https://example.com/api/", "https://example.com?token=secret", "file:///tmp/x"]) {
    assert.throws(() => configFromEnv({ SCOPEWEAVER_URL: url }), (error) => !error.message.includes("secret"));
  }
  for (const options of [{ SCOPEWEAVER_TIMEOUT_MS: "NaN" }, { SCOPEWEAVER_TIMEOUT_MS: "0" },
    { SCOPEWEAVER_LANGUAGE: "xx" }, { SCOPEWEAVER_ALLOW_WRITES: "yes" }, { SCOPEWEAVER_TOKEN: "x\ny" }]) {
    assert.throws(() => configFromEnv(options));
  }
});

test("all six canonical languages configure the adapter and reach Accept-Language unchanged", async (t) => {
  let expected;
  const url = await fixture(t, (req, res) => {
    assert.equal(req.headers["accept-language"], expected);
    assert.equal(req.headers.authorization, "Bearer fixture-token");
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ language: req.headers["accept-language"] }));
  });
  assert.equal(configFromEnv({}).language, "en");
  for (const language of ["en", "ko", "zh-Hans", "zh-Hant", "ja", "es"]) {
    expected = language;
    const config = configFromEnv({ SCOPEWEAVER_URL: url, SCOPEWEAVER_TOKEN: "fixture-token", SCOPEWEAVER_LANGUAGE: language });
    assert.equal(config.language, language);
    assert.deepEqual(await new ScopeWeaverClient(config).request("/api/settings"), { language });
  }
  for (const language of ["xx", "zh", "zh-CN", "zh-TW", "ja-JP", "es-MX", "EN", "en-US", "en\n"]) {
    assert.throws(() => configFromEnv({ SCOPEWEAVER_LANGUAGE: language }), /SCOPEWEAVER_LANGUAGE/);
  }
});

test("Korean adapter metadata and configuration diagnostics are localized", () => {
  const client = new ScopeWeaverClient(configFromEnv({ SCOPEWEAVER_LANGUAGE: "ko" }));
  assert.equal(createTools(client)[0].title, "ScopeWeaver 상태");
  assert.throws(() => configFromEnv({ SCOPEWEAVER_LANGUAGE: "ko", SCOPEWEAVER_TIMEOUT_MS: "0" }), /정수/);
});

test("writes are absent by default and unknown/path-injection arguments are rejected before HTTP", async () => {
  const client = new ScopeWeaverClient(configFromEnv({}));
  const reads = createTools(client);
  assert.equal(reads.length, 6);
  assert.ok(reads.every((tool) => tool.annotations.readOnlyHint));
  const tool = reads.find((tool) => tool.name === "scopeweaver_get_task");
  await assert.rejects(executeTool(tool, { task_id: "../../settings" }));
  await assert.rejects(executeTool(tool, { task_id: "1", command: "ignored" }));
  const writes = createTools(new ScopeWeaverClient(configFromEnv({ SCOPEWEAVER_ALLOW_WRITES: "true" })));
  assert.equal(writes.length, 8);
  const create = writes.find((tool) => tool.name === "scopeweaver_create_task");
  assert.equal(create.schema.parse({ description: "lab", goal: "test" }).timeout_seconds, 600);
  await assert.rejects(executeTool(create, { description: " ", goal: "test" }));
  await assert.rejects(executeTool(create, { description: "lab", goal: "test", timeout_seconds: 0 }));
});

test("concurrent calls share login, send authorization and Korean locale, and paginate tasks", async (t) => {
  let logins = 0;
  const url = await fixture(t, async (req, res) => {
    res.setHeader("Content-Type", "application/json");
    if (req.url === "/api/auth/login") {
      logins++;
      let body = "";
      for await (const chunk of req) body += chunk;
      assert.deepEqual(JSON.parse(body), { username: "ARTEX", password: "fixture-password" });
      return res.end(JSON.stringify({ token: "fixture-token" }));
    }
    assert.equal(req.headers.authorization, "Bearer fixture-token");
    assert.equal(req.headers["accept-language"], "ko");
    res.end(JSON.stringify({ tasks: [{ id: "3" }, { id: "2" }, { id: "1" }], active: "3" }));
  });
  const client = new ScopeWeaverClient(configFromEnv({ SCOPEWEAVER_URL: url, SCOPEWEAVER_PASSWORD: "fixture-password", SCOPEWEAVER_LANGUAGE: "ko" }));
  const tool = createTools(client).find((tool) => tool.name === "scopeweaver_list_tasks");
  const results = await Promise.all([executeTool(tool, { page: 2, limit: 1 }), executeTool(tool, {})]);
  assert.equal(logins, 1);
  assert.deepEqual(results[0].structuredContent.data.tasks, [{ id: "2" }]);
  assert.equal(results[0].structuredContent.data.total, 3);
});

test("backend errors redact credentials and writes are never retried", async (t) => {
  let requests = 0;
  const url = await fixture(t, (req, res) => {
    requests++;
    res.writeHead(401, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ error: "fixture-token fixture-password" }));
  });
  const client = new ScopeWeaverClient(configFromEnv({ SCOPEWEAVER_URL: url, SCOPEWEAVER_TOKEN: "fixture-token", SCOPEWEAVER_PASSWORD: "fixture-password" }));
  await assert.rejects(client.request("/api/tasks", { method: "POST", body: {} }), (error) => {
    assert.match(error.message, /HTTP 401.*redacted.*Sign in again/);
    assert.ok(!error.message.includes("fixture-token"));
    assert.ok(!error.message.includes("fixture-password"));
    return true;
  });
  assert.equal(requests, 1);
});

test("redirects do not forward credentials to another server", async (t) => {
  let leaked = false;
  const destination = await fixture(t, (_, res) => { leaked = true; res.end("{}"); });
  const url = await fixture(t, (_, res) => { res.writeHead(302, { Location: destination }); res.end(); });
  const client = new ScopeWeaverClient(configFromEnv({ SCOPEWEAVER_URL: url, SCOPEWEAVER_TOKEN: "fixture-token" }));
  await assert.rejects(client.request("/api/tasks"));
  assert.equal(leaked, false);
});

test("timeouts, cancellation, malformed JSON and excessive responses fail explicitly", async (t) => {
  const url = await fixture(t, (req, res) => {
    if (req.url === "/api/invalid") return res.end("html");
    if (req.url === "/api/large") return res.end("x".repeat(2 * 1024 * 1024 + 1));
    // /api/hang intentionally leaves the response open.
  });
  const client = new ScopeWeaverClient(configFromEnv({ SCOPEWEAVER_URL: url, SCOPEWEAVER_TOKEN: "fixture-token", SCOPEWEAVER_TIMEOUT_MS: "100" }));
  await assert.rejects(client.request("/api/invalid"), /non-JSON/);
  await assert.rejects(client.request("/api/large"), /exceeds 2 MiB/);
  await assert.rejects(client.request("/api/hang"), /timed out/);
  const controller = new AbortController();
  controller.abort();
  await assert.rejects(client.request("/api/hang", { signal: controller.signal }), /cancelled/);
});

test("Pi extension shares schemas and reports backend failures by throwing", async () => {
  const tools = [];
  extension({ registerTool: (tool) => tools.push(tool) });
  assert.equal(tools.length, process.env.SCOPEWEAVER_ALLOW_WRITES === "true" ? 8 : 6);
  const task = tools.find((tool) => tool.name === "scopeweaver_get_task");
  assert.equal(task.parameters.type, "object");
  await assert.rejects(task.execute("id", { task_id: "../settings" }), /Invalid/);
});

const adapterRoot = fileURLToPath(new URL("..", import.meta.url));
const withoutHumanMetadata = schema => {
  if (!schema || typeof schema !== "object" || Array.isArray(schema)) return schema;
  const { title, description, ...machine } = schema;
  // Recurse only into the schema locations we emit; const/default/enum are raw data.
  if (schema.properties) machine.properties = Object.fromEntries(Object.entries(schema.properties)
    .map(([name, value]) => [name, withoutHumanMetadata(value)]));
  if (schema.items) machine.items = withoutHumanMetadata(schema.items);
  return machine;
};
const clientFor = (language, extra = {}) => new ScopeWeaverClient(configFromEnv({
  SCOPEWEAVER_LANGUAGE: language, SCOPEWEAVER_ALLOW_WRITES: "true", SCOPEWEAVER_TOKEN: "fixture-token", ...extra,
}));

test("schema comparison preserves description properties and literal const/default/enum data", () => {
  const baseline = {
    type: "object", title: "Human title", description: "Human description",
    properties: {
      description: { type: "string", minLength: 1, maxLength: 10000, description: "Human parameter description" },
      title: { type: "string", title: "Human parameter title" },
      values: { type: "array", items: { type: "string", description: "Human item description" } },
    },
    required: ["description"], additionalProperties: false,
    const: { description: "literal const", title: "literal const title" },
    default: { description: "literal default", title: "literal default title" },
    enum: [{ description: "literal enum", title: "literal enum title" }],
  };
  const normalized = withoutHumanMetadata(baseline);
  for (const mutate of [
    schema => { delete schema.properties.description; },
    schema => { schema.properties.description.type = "number"; },
    schema => { schema.properties.description.maxLength = 9999; },
    schema => { schema.default.description = "changed default"; },
    schema => { schema.const.title = "changed const"; },
    schema => { schema.enum[0].description = "changed enum"; },
  ]) {
    const changed = structuredClone(baseline);
    mutate(changed);
    assert.notDeepEqual(withoutHumanMetadata(changed), normalized, "machine schema or literal data change was hidden");
  }
  const translated = structuredClone(baseline);
  translated.title = "번역 제목";
  translated.description = "번역 설명";
  translated.properties.description.description = "매개변수 설명";
  translated.properties.title.title = "매개변수 제목";
  translated.properties.values.items.description = "항목 설명";
  assert.deepEqual(withoutHumanMetadata(translated), normalized);
  assert.deepEqual(normalized.properties.description, { type: "string", minLength: 1, maxLength: 10000 });
  assert.deepEqual(normalized.default, baseline.default);
  assert.deepEqual(normalized.const, baseline.const);
  assert.deepEqual(normalized.enum, baseline.enum);
});
const invalidArguments = [
  ["get_task", {}], ["get_task", { task_id: 3 }], ["health", { extra: "literal" }],
  ["get_task", { task_id: "../settings" }], ["get_task", { task_id: "1".repeat(20) }],
  ["list_findings", { severity: "urgent" }], ["list_tasks", { page: 0 }],
  ["list_tasks", { page: 1.2 }], ["list_tasks", { limit: 101 }],
  ["create_task", { description: " ", goal: "literal" }],
  ["create_task", { description: "literal", goal: "literal", company_ids: [0] }],
  ["create_task", { description: "literal", goal: "literal", company_ids: Array(33).fill(1) }],
  ["create_task", { description: "literal", goal: "literal", company_ids: "1" }],
  ["control_task", { task_id: "1", action: "delete" }],
];

test("adapter catalogs are complete with matching placeholders and literal technical tokens", () => {
  assert.deepEqual(SUPPORTED_LANGUAGES, ["en", "ko", "zh-Hans", "zh-Hant", "ja", "es"]);
  const keys = Object.keys(MESSAGES.en).sort();
  const placeholders = text => [...text.matchAll(/\{([a-z]+)\}/g)].map(match => match[1]).sort();
  for (const language of SUPPORTED_LANGUAGES) {
    assert.deepEqual(Object.keys(MESSAGES[language]).sort(), keys, language);
    for (const key of keys) {
      assert.ok(typeof MESSAGES[language][key] === "string" && MESSAGES[language][key].trim(), `${language}:${key}`);
      assert.deepEqual(placeholders(MESSAGES[language][key]), placeholders(MESSAGES.en[key]), `${language}:${key}`);
      if (key.startsWith("config.")) for (const token of MESSAGES.en[key].match(/SCOPEWEAVER_[A-Z_]+/g) || []) assert.ok(MESSAGES[language][key].includes(token));
    }
    assert.ok(MESSAGES[language]["response.oversized"].includes("2 MiB"));
    assert.ok(MESSAGES[language]["config.timeout"].includes("120000"));
    assert.ok(MESSAGES[language]["tools.create_task.description"].includes("get_task"));
  }
});

test("six language tool schemas preserve all machine contracts and localized validation is client-local", async (t) => {
  const before = z.config().localeError;
  const all = Object.fromEntries(SUPPORTED_LANGUAGES.map(language => [language, createTools(clientFor(language))]));
  const byName = (tools, short) => tools.find(tool => tool.name === `scopeweaver_${short}`);
  for (const language of SUPPORTED_LANGUAGES) {
    for (let i = 0; i < all.en.length; i++) {
      const tool = all[language][i], english = all.en[i];
      assert.equal(tool.name, english.name);
      assert.deepEqual(tool.annotations, english.annotations);
      assert.deepEqual(withoutHumanMetadata(z.toJSONSchema(tool.schema)), withoutHumanMetadata(z.toJSONSchema(english.schema)));
      assert.deepEqual(z.toJSONSchema(tool.outputSchema), z.toJSONSchema(english.outputSchema));
      assert.equal(tool.title, MESSAGES[language][`tools.${tool.name.slice(12)}.title`]);
      assert.equal(tool.description, MESSAGES[language][`tools.${tool.name.slice(12)}.description`]);
    }
    for (const [short, args] of invalidArguments) {
      const result = byName(all[language], short).schema.safeParse(args);
      const english = byName(all.en, short).schema.safeParse(args);
      assert.equal(result.success, false, `${language}:${short}`);
      assert.deepEqual(result.error.issues.map(({ message, ...issue }) => issue), english.error.issues.map(({ message, ...issue }) => issue));
      if (language !== "en") assert.notEqual(result.error.issues[0].message, english.error.issues[0].message, `${language}:${short}`);
      await assert.rejects(executeTool(byName(all[language], short), args), error => error.issues?.[0].message === result.error.issues[0].message);
    }
  }
  const url = await fixture(t, (_, res) => { res.writeHead(500, { "Content-Type": "application/json" }); res.end("{}"); });
  const isolated = Object.fromEntries(SUPPORTED_LANGUAGES.map(language => [language, clientFor(language, { SCOPEWEAVER_URL: url })]));
  for (const language of ["ja", "en", "ko", "ja", "es", "en"]) {
    const local = byName(all[language], "get_task").schema.safeParse({});
    assert.ok(local.error.message.includes(MESSAGES[language]["validation.invalid_type"].split("{")[0]));
    await assert.rejects(isolated[language].request("/api/settings"), {
      message: translator(language)("response.http", { status: 500, detail: MESSAGES[language]["response.failed"], hint: "" }),
    });
  }
  assert.equal(z.config().localeError, before, "adapter changed global Zod language");
});

test("MCP startup diagnostics use the selected language without stdout or credential leakage", () => {
  for (const language of SUPPORTED_LANGUAGES) {
    const result = spawnSync(process.execPath, ["src/mcp.js"], {
      cwd: adapterRoot, encoding: "utf8", timeout: 10000,
      env: { ...process.env, SCOPEWEAVER_LANGUAGE: language, SCOPEWEAVER_TIMEOUT_MS: "0", SCOPEWEAVER_URL: "http://127.0.0.1:8787", SCOPEWEAVER_TOKEN: "fixture-secret-token" },
    });
    assert.equal(result.status, 1, result.error?.message || result.stderr);
    assert.equal(result.stdout, "");
    assert.equal(result.stderr.trim(), MESSAGES[language]["config.timeout"]);
    assert.ok(!result.stderr.includes("fixture-secret-token"));
  }
});

test("configuration and authentication diagnostics localize before and after client creation without credentials", async (t) => {
  const url = await fixture(t, (_, res) => { res.setHeader("Content-Type", "application/json"); res.end('{"token":""}'); });
  for (const language of SUPPORTED_LANGUAGES) {
    const local = translator(language);
    for (const [extra, key] of [
      [{ SCOPEWEAVER_URL: "https://secret@example.com" }, "config.url_origin"],
      [{ SCOPEWEAVER_URL: "not a URL secret" }, "config.url_invalid"],
      [{ SCOPEWEAVER_TIMEOUT_MS: "0" }, "config.timeout"],
      [{ SCOPEWEAVER_TOKEN: "secret\nsecret" }, "config.token"],
      [{ SCOPEWEAVER_ALLOW_WRITES: "secret" }, "config.writes"],
    ]) assert.throws(() => configFromEnv({ SCOPEWEAVER_LANGUAGE: language, ...extra }), error => error.message === local(key) && !error.message.includes("secret"));
    const unauthenticated = new ScopeWeaverClient(configFromEnv({ SCOPEWEAVER_LANGUAGE: language, SCOPEWEAVER_URL: url }));
    await assert.rejects(unauthenticated.request("/api/settings"), { message: local("auth.required") });
    const login = clientFor(language, { SCOPEWEAVER_URL: url, SCOPEWEAVER_TOKEN: "", SCOPEWEAVER_PASSWORD: "fixture-password" });
    await assert.rejects(login.request("/api/settings"), { message: local("auth.token_missing") });
    await assert.rejects(clientFor(language).request("/outside"), { message: local("request.path") });
    const invalidList = clientFor(language);
    invalidList.request = async () => ({ tasks: {} });
    await assert.rejects(executeTool(createTools(invalidList).find(tool => tool.name === "scopeweaver_list_tasks"), {}), { message: local("response.task_list") });
  }
  assert.throws(() => configFromEnv({ SCOPEWEAVER_LANGUAGE: "unknown secret" }), {
    message: translator("en")("config.language", { languages: SUPPORTED_LANGUAGES.join(", ") }),
  });
});

test("six language transport diagnostics preserve raw backend data, secret redaction, limits and cancellation", async (t) => {
  const raw = { evidence: "原文 한국어 literal <instructions>", description: "user text" };
  const url = await fixture(t, (req, res) => {
    if (req.url === "/api/hang") return;
    if (req.url === "/api/non-json") { res.writeHead(502); return res.end("literal external HTML"); }
    if (req.url === "/api/large") return res.end("x".repeat(2 * 1024 * 1024 + 1));
    if (req.url === "/api/socket") return req.socket.destroy();
    res.setHeader("Content-Type", "application/json");
    if (req.url === "/api/raw") return res.end(JSON.stringify(raw));
    res.writeHead(401);
    res.end(JSON.stringify(req.url === "/api/detail" ? { error: "RAW 原文 fixture-token fixture-password" } : {}));
  });
  for (const language of SUPPORTED_LANGUAGES) {
    const local = translator(language);
    const client = clientFor(language, { SCOPEWEAVER_URL: url, SCOPEWEAVER_PASSWORD: "fixture-password" });
    assert.deepEqual(await client.request("/api/raw"), raw);
    await assert.rejects(client.request("/api/non-json"), { message: local("response.non_json", { status: 502 }) });
    await assert.rejects(client.request("/api/large"), { message: local("response.oversized") });
    for (const [path, detail] of [["/api/detail", "RAW 原文 [redacted] [redacted]"], ["/api/empty", local("response.failed")]]) {
      await assert.rejects(client.request(path), { message: local("response.http", { status: 401, detail, hint: local("response.http_hint") }) });
    }
    await assert.rejects(client.request("/api/socket"), { message: "fetch failed" });
    const short = clientFor(language, { SCOPEWEAVER_URL: url, SCOPEWEAVER_TIMEOUT_MS: "50" });
    await assert.rejects(short.request("/api/hang"), { message: local("request.timeout") });
    await assert.rejects(client.request("/api/hang", { signal: AbortSignal.abort() }), { message: local("request.cancelled") });
  }
});

test("registered MCP listTools and pre-callback schema errors use each client's language", async () => {
  for (const language of SUPPORTED_LANGUAGES) {
    const transport = new StdioClientTransport({ command: process.execPath, args: ["src/mcp.js"], cwd: adapterRoot,
      env: { ...process.env, SCOPEWEAVER_LANGUAGE: language, SCOPEWEAVER_ALLOW_WRITES: "true", SCOPEWEAVER_TOKEN: "fixture-token", SCOPEWEAVER_PASSWORD: "fixture-password" }, stderr: "pipe" });
    const client = new Client({ name: "adapter-language-test", version: "1.0.0" });
    try {
      await client.connect(transport);
      const listed = await client.listTools();
      assert.equal(listed.tools.length, 8);
      for (const tool of listed.tools) {
        assert.equal(tool.title, MESSAGES[language][`tools.${tool.name.slice(12)}.title`]);
        assert.equal(tool.description, MESSAGES[language][`tools.${tool.name.slice(12)}.description`]);
      }
      assert.equal(listed.tools.find(tool => tool.name === "scopeweaver_get_task").inputSchema.properties.task_id.description,
        MESSAGES[language]["schema.id.description"]);
      const local = createTools(clientFor(language));
      for (const [short, args] of invalidArguments.slice(0, 9)) {
        const result = await client.callTool({ name: `scopeweaver_${short}`, arguments: args });
        assert.equal(result.isError, true);
        const text = result.content.map(content => content.text || "").join("");
        assert.ok(text.includes("Input validation error"), "SDK-owned wrapper should remain library text");
        const expected = local.find(tool => tool.name === `scopeweaver_${short}`).schema.safeParse(args).error.issues[0].message;
        assert.ok(text.includes(expected), `${language}:${short}: ${text}`);
        assert.ok(!text.includes("fixture-token") && !text.includes("fixture-password"));
      }
    } finally { await client.close(); }
  }
});

test("Pi registers translated labels, descriptions and schema errors without changing identifiers", async () => {
  const names = ["SCOPEWEAVER_LANGUAGE", "SCOPEWEAVER_ALLOW_WRITES", "SCOPEWEAVER_TOKEN"];
  const original = Object.fromEntries(names.map(name => [name, process.env[name]]));
  try {
    for (const language of SUPPORTED_LANGUAGES) {
      Object.assign(process.env, { SCOPEWEAVER_LANGUAGE: language, SCOPEWEAVER_ALLOW_WRITES: "true", SCOPEWEAVER_TOKEN: "fixture-token" });
      const registered = [];
      extension({ registerTool: tool => registered.push(tool) });
      assert.equal(registered.length, 8);
      for (const tool of registered) {
        assert.equal(tool.label, MESSAGES[language][`tools.${tool.name.slice(12)}.title`]);
        assert.equal(tool.description, MESSAGES[language][`tools.${tool.name.slice(12)}.description`]);
      }
      const task = registered.find(tool => tool.name === "scopeweaver_get_task");
      const local = createTools(clientFor(language)).find(tool => tool.name === task.name).schema.safeParse({}).error.issues[0].message;
      await assert.rejects(task.execute("literal-call-id", {}), error => error.message.includes(local));
    }
  } finally {
    for (const name of names) if (original[name] === undefined) delete process.env[name]; else process.env[name] = original[name];
  }
});

test("copied standalone adapter includes all message resources without checkout locale files", () => {
  const copied = mkdtempSync(join(tmpdir(), "scopeweaver-adapter-language-"));
  try {
    for (const path of ["src", "package.json"]) cpSync(join(adapterRoot, path), join(copied, path), { recursive: true });
    // Reuse the locked dependencies already installed for this test, without downloads.
    symlinkSync(join(adapterRoot, "node_modules"), join(copied, "node_modules"), "junction");
    const result = spawnSync(process.execPath, ["--input-type=module", "-e", `
      import assert from 'node:assert/strict';
      import { configFromEnv, ScopeWeaverClient } from './src/client.js';
      import { createTools } from './src/tools.js';
      import { SUPPORTED_LANGUAGES, MESSAGES } from './src/messages.js';
      for (const language of SUPPORTED_LANGUAGES) {
        const client = new ScopeWeaverClient(configFromEnv({SCOPEWEAVER_LANGUAGE:language}));
        const tools = createTools(client);
        assert.equal(tools[0].title, MESSAGES[language]['tools.health.title']);
        assert.equal(client.t('response.failed'), MESSAGES[language]['response.failed']);
        assert.equal(tools[2].schema.safeParse({}).success, false);
      }
    `], { cwd: copied, encoding: "utf8", timeout: 10000 });
    assert.equal(result.status, 0, result.error?.message || result.stderr);
  } finally { rmSync(copied, { recursive: true, force: true }); }
});
