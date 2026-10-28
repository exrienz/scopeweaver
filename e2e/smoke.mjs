// Essential browser smoke test against a running ScopeWeaver binary and its PostgreSQL database.
// It initializes the password, so the database must be fresh and disposable (scopeweaver_test_*).
//
// Usage: SCOPEWEAVER_URL=http://127.0.0.1:8787 ARTEX_PG_DSN=postgres://.../scopeweaver_test_e2e node smoke.mjs
// Requires the psql client; it toggles connections to the test database to simulate an outage.
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import { chromium, request } from "playwright";
import { runChatPasteChecks } from "./chat-paste.mjs";
import { runOllamaProfileChecks } from "./ollama-profiles.mjs";
import { LOCALES, intlLocale, namespaceMessages } from "./locale-resources.mjs";

const base = process.env.SCOPEWEAVER_URL || "http://127.0.0.1:8787";
const dsn = new URL((process.env.ARTEX_PG_DSN || "").replace("@/", "@localhost/"));
const dbName = dsn.pathname.slice(1);
if (!dbName.startsWith("scopeweaver_test_")) throw new Error(`refusing to use non-test database "${dbName}"`);
const out = process.env.E2E_OUT || "e2e-results";
fs.mkdirSync(out, { recursive: true });

const PASSWORD = "e2e-pass-1234";
const interfaces = namespaceMessages("interface"), settings = namespaceMessages("settings");
const backendCatalogs = Object.fromEntries(LOCALES.filter(locale => !["en", "ko"].includes(locale)).map(locale => [locale,
  JSON.parse(fs.readFileSync(new URL(`../locale/${locale}.json`, import.meta.url), "utf8")),
]));
const backendText = (locale, key) => locale === "en" ? key : locale === "ko"
  ? ({ "Unauthorized": "인증이 필요합니다", "Data source is temporarily unavailable; please try again later": "데이터 소스를 일시적으로 사용할 수 없습니다. 잠시 후 다시 시도하세요" })[key]
  : backendCatalogs[locale][key];
const TEXT = Object.fromEntries(LOCALES.map(locale => [locale, {
  unavailable: settings[locale].setupUnavailableTitle,
  dataError: backendText(locale, "Data source is temporarily unavailable; please try again later"),
  usage: settings[locale].judgeUsageTitle,
  fallback: ["m1207", "m1208", "m1209", "m1210", "m1211"].map(key => interfaces[locale][key]).join(["en", "es"].includes(locale) ? " " : ""),
}]));

function sql(query, database = "postgres") {
  const admin = new URL(dsn);
  admin.pathname = `/${database}`;
  return execFileSync("psql", ["-tAc", query, admin.href], { encoding: "utf8" }).trim();
}
function setDatabaseAvailable(on) {
  sql(`ALTER DATABASE ${dbName} WITH ALLOW_CONNECTIONS ${on}`);
  if (!on) sql(`SELECT count(pg_terminate_backend(pid)) FROM pg_stat_activity WHERE datname = '${dbName}'`);
}
function check(ok, message) {
  if (!ok) throw new Error(message);
}

const failures = [];
const pageErrors = [];
async function step(name, fn) {
  try {
    await fn();
    console.log(`PASS ${name}`);
  } catch (err) {
    failures.push(name);
    console.log(`FAIL ${name}: ${err.stack || err.message}`);
  }
}

const browser = await chromium.launch();
async function openPage(locale, token) {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  await context.addInitScript(([l, t]) => {
    localStorage.setItem("scopeweaver.locale", l);
    if (t) localStorage.setItem("artex_token", t);
  }, [locale, token]);
  const page = await context.newPage();
  page.on("pageerror", (err) => pageErrors.push(`${locale}: ${err.message}`));
  return page;
}

