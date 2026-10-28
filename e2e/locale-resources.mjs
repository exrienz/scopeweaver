import { catalogs } from "../web/src/i18n/catalog.ts";
export { LOCALES, intlLocale } from "../web/src/i18n/config.ts";

// Read the same complete namespace resources as the UI, for every shipped locale.
export function namespaceMessages(namespace) {
  return Object.fromEntries(Object.entries(catalogs).map(([locale, catalog]) => [locale,
    Object.fromEntries(Object.entries(catalog)
      .filter(([key]) => key.startsWith(`${namespace}.`))
      .map(([key, value]) => [key.slice(namespace.length + 1), value])),
  ]));
}
