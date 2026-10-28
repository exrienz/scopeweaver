import { LOCALES } from "../i18n/config.ts";
import { translateIn } from "../i18n/runtime.ts";
import assert from "node:assert/strict";
import fs from "node:fs";
import { registerHooks } from "node:module";
import test from "node:test";

// Resolve the same aliases as Next.js while loading the real TypeScript parser.
const hooks = registerHooks({
  resolve(specifier, context, nextResolve) {
    const url = specifier.startsWith("@/")
      ? new URL(`../${specifier.slice(2)}`, import.meta.url)
      : specifier.startsWith(".")
        ? new URL(specifier, context.parentURL)
        : null;
    if (url && !/\.[a-z]+$/i.test(url.pathname)) {
      for (const suffix of [".ts", "/index.ts"]) {
        const candidate = new URL(url.href + suffix);
        if (fs.existsSync(candidate)) return nextResolve(candidate.href, context);
      }
    }
    return nextResolve(specifier, context);
  },
});
const {
  classifyCompanyScopeLine,
  companyScopeRuleError,
  companyScopeRuleText,
  isCompanyScopeKind,
  normalizeCompanyScopeValue,
  parseCompanyScopeText,
} = await import("./company-scope.ts");
hooks.deregister();

test("host is a company scope kind", () => {
  assert.equal(isCompanyScopeKind("host"), true);
});

test("typed host accepts a hostname or URL", () => {
  assert.equal(companyScopeRuleError({ kind: "host", value: "cms.example.no" }), "");
  assert.equal(companyScopeRuleError({ kind: "host", value: "https://cms.example.no:8443/x" }), "");
});

test("typed host refuses IPs, bare TLDs and empty values", () => {
  assert.notEqual(companyScopeRuleError({ kind: "host", value: "8.8.8.8" }), "");
  assert.notEqual(companyScopeRuleError({ kind: "host", value: "no" }), "");
  assert.notEqual(companyScopeRuleError({ kind: "host", value: "" }), "");
});

test("untyped lines keep classifying dotted names as domains", () => {
  assert.deepEqual(classifyCompanyScopeLine("cms.example.no", 1).rule, { kind: "domain", value: "cms.example.no" });
  assert.deepEqual(classifyCompanyScopeLine("8.8.8.8", 1).rule, { kind: "ip", value: "8.8.8.8" });
});

test("host values normalize like domains", () => {
  assert.equal(normalizeCompanyScopeValue({ kind: "host", value: "https://CMS.example.no:8443/x" }), "cms.example.no");
});

test("host labels and hints translate in every shipped locale", () => {
  for (const locale of LOCALES) {
    for (const key of ["interface.m2883", "interface.m2884", "interface.m2885"]) {
      const text = translateIn(locale, key);
      assert.notEqual(text, key, `${locale} ${key}`);
      assert.notEqual(text, "", `${locale} ${key}`);
    }
  }
});

test("a host: line is a host rule and round-trips through the editor text", () => {
  assert.deepEqual(classifyCompanyScopeLine("host:cms.example.no", 1).rule, { kind: "host", value: "cms.example.no" });
  assert.deepEqual(classifyCompanyScopeLine("HOST: cms.example.no", 1).rule, { kind: "host", value: "cms.example.no" });
  assert.notEqual(classifyCompanyScopeLine("host:8.8.8.8", 1).error ?? "", "");
  const text = companyScopeRuleText({ kind: "host", value: "cms.example.no" });
  assert.equal(text, "host:cms.example.no");
  assert.deepEqual(parseCompanyScopeText(text).rules, [{ kind: "host", value: "cms.example.no" }]);
  assert.equal(companyScopeRuleText({ kind: "domain", value: "example.no" }), "example.no");
});
