// Static landing regression: no application server, database, or external service required.
import assert from "node:assert/strict";
import fs from "node:fs";
import { createServer } from "node:http";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const landing = path.join(root, "landing");
const locales = JSON.parse(fs.readFileSync(path.join(root, "locale/definitions.json"))).locales;
const out = path.resolve(process.env.E2E_OUT || path.join(root, "e2e/e2e-results"), "landing-locales");
fs.mkdirSync(out, { recursive: true });
const copied = { en: "Copied", ko: "복사됨", "zh-Hans": "已复制", "zh-Hant": "已複製", ja: "コピーしました", es: "Copiado" };
const contentTypes = {
  ".html": "text/html; charset=utf-8", ".css": "text/css", ".js": "text/javascript",
  ".svg": "image/svg+xml", ".png": "image/png", ".jpg": "image/jpeg", ".woff2": "font/woff2",
};
const mount = "/scopeweaver/";
const server = createServer((req, res) => {
  try {
    const pathname = decodeURIComponent(new URL(req.url, "http://localhost").pathname);
    assert.ok(pathname.startsWith(mount));
    let file = path.resolve(landing, pathname.slice(mount.length));
    assert.ok(file === landing || file.startsWith(landing + path.sep));
    if (fs.statSync(file).isDirectory()) file = path.join(file, "index.html");
    const body = fs.readFileSync(file);
    res.writeHead(200, { "Content-Type": contentTypes[path.extname(file)] || "application/octet-stream" });
    res.end(body);
  } catch {
    res.writeHead(404);
    res.end("Not found");
  }
});
const results = [];
let browser;
let englishCommand;
try {
  await new Promise((resolve, reject) => { server.once("error", reject); server.listen(0, "127.0.0.1", resolve); });
  const base = `http://127.0.0.1:${server.address().port}${mount}`;
  browser = await chromium.launch();
  for (const [index, { id }] of locales.entries()) for (const width of [1440, 390]) {
    const context = await browser.newContext({ viewport: { width, height: 900 } });
    await context.addInitScript(() => Object.defineProperty(navigator, "clipboard", {
      configurable: true, value: { writeText: async text => { window.__copiedText = text; } },
    }));
    const page = await context.newPage();
    const errors = [];
    page.on("pageerror", error => errors.push(error.message));
    page.on("response", response => { if (response.status() >= 400) errors.push(`${response.status()} ${response.url()}`); });
    const url = base + (id === "en" ? "" : `${id}/`);
    try {
      await page.goto(url, { waitUntil: "networkidle" });
      assert.equal(await page.locator("html").getAttribute("lang"), id);
      await checkOverflow();
      const broken = await page.locator("img:visible").evaluateAll(images => images.filter(image => {
        const rect = image.getBoundingClientRect();
        return rect.top < innerHeight && rect.bottom > 0 && (!image.complete || image.naturalWidth === 0);
      }).map(image => image.src));
      assert.deepEqual(broken, []);
      await page.screenshot({ path: path.join(out, `${id}-${width}-hero.png`) });

      const menu = page.locator(".language-menu"), summary = menu.locator("summary");
      await summary.click();
      assert.notEqual(await menu.getAttribute("open"), null);
      const names = await menu.locator(".language-options > *").allTextContents();
      assert.deepEqual(names.map(text => text.trim()).sort(), locales.map(locale => locale.nativeName).sort());
      await checkOverflow();
      const geometry = await menu.evaluate(element => {
        const panel = element.querySelector(".language-options").getBoundingClientRect();
        const trigger = element.querySelector("summary").getBoundingClientRect();
        const options = [...element.querySelectorAll(".language-options > *")].map(option => {
          const rect = option.getBoundingClientRect();
          const hit = document.elementFromPoint(rect.x + rect.width / 2, rect.y + rect.height / 2);
          return { text: option.textContent.trim(), width: rect.width, height: rect.height, hit: hit === option || option.contains(hit) };
        });
        const style = getComputedStyle(element);
        return { overflow: style.overflow, display: style.display, left: panel.left, right: panel.right, top: panel.top,
          bottom: panel.bottom, viewportWidth: innerWidth, viewportHeight: innerHeight,
          triggerWidth: trigger.width, triggerHeight: trigger.height, options };
      });
      assert.ok(geometry.left >= 0 && geometry.right <= width, "language panel must fit the viewport horizontally");
      assert.ok(geometry.top >= 0 && geometry.bottom <= geometry.viewportHeight, "language panel must fit the viewport vertically");
      assert.ok(geometry.triggerWidth >= 44 && geometry.triggerHeight >= 44, "language trigger target must be at least 44px");
      assert.ok(geometry.options.every(option => option.width >= 44 && option.height >= 44), "language option targets must be at least 44px");
      assert.ok(geometry.options.every(option => option.hit), `language options clipped or covered: ${JSON.stringify(geometry)}`);
      await page.screenshot({ path: path.join(out, `${id}-${width}-menu.png`) });
      const next = locales[(index + 1) % locales.length].id;
      await menu.locator(`a[hreflang="${next}"]`).click();
      await checkNavigation(next);

      await page.goto(url, { waitUntil: "networkidle" });
      await summary.focus();
      await page.keyboard.press("Enter");
      assert.notEqual(await menu.getAttribute("open"), null, "Enter must open the native disclosure");
      assert.equal(await summary.evaluate(element => element.matches(":focus-visible") && getComputedStyle(element).outlineStyle !== "none"), true);
      const first = menu.locator(".language-options a").first();
      const keyboardTarget = await first.getAttribute("hreflang");
      await page.keyboard.press("Tab");
      assert.equal(await first.evaluate(element => document.activeElement === element), true, "Tab must reach the first language link");
      await page.keyboard.press("Enter");
      await checkNavigation(keyboardTarget);

      await page.goto(url, { waitUntil: "networkidle" });
      const button = page.locator('[data-copy-target="#inst-docker"]');
      await button.scrollIntoViewIfNeeded();
      await button.click();
      await page.waitForFunction(() => typeof window.__copiedText === "string");
      const command = await page.evaluate(() => window.__copiedText);
      if (id === "en" && width === 1440) englishCommand = command;
      assert.equal(command, englishCommand);
      assert.ok(command.includes("docker compose up -d --build"));
      assert.equal(await button.getAttribute("aria-label"), copied[id]);
      assert.deepEqual(errors, []);
      results.push({ locale: id, width, status: "PASS", menu: geometry });
      console.log(`PASS landing ${id} ${width}: native menu click/keyboard, unclipped targets, copy, assets`);

      async function checkOverflow() {
        assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1), "horizontal overflow");
      }
      async function checkNavigation(target) {
        await page.waitForFunction(locale => document.documentElement.lang === locale, target);
        // Static editions use both directory links and explicit index.html links.
        assert.equal(page.url().replace(/\/index\.html$/, "/"), base + (target === "en" ? "" : `${target}/`));
      }
    } catch (error) {
      results.push({ locale: id, width, status: "FAIL", error: error.stack });
      console.error(`FAIL landing ${id} ${width}: ${error.stack}`);
    } finally {
      await context.close();
    }
  }
} finally {
  try { await browser?.close(); }
  finally {
    await new Promise(resolve => server.close(resolve));
    fs.writeFileSync(path.join(out, "results.json"), JSON.stringify(results, null, 2) + "\n");
  }
}
if (results.some(result => result.status === "FAIL")) process.exitCode = 1;
