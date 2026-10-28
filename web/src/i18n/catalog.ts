// Assemble every shipped locale from English-authoritative namespaces.
// Keys are "<namespace>.<key>". Every namespace module types its Korean table
// as Record<keyof typeof en, string>, so a missing or extra Korean entry is a
// compile error; i18n.test.mjs additionally checks placeholder/tag parity.

import { LOCALES, type Locale } from "./config.ts";
import * as app from "./messages/app.ts";
import * as common from "./messages/common.ts";
import * as english from "./messages/english.ts";
import * as es from "./messages/es.ts";
import * as interfaceMessages from "./messages/interface.ts";
import * as ja from "./messages/ja.ts";
import * as providers from "./messages/providers.ts";
import * as settings from "./messages/settings.ts";
import * as zhHans from "./messages/zh-hans.ts";
import * as zhHant from "./messages/zh-hant.ts";

const namespaces = {
  providers,
  settings,
  english,
  interface: interfaceMessages,
  app,
  common,
};

type Namespaces = typeof namespaces;
type NamespaceName = keyof Namespaces & string;

export type MessageKey = {
  [N in NamespaceName]: `${N}.${keyof Namespaces[N]["en"] & string}`;
}[NamespaceName];

export type Catalog = Record<MessageKey, string>;

type NamespaceCatalogs = { [N in NamespaceName]: Record<keyof Namespaces[N]["en"], string> };

function namespaceTables(locale: "en" | "ko"): NamespaceCatalogs {
  return Object.fromEntries(Object.entries(namespaces).map(([ns, mod]) => [ns, mod[locale]])) as NamespaceCatalogs;
}

// Each new catalog is registered here; runtime support comes from definitions.
const resources = {
  en: namespaceTables("en"),
  ko: namespaceTables("ko"),
  "zh-Hans": zhHans,
  "zh-Hant": zhHant,
  ja,
  es,
} satisfies Record<Locale, NamespaceCatalogs>;

export const catalogLocales = Object.keys(resources);

function flatten(locale: Locale): Catalog {
  const out: Record<string, string> = {};
  for (const [ns, table] of Object.entries(resources[locale])) {
    for (const [key, value] of Object.entries(table)) out[`${ns}.${key}`] = value;
  }
  return out as Catalog;
}

export const catalogs = Object.fromEntries(LOCALES.map((locale) => [locale, flatten(locale)])) as Record<
  Locale,
  Catalog
>;

export const namespaceNames = Object.keys(namespaces);
