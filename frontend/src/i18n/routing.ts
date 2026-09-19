import { defineRouting } from "next-intl/routing";

export const locales = ["en", "zh-TW", "zh-CN", "ja", "de"] as const;
export type Locale = (typeof locales)[number];

export const routing = defineRouting({
  locales,
  defaultLocale: "en",
  localePrefix: "as-needed",
});

export const localeNames: Record<Locale, string> = {
  en: "English",
  "zh-TW": "繁體中文",
  "zh-CN": "简体中文",
  ja: "日本語",
  de: "Deutsch",
};

/** Default display currency per locale, used when the visitor has not chosen one. */
export const defaultCurrencyForLocale: Record<Locale, string> = {
  en: "USD",
  "zh-TW": "TWD",
  "zh-CN": "CNY",
  ja: "JPY",
  de: "EUR",
};