const localeSelector = (p) => p.locator('select:has(option[value="zh-Hans"])');
async function switchLocale(p, locale) {
  await localeSelector(p).selectOption(locale);
  await p.waitForFunction((l) => document.documentElement.lang === l, locale);
  const preference = await p.evaluate(() => ({stored:localStorage.getItem("scopeweaver.locale"),cookie:document.cookie}));
  check(preference.stored === locale && preference.cookie.includes(`scopeweaver_locale=${locale}`), "preference stores disagree");
}
async function checkDraftSwitches(p, selector, draft) {
  await p.locator(selector).fill(draft);
  await p.locator(selector).evaluate((node) => {window.__i18nDraftNode=node;});
  for (const locale of LOCALES) {
    await switchLocale(p, locale);
    check(await p.locator(selector).inputValue() === draft, `${locale} lost draft`);
    check(await p.locator(selector).evaluate((node) => node === window.__i18nDraftNode), `${locale} remounted draft control`);
  }
  await switchLocale(p, "zh-Hans");
}

// 1. A database outage must not expose the first-time setup form.
for (const locale of LOCALES) {
  await step(`[${locale}] setup fails closed while the database is unavailable`, async () => {
    const page = await openPage(locale);
    setDatabaseAvailable(false);
    try {
      await page.goto(`${base}/setup`);
      await page.getByText(TEXT[locale].unavailable).waitFor();
      check(await page.getByText(TEXT[locale].dataError).isVisible(), "localized outage message missing");
      check((await page.locator('input[type="password"]').count()) === 0, "password form shown during outage");
      await page.screenshot({ path: `${out}/setup-unavailable-${locale}.png` });
    } finally {
      setDatabaseAvailable(true);
      await page.context().close();
    }
  });
}

// 2. First-time setup, then login with the same password.
const page = await openPage("en");
let token = "";
await step("[zh-Hans] setup password drafts survive EN/KO/Chinese switches", async () => {
  await page.goto(`${base}/setup`);
  await page.fill("#confirm", "confirmation 原文");
  await checkDraftSwitches(page, "#password", "unsaved-password 原文");
  check(await page.locator("#confirm").inputValue() === "confirmation 原文", "confirmation draft lost");
  await page.getByRole("button", {name:"设置密码并登录", exact:true}).waitFor();
  await page.screenshot({path:`${out}/i18n-setup-draft-zh-Hans.png`});
  await switchLocale(page,"en");
});

await step("[en] setup creates the password and login accepts it", async () => {
  await page.goto(`${base}/setup`);
  await page.fill("#password", PASSWORD);
  await page.fill("#confirm", PASSWORD);
  await page.locator('button[type="submit"]').click();
  await page.waitForURL(/\/function\/tasks/);
  const second = await page.request.post(`${base}/api/auth/init`, { data: { password: "another-pass-99" } });
  check(second.status() === 403, `second setup returned ${second.status()}`);

  await page.evaluate(() => localStorage.removeItem("artex_token"));
  await page.goto(`${base}/login`);
  await page.fill("#password", PASSWORD);
  await page.locator("#agree-terms").click();
  await page.locator('button[type="submit"]').click();
  await page.waitForURL(/\/function\/tasks/);
  token = await page.evaluate(() => localStorage.getItem("artex_token"));
  check(token, "login stored no token");
});

await step("[zh-Hans] login draft and persisted reload preserve preference", async () => {
  const p=await openPage("en");
  try {
    await p.goto(`${base}/login`);
    await checkDraftSwitches(p,"#password","unsaved-login 原文");
    await p.getByRole("button", {name:"登录",exact:true}).waitFor();
    // Remove the per-navigation initializer so reload reads the stores itself.
    const context=await browser.newContext({storageState:await p.context().storageState()});
    const restored=await context.newPage();
    restored.on("pageerror",err=>pageErrors.push(`restore: ${err.message}`));
    await restored.goto(`${base}/login`);
    await restored.waitForFunction(()=>document.documentElement.lang==="zh-Hans");
    check(await localeSelector(restored).inputValue()==="zh-Hans","stored Chinese did not restore");
    await restored.screenshot({path:`${out}/i18n-login-restored-zh-Hans.png`});
    await context.close();
  } finally {await p.context().close();}
});

