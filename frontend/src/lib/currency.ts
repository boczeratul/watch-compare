/** Pure helpers shared by server and client components. */
export type RateTable = Record<string, number>; // 1 USD = rate[quote]

export function convertFromUsd(usd: number, currency: string, rates: RateTable): number | null {
  if (currency === "USD") return usd;
  const r = rates[currency];
  return r ? usd * r : null;
}

const ZERO_DECIMAL = new Set(["JPY", "KRW", "TWD"]);

export function formatMoney(amount: number, currency: string, locale: string): string {
  const zero = ZERO_DECIMAL.has(currency);
  try {
    return new Intl.NumberFormat(locale, {
      style: "currency",
      currency,
      // "symbol" (not "narrowSymbol") so dollar-family currencies stay distinguishable: NT$, HK$, A$…
      currencyDisplay: "symbol",
      maximumFractionDigits: zero ? 0 : amount >= 1000 ? 0 : 2,
      minimumFractionDigits: 0,
    }).format(amount);
  } catch {
    return `${currency} ${Math.round(amount).toLocaleString(locale)}`;
  }
}

/** Round a converted price so it reads like a price tag rather than an FX quote. */
export function roundDisplay(amount: number, currency: string): number {
  if (ZERO_DECIMAL.has(currency)) return Math.round(amount / 100) * 100;
  return amount >= 10000 ? Math.round(amount / 10) * 10 : Math.round(amount);
}
