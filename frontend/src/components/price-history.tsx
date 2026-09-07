import { getTranslations, getLocale } from "next-intl/server";
import type { PricePoint } from "@/lib/api";
import { Price } from "./price";

export async function PriceHistory({ points }: { points: PricePoint[] }) {
  const t = await getTranslations("listing");
  const locale = await getLocale();
  if (points.length < 2) return null;
  const asc = [...points].reverse();
  const usd = asc.map((p) => p.priceUsd ?? 0);
  const min = Math.min(...usd);
  const max = Math.max(...usd);
  const w = 320;
  const h = 64;
  const pts = usd.map((v, i) => `${(i / (usd.length - 1)) * w},${h - ((v - min) / (max - min || 1)) * (h - 8) - 4}`).join(" ");
  return (
    <section className="rounded-lg border border-slate-200 bg-white p-4">
      <h2 className="mb-3 text-base font-semibold text-slate-900">{t("priceHistory")}</h2>
      <svg viewBox={`0 0 ${w} ${h}`} className="h-16 w-full text-emerald-700" role="img" aria-label={t("priceHistory")}>
        <polyline fill="none" stroke="currentColor" strokeWidth="2" points={pts} />
      </svg>
      <ul className="mt-3 max-h-40 space-y-1 overflow-auto text-sm">
        {points.map((p) => (
          <li key={p.observedAt} className="flex justify-between text-slate-600">
            <span>{new Intl.DateTimeFormat(locale, { dateStyle: "medium" }).format(new Date(p.observedAt))}</span>
            <Price usd={p.priceUsd} original={p.price} originalCurrency={p.currency} size="sm" showOriginal={false} />
          </li>
        ))}
      </ul>
    </section>
  );
}