await step("[zh-Hans] cookie-only and locale-storage privacy fallback restore Chinese", async () => {
  for (const privacy of [false,true]) {
    const context=await browser.newContext();
    await context.addCookies([{name:"scopeweaver_locale",value:"zh-Hans",url:base}]);
    if (privacy) await context.addInitScript(() => {
      const get=Storage.prototype.getItem, set=Storage.prototype.setItem;
      Storage.prototype.getItem=function(key) {if(key==="scopeweaver.locale")throw new Error("blocked locale storage");return get.call(this,key);};
      Storage.prototype.setItem=function(key,value) {if(key==="scopeweaver.locale")throw new Error("blocked locale storage");return set.call(this,key,value);};
    });
    const p=await context.newPage();
    p.on("pageerror",err=>pageErrors.push(`cookie/privacy: ${err.message}`));
    try {
      await p.goto(`${base}/login`);
      await p.waitForFunction(()=>document.documentElement.lang==="zh-Hans");
      check(await localeSelector(p).inputValue()==="zh-Hans","cookie preference failed");
      await p.getByRole("button",{name:"登录",exact:true}).waitFor();
    } finally {await context.close();}
  }
});

await step("[zh-Hans] task, settings and chat drafts survive language switches", async () => {
  const p=await openPage("en",token);
  try {
    await p.goto(`${base}/function/tasks`);
    await p.getByRole("button", {name:"New task",exact:true}).click();
    await p.fill("#name","custom task 原文");
    await p.fill("#goal","local-only goal {raw}");
    await checkDraftSwitches(p,"#description","unsaved task description 原文 <b>{raw}</b>");
    check(await p.locator("#name").inputValue()==="custom task 原文","task name lost");
    check(await p.locator("#goal").inputValue()==="local-only goal {raw}","task goal lost");
    await p.screenshot({path:`${out}/i18n-task-draft-zh-Hans.png`});
    await p.goto(`${base}/system/settings`);
    await p.locator("#global-proxy:not([disabled])").waitFor();
    await checkDraftSwitches(p,"#global-proxy","http://unsaved.example.invalid:8123");
    await p.goto(`${base}/chat`);
    await p.locator("textarea:not([disabled])").first().waitFor();
    await checkDraftSwitches(p,"textarea:not([disabled])","unsaved chat 原文 {raw}");
    await p.screenshot({path:`${out}/i18n-chat-draft-zh-Hans.png`});
  } finally {await p.context().close();}
});

for (const locale of LOCALES) {
  await step(`[${locale}] first API, SSE reconnect, draft and persisted preference`, async () => {
    const p = await openPage(locale, token);
    try {
      const firstApi = p.waitForRequest(r => new URL(r.url()).pathname.startsWith("/api/") && r.method() === "GET");
      const stream = p.waitForResponse(r => r.url().includes("/api/logs/stream") && new URL(r.url()).searchParams.get("lang") === locale);
      await p.goto(`${base}/system/logs`);
      check((await firstApi).headers()["accept-language"] === locale, "first API did not use stored locale");
      const response = await stream, url = new URL(response.url());
      check(url.origin === new URL(base).origin && url.searchParams.get("since") === "0", "SSE origin or cursor changed");
      check(!!url.searchParams.get("token"), "SSE auth missing");
      check(response.headers()["content-language"] === locale, "SSE Content-Language missing");
      check((response.headers()["content-type"] || "").includes("text/event-stream"), "not SSE");
      const filter = p.getByRole("main").getByRole("textbox");
      const raw = "raw-log-filter 原文 {raw}";
      await filter.fill(raw);
      const alternate = locale === "en" ? "ko" : "en";
      for (const next of [alternate, locale]) {
        const reconnect = p.waitForResponse(r => r.url().includes("/api/logs/stream") && new URL(r.url()).searchParams.get("lang") === next);
        await switchLocale(p, next); await reconnect;
        check(await filter.inputValue() === raw, "reconnect lost log filter draft");
      }
      const restoredContext = await browser.newContext({ storageState: await p.context().storageState() });
      try {
        const restored = await restoredContext.newPage();
        restored.on("pageerror", error => pageErrors.push(`${locale} restore: ${error.message}`));
        const restoredApi = restored.waitForRequest(r => new URL(r.url()).pathname.startsWith("/api/") && r.method() === "GET");
        await restored.goto(`${base}/system/logs`);
        check((await restoredApi).headers()["accept-language"] === locale, "restored first API locale changed");
        await restored.waitForFunction(l => document.documentElement.lang === l, locale);
        check(await localeSelector(restored).inputValue() === locale, "stored preference did not restore without initializer");
      } finally { await restoredContext.close(); }
      const anonymous = await request.newContext();
      try {
        const denied = await anonymous.get(`${base}/api/intercept/judge/usage?lang=${locale}`);
        check(denied.status() === 401 && denied.headers()["content-language"] === locale, "localized API error transport failed");
        check(JSON.stringify(await denied.json()).includes(backendText(locale, "Unauthorized")), "API error not translated");
      } finally { await anonymous.dispose(); }
      await p.screenshot({ path: `${out}/i18n-sse-${locale}.png` });
    } finally { await p.context().close(); }
  });
}
// 3. Live events use the UI origin, so a reverse proxy only needs one public port.
await step("[en] log stream SSE stays on the UI origin", async () => {
  const stream = page.waitForResponse((r) => r.url().includes("/api/logs/stream"));
  await page.goto(`${base}/system/logs`);
  const response = await stream;
  check(new URL(response.url()).origin === new URL(base).origin, `SSE went to ${new URL(response.url()).origin}`);
  check((response.headers()["content-type"] || "").includes("text/event-stream"), "response is not an event stream");
});

