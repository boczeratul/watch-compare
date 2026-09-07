import { getTranslations } from "next-intl/server";
import { ExternalLink } from "lucide-react";
import { Link } from "@/i18n/navigation";
import type { Listing } from "@/lib/api";
import { Price } from "./price";
import { SourceBadge } from "./source-badge";

/** Cross-marketplace comparison for one reference: cheapest first, current listing highlighted. */
export async function CompareTable({ current, others }: { current: Listing; others: Listing[] }) {
  const t = await getTranslations("listing");
  const tc = await getTranslations("condition");
  const rows = [...others, current].sort((a, b) => (a.priceUsd ?? Infinity) - (b.priceUsd ?? Infinity));
  const cheapest = rows[0];
  return (
    <section aria-labelledby="compare-title" className="rounded-lg border border-slate-200 bg-white">
      <h2 id="compare-title" className="border-b border-slate-200 px-4 py-3 text-base font-semibold text-slate-900">
        {t("compareTitle")}
      </h2>
      {others.length === 0 ? (
        <p className="px-4 py-6 text-sm text-slate-500">{t("compareEmpty")}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <tbody>
              {rows.map((l) => {
                const isCurrent = l.id === current.id;
                return (
                  <tr key={l.id} className={isCurrent ? "bg-emerald-50/60" : "hover:bg-slate-50"}>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <SourceBadge source={l.source} name={l.sourceName} />
                        {l.id === cheapest.id && <span className="rounded bg-emerald-600 px-1.5 py-0.5 text-[11px] font-semibold text-white">{t("cheapestBadge")}</span>}
                        {isCurrent && <span className="text-xs text-slate-500">{t("thisListing")}</span>}
                      </div>
                      <p className="mt-1 line-clamp-1 text-slate-700">{l.title}</p>
                    </td>
                    <td className="px-4 py-3 text-slate-600">{tc(l.condition)}</td>
                    <td className="px-4 py-3 text-slate-600">{[l.locationCity, l.locationCountry].filter(Boolean).join(", ")}</td>
                    <td className="px-4 py-3 text-right">
                      <Price usd={l.priceUsd} original={l.priceExclTax ?? l.price} originalCurrency={l.currency} />
                    </td>
                    <td className="px-4 py-3 text-right">
                      {isCurrent ? (
                        <a href={l.url} target="_blank" rel="noopener noreferrer nofollow" className="inline-flex items-center gap-1 text-slate-700 hover:text-slate-900">
                          <ExternalLink className="h-4 w-4" aria-hidden />
                        </a>
                      ) : (
                        <Link href={`/listing/${l.id}`} className="text-slate-700 underline hover:text-slate-900">
                          →
                        </Link>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
