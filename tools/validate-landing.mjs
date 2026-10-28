#!/usr/bin/env node
/* Integrity check for the ScopeWeaver landing page.
   Lives outside landing/ so it is never published. Verifies that every
   referenced asset exists, that relative paths resolve under /scopeweaver/
   and each locale subdirectory, that six-language structural parity holds, and that no
   planning/seed/contract metadata leaked into the shipped folder. */

import { readFileSync, existsSync, readdirSync, statSync } from "node:fs";
import { resolve, dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const landing = join(root, "landing");
let errors = 0;
let checks = 0;

const fail = (m) => {
  errors++;
  console.error("  ✗ " + m);
};
const ok = (m) => {
  checks++;
  console.log("  ✓ " + m);
};

/* --- 1. Referenced assets resolve relative to each HTML file --- */
const locales = JSON.parse(readFileSync(join(root, "locale", "definitions.json"), "utf8")).locales;
const pages = locales.map(({id: lang}) => {
  const base = lang === "en" ? landing : join(landing, lang);
  return { file: join(base, "index.html"), base, lang };
});

const attrRe = /(?:src|href)="([^"#?][^"]*)"|srcset="([^"]+)"/g;

for (const { file, base, lang } of pages) {
  if (!existsSync(file)) {
    fail(`missing page: ${file}`);
    continue;
  }
  const html = readFileSync(file, "utf8");

  // lang attribute
  if (!new RegExp(`<html lang="${lang}"`).test(html))
    fail(`${lang}: <html lang="${lang}"> missing`);
  else ok(`${lang}: html lang is "${lang}"`);

  let m;
  const refs = new Set();
  while ((m = attrRe.exec(html))) {
    if (m[1]) refs.add(m[1]);
    if (m[2]) m[2].split(",").forEach((s) => refs.add(s.trim().split(/\s+/)[0]));
  }
  for (const ref of refs) {
    if (/^(https?:|mailto:|data:|\/\/)/.test(ref)) continue; // external/absolute
    const clean = ref.replace(/[#?].*$/, "");
    if (clean === "" || clean === "./" || clean === "../") continue;
    const target = resolve(base, clean);
    // must stay inside landing/
    if (!target.startsWith(landing)) {
      fail(`${lang}: reference escapes landing/: ${ref}`);
      continue;
    }
    if (!existsSync(target)) fail(`${lang}: missing asset -> ${ref}`);
  }
  ok(`${lang}: ${refs.size} references scanned, all local assets present`);

  // no absolute-root asset paths (would break under /scopeweaver/ base)
  if (/(?:src|href)="\/(?!\/)/.test(html))
    fail(`${lang}: absolute root path found (breaks project-pages base)`);
  else ok(`${lang}: no absolute root asset paths`);
}

/* --- 2. Every shipped language has the English structure and cross-links --- */
const en = readFileSync(join(landing, "index.html"), "utf8");
const ids = html => [...html.matchAll(/\bid="([^"]+)"/g)].map(match => match[1]).sort();
const countRole = (html, role) => (html.match(new RegExp(`role="${role}"`, "g")) || []).length;
for (const { file, lang } of pages) {
  if (!existsSync(file)) continue; // Missing pages were already reported above.
  const html = readFileSync(file, "utf8");
  if (JSON.stringify(ids(html)) !== JSON.stringify(ids(en))) fail(`${lang}: structural id parity mismatch`);
  else ok(`${lang}: English id parity (${ids(en).length} ids)`);
  for (const role of ["tab", "tabpanel", "tablist"]) {
    if (countRole(html, role) !== countRole(en, role)) fail(`${lang}: ${role} count differs from English`);
    else ok(`${lang}: ${role} count matches (${countRole(en, role)})`);
  }
  const copies = value => (value.match(/class="copy-btn"/g) || []).length;
  if (copies(html) !== copies(en)) fail(`${lang}: copy-btn count differs from English`);
  else ok(`${lang}: copy-btn count matches (${copies(en)})`);
  for (const { id: other } of locales) {
    if (other === lang) continue;
    const href = lang === "en" ? `${other}/` : other === "en" ? "../" : `../${other}/`;
    if (!html.includes(`href="${href}"`) && !html.includes(`href="${href}index.html"`)) fail(`${lang}: missing language link to ${other}`);
    else ok(`${lang}: links to ${other}`);
    const canonical = `https://cskwork.github.io/scopeweaver/${other === "en" ? "" : `${other}/`}`;
    if (!html.includes(`hreflang="${other}" href="${canonical}"`)) fail(`${lang}: missing alternate metadata for ${other}`);
  }
  const figures = (html.match(/<figure/g) || []).length;
  const captions = (html.match(/figcaption class="demo"/g) || []).length;
  if (captions < figures) fail(`${lang}: ${figures} figures but only ${captions} demo captions`);
  else ok(`${lang}: ${captions} demo captions for ${figures} figures`);
}

/* --- 3. Copy buttons preserve executable source commands, including casing --- */
function copiedCodeChanges(source, translated) {
  const codeById = html => new Map([...html.matchAll(/<code\b[^>]*\bid="([^"]+)"[^>]*>([\s\S]*?)<\/code>/g)].map(([, id, body]) => [id, body.replace(/<[^>]*>/g, "")]));
  const expected = codeById(source);
  const actual = codeById(translated);
  return [...source.matchAll(/data-copy-target="#([^"]+)"/g)].map(([, id]) => id).filter(id => !expected.has(id) || actual.get(id) !== expected.get(id));
}
const copiedFixture = '<button data-copy-target="#inst-docker"></button><code id="inst-docker">docker compose up -d --build</code>';
if (copiedCodeChanges(copiedFixture, copiedFixture.replace("docker compose", "Docker compose")).join() !== "inst-docker") fail("copied-command guard accepted altered Docker casing fixture");
else ok("copied-command guard rejects altered Docker casing fixture");
for (const { file, lang } of pages) {
  if (!existsSync(file)) continue;
  const changed = copiedCodeChanges(en, readFileSync(file, "utf8"));
  if (changed.length) fail(`${lang}: copied command text differs from English: ${changed.join(", ")}`);
  else ok(`${lang}: copied commands preserve English source literals and casing`);
}

/* --- 4. No planning/seed/contract metadata shipped in landing/ --- */
const banned = [/THESIS:/, /OWN-WORLD:/, /surface seed/i, /Direction contract/i, /impeccable:product-schema/];
const walk = (dir) => {
  for (const entry of readdirSync(dir)) {
    const p = join(dir, entry);
    const st = statSync(p);
    if (st.isDirectory()) walk(p);
    else if (/\.(html|css|js|txt|xml|json|md)$/.test(entry)) {
      const txt = readFileSync(p, "utf8");
      for (const re of banned)
        if (re.test(txt)) fail(`planning metadata leaked into ${p} (${re})`);
    }
  }
};
walk(landing);
ok("no planning/seed/contract metadata found in landing/");

/* --- 5. .nojekyll present --- */
if (!existsSync(join(landing, ".nojekyll"))) fail(".nojekyll missing");
else ok(".nojekyll present");

console.log(`\n${checks} checks passed, ${errors} error(s).`);
process.exit(errors ? 1 : 0);
