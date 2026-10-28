// date-fns objects cannot be serialized in the shared locale declaration.
import { format } from "date-fns";
import { enUS, ko, zhCN, zhTW, ja, es } from "date-fns/locale";
import type { Locale as CalendarLocale } from "react-day-picker";
import type { Locale } from "./config.ts";

export const DATE_FNS_LOCALES = { en: enUS, ko, "zh-Hans": zhCN, "zh-Hant": zhTW, ja, es } satisfies Record<Locale, CalendarLocale>;

export function calendarLocale(locale: Locale, supplied?: Partial<CalendarLocale>): Partial<CalendarLocale> {
  return supplied ?? DATE_FNS_LOCALES[locale];
}

export function formatRangeDate(locale: Locale, date: Date): string {
  return format(date, "d MMM yyyy", { locale: DATE_FNS_LOCALES[locale] });
}
