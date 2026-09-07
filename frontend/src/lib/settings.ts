export const CURRENCY_COOKIE = "wc_currency";
export const SUPPORTED_CURRENCIES = ["USD", "EUR", "GBP", "CHF", "JPY", "TWD", "HKD", "SGD", "CNY", "KRW", "AUD", "CAD"] as const;
export type Currency = (typeof SUPPORTED_CURRENCIES)[number];

export function isCurrency(v: string | undefined | null): v is Currency {
  return !!v && (SUPPORTED_CURRENCIES as readonly string[]).includes(v);
}
