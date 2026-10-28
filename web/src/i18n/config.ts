// Locale constants and pure helpers shared by the UI runtime, the API client
// and the node tests. Keep this module free of DOM access and path aliases so
// `node --test` can import it directly.

import definition from "../../../locale/definitions.json" with { type: "json" };

// The union provides compile-time catalog coverage; the shared declaration
// controls runtime support, labels, Intl mappings and aliases.
export type Locale = "en" | "ko" | "zh-Hans" | "zh-Hant" | "ja" | "es";
export const LOCALE_DEFINITIONS = definition.locales;
export const LOCALES = definition.locales.map(({ id }) => id as Locale);
export const DEFAULT_LOCALE = definition.default as Locale;

// Persistence contract shared with the backend (see status notes):
// localStorage for the UI, a cookie so the server can negotiate too.
export const LOCALE_STORAGE_KEY = "scopeweaver.locale";
export const LOCALE_COOKIE = "scopeweaver_locale";
export const LOCALE_COOKIE_MAX_AGE = 365 * 24 * 60 * 60;

// Query parameter used where headers cannot be set (EventSource, downloads).
export const LOCALE_QUERY_PARAM = "lang";

// Native names are shown in the language selector regardless of UI locale.
export const LOCALE_NATIVE_NAMES = Object.fromEntries(
  LOCALE_DEFINITIONS.map(({ id, nativeName }) => [id, nativeName]),
) as Record<Locale, string>;

const INTL_LOCALES = Object.fromEntries(
  LOCALE_DEFINITIONS.map(({ id, intlLocale }) => [id, intlLocale]),
) as Record<Locale, string>;

export function isLocale(value: unknown): value is Locale {
  return typeof value === "string" && (LOCALES as readonly string[]).includes(value);
}

// normalizeLocale accepts stored values and BCP-47 tags ("ko-KR", "en_US")
// and maps them onto a supported locale; anything else yields null.
export function normalizeLocale(value: unknown): Locale | null {
  if (typeof value !== "string") return null;
  const tag = value.trim().toLowerCase().replaceAll("_", "-");
  const [base, ...parts] = tag.split("-");
  // Preserve the existing permissive EN/KO regional-tag behavior.
  if (base === "en" || base === "ko") return isLocale(base) ? base : null;
  if ([base, ...parts].some((part) => !part || /[ ;]/.test(part))) return null;
  const declared = (candidate: string): Locale | null => {
    const found = LOCALE_DEFINITIONS.find(({ id, aliases }) =>
      [id, ...aliases].some((alias) => alias.toLowerCase() === candidate));
    return found && isLocale(found.id) ? found.id : null;
  };
  if (base === "zh") {
    const scripts = [...new Set(parts.filter((part) => part.length === 4))];
    if (scripts.length > 1) return null;
    // Script-specific content wins a conflicting region, only when declared.
    if (scripts.length) return declared(`${base}-${scripts[0]}`);
    // Bare zh and regional aliases are explicit policy in the declaration.
    return declared(tag);
  }
  const exact = declared(tag);
  if (exact) return exact;
  const prefixes = LOCALE_DEFINITIONS.flatMap(({ id, aliases }) => [id, ...aliases])
    .sort((a, b) => b.length - a.length);
  for (const prefix of prefixes) {
    if (tag.startsWith(`${prefix.toLowerCase()}-`)) return declared(prefix.toLowerCase());
  }
  return null;
}

export function intlLocale(locale: Locale): string {
  return INTL_LOCALES[locale];
}

export function localeCookie(locale: Locale): string {
  return `${LOCALE_COOKIE}=${locale}; Path=/; Max-Age=${LOCALE_COOKIE_MAX_AGE}; SameSite=Lax`;
}

export function readCookieLocale(cookie: string): Locale | null {
  for (const part of cookie.split(";")) {
    const [name, ...rest] = part.trim().split("=");
    if (name === LOCALE_COOKIE) {
      try { return normalizeLocale(decodeURIComponent(rest.join("="))); }
      catch { return null; }
    }
  }
  return null;
}

// withLocaleQuery appends lang=<locale> to a URL (absolute or path-only),
// keeping any hash fragment and leaving an explicit lang parameter untouched.
export function withLocaleQuery(url: string, locale: Locale): string {
  const hashIndex = url.indexOf("#");
  const base = hashIndex >= 0 ? url.slice(0, hashIndex) : url;
  const hash = hashIndex >= 0 ? url.slice(hashIndex) : "";
  const queryIndex = base.indexOf("?");
  if (queryIndex >= 0) {
    const params = new URLSearchParams(base.slice(queryIndex + 1));
    if (params.has(LOCALE_QUERY_PARAM)) return url;
  }
  const sep = queryIndex >= 0 ? (base.endsWith("?") || base.endsWith("&") ? "" : "&") : "?";
  return `${base}${sep}${LOCALE_QUERY_PARAM}=${locale}${hash}`;
}

// withLocaleHeaders returns a header record carrying Accept-Language. Headers
// supplied by the caller (auth, content type, or an explicit Accept-Language)
// always win.
export function withLocaleHeaders(locale: Locale, headers?: HeadersInit): Record<string, string> {
  const out: Record<string, string> = { "Accept-Language": locale };
  if (!headers) return out;
  const entries: [string, string][] =
    typeof Headers !== "undefined" && headers instanceof Headers
      ? Array.from(headers.entries())
      : Array.isArray(headers)
        ? (headers as [string, string][])
        : Object.entries(headers as Record<string, string>);
  for (const [key, value] of entries) {
    if (key.toLowerCase() === "accept-language") delete out["Accept-Language"];
    out[key] = value;
  }
  return out;
}