// 4. Approval-model usage is counted separately and stays visible after the judge is disabled.
sql(`INSERT INTO llm_usage (ts, worker, profile_name, input_tokens, output_tokens, status) VALUES
  (now(), 'judge', 'e2e-judge', 1200, 300, 'ok'),
  (now() - interval '3 days', 'judge', 'e2e-judge', 800, 200, 'ok'),
  (now(), 'planner', 'e2e-other', 99999, 99999, 'ok')`, dbName);

await step("[api] approval usage excludes other agents and fills 30 days", async () => {
  const response = await page.request.get(`${base}/api/intercept/judge/usage?days=30`, { headers: { Authorization: `Bearer ${token}` } });
  const usage = await response.json();
  check(usage.calls === 2 && usage.input_tokens === 2000 && usage.output_tokens === 500, `totals ${JSON.stringify(usage).slice(0, 120)}`);
  check(usage.daily.length === 30 && usage.daily.filter((d) => d.calls > 0).length === 2, "daily series is not 30 zero-filled days");
  // A separate context carries no session cookie.
  const anonymous = await request.newContext();
  const denied = await anonymous.get(`${base}/api/intercept/judge/usage`);
  check(denied.status() === 401, `unauthenticated request returned ${denied.status()}`);
  await anonymous.dispose();
});

for (const locale of LOCALES) {
  await step(`[${locale}] approval usage card shows past spend while the judge is disabled`, async () => {
    const p = await openPage(locale, token);
    await p.goto(`${base}/system/intercept`);
    await p.getByRole("tab").nth(1).click();
    await p.getByText(TEXT[locale].usage).waitFor();
    await p.getByText(TEXT[locale].fallback, { exact: true }).waitFor();
    const inputTokens = new Intl.NumberFormat(intlLocale(locale), { notation: "compact", maximumFractionDigits: 1 }).format(2000);
    const inputStat = p.getByText(settings[locale].judgeUsageInput, { exact: true }).locator("..");
    await inputStat.getByText(inputTokens, { exact: true }).waitFor();
    const bars = p.locator("div.flex.h-16 > div");
    check((await bars.count()) === 30, `${await bars.count()} daily bars`);
    check((await p.locator("div.flex.h-16 > div.bg-violet-500\\/60").count()) === 2, "expected 2 days with usage");
    await p.screenshot({ path: `${out}/approval-usage-${locale}.png`, fullPage: true });
    await p.context().close();
  });
}

// 5. Native Ollama templates, keyless save/activate and the opt-in local-only policy (fixture endpoint).
await runOllamaProfileChecks({ base, token, step, check, openPage, out });

await runChatPasteChecks({ base, token, step, check, openPage });

await step("no uncaught page errors", async () => check(pageErrors.length === 0, pageErrors.join(" | ")));

await browser.close();
console.log(failures.length ? `E2E failed: ${failures.length} step(s)` : "E2E passed");
process.exit(failures.length ? 1 : 0);
