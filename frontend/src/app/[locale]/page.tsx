import { getTranslations, setRequestLocale, getLocale } from "next-intl/server";
import { cookies } from "next/headers";
import { ArrowRight, Clock3, Layers, Coins } from "lucide-react";
import { Link } from "@/i18n/navigation";
import { api, safe, type SearchResult } from "@/lib/api";
import { SearchBar } from "@/components/search-bar";
import { ListingGrid } from "@/components/listing-grid";
import { CURRENCY_COOKIE, isCurrency } from "@/lib/settings";
import { defaultCurrencyForLocale, type Locale } from "@/i18n/routing";
import { convertFromUsd, formatMoney, roundDisplay } from "@/lib/currency";

const EMPTY: SearchResult = { items: [], total: 0, page: 1, perPage: 8, totalPages: 0, facets: { brands: [], sources: [], conditions: [], movements: [], countries: [], dialColors: [], years: [] } };
const DEAL_CAP_USD = 5000;

export default async function HomePage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations("home");
  const currentLocale = await getLocale();
  const jar = await cookies();
  const cookieCurrency = jar.get(CURRENCY_COOKIE)?.value;
  const currency = isCurrency(cookieCurrency) ? cookieCurrency : defaultCurrencyForLocale[locale as Locale];

  const [stats, brands, newest, deals, rates] = await Promise.all([
    safe(api.stats(), { activeListings: 0, brands: 0, lastCrawlAt: null, sources: [] }),
    safe(api.brands(), { items: [] }),
    safe(api.searchListings({ sort: "newest", per_page: "8" }), EMPTY),
    safe(api.searchListings({ sort: "price_asc", price_min: "1000", price_max: String(DEAL_CAP_USD), currency: "USD", per_page: "8", condition: "new,unworn,very_good,good" }), EMPTY),
    safe(api.rates(), { base: "USD" as const, supported: [], items: [] }),
  ]);
  const rateTable = Object.fromEntries(rates.items.map((r) => [r.quote, r.rate]));
  const capLocal = convertFromUsd(DEAL_CAP_USD, currency, rateTable);
  const capLabel = capLocal ? formatMoney(roundDisplay(capLocal, currency), currency, currentLocale) : `US$${DEAL_CAP_USD}`;
  const popular = brands.items.filter((b) => b.listingCount > 0).slice(0, 18);
  const activeSources = stats.sources.filter((s) => s.count > 0).length || stats.sources.length;

  return (
    <div>
      <section className="border-b border-slate-200 bg-gradient-to-b from-slate-50 to-white">
        <div className="mx-auto max-w-4xl px-4 py-14 text-center sm:px-6 sm:py-20">
          <h1 className="text-3xl font-bold tracking-tight text-slate-900 sm:text-5xl">{t("heroTitle")}</h1>
          <p className="mx-auto mt-4 max-w-2xl text-base text-slate-600 sm:text-lg">
            {t("heroSubtitle", { count: stats.activeListings.toLocaleString(currentLocale), sources: activeSources })}
          </p>
          <div className="mx-auto mt-8 max-w-2xl">
            <SearchBar size="lg" />
          </div>
          <dl className="mt-8 flex flex-wrap justify-center gap-x-10 gap-y-3 text-sm text-slate-500">
            <div><dt className="inline font-semibold text-slate-900">{stats.activeListings.toLocaleString(currentLocale)}</dt> <dd className="inline">{t("statListings")}</dd></div>
            <div><dt className="inline font-semibold text-slate-900">{stats.brands}</dt> <dd className="inline">{t("statBrands")}</dd></div>
            <div><dt className="inline font-semibold text-slate-900">{activeSources}</dt> <dd className="inline">{t("statSources")}</dd></div>
            {stats.lastCrawlAt && (
              <div className="text-slate-400">{t("lastUpdated", { date: new Intl.DateTimeFormat(currentLocale, { dateStyle: "medium" }).format(new Date(stats.lastCrawlAt)) })}</div>
            )}
          </dl>
        </div>
      </section>

      <div className="mx-auto max-w-7xl space-y-14 px-4 py-12 sm:px-6">
        {popular.length > 0 && (
          <section>
            <SectionHeader title={t("popularBrands")} href="/brands" linkLabel={t("allBrands")} />
            <ul className="mt-4 flex flex-wrap gap-2">
              {popular.map((b) => (
                <li key={b.slug}>
                  <Link href={{ pathname: "/search", query: { brand: b.slug } }} className="inline-flex items-center gap-2 rounded-full border border-slate-300 bg-white px-4 py-1.5 text-sm font-medium text-slate-800 hover:border-slate-900">
                    {b.name}
                    <span className="text-xs text-slate-400">{b.listingCount.toLocaleString(currentLocale)}</span>
                  </Link>
                </li>
              ))}
            </ul>
          </section>
        )}

        {newest.items.length > 0 && (
          <section>
            <SectionHeader title={t("newest")} href="/search" query={{ sort: "newest" }} linkLabel={t("viewAll")} />
            <div className="mt-4"><ListingGrid items={newest.items} /></div>
          </section>
        )}

        {deals.items.length > 0 && (
          <section>
            <SectionHeader title={t("bestDeals", { price: capLabel })} href="/search" query={{ sort: "price_asc", price_min: "1000", price_max: String(DEAL_CAP_USD) }} linkLabel={t("viewAll")} />
            <div className="mt-4"><ListingGrid items={deals.items} /></div>
          </section>
        )}

        <section className="rounded-xl border border-slate-200 bg-slate-50 p-6 sm:p-8">
          <h2 className="text-xl font-semibold text-slate-900">{t("howTitle")}</h2>
          <div className="mt-6 grid gap-6 sm:grid-cols-3">
            {[
              { icon: Clock3, title: t("how1Title"), body: t("how1Body") },
              { icon: Layers, title: t("how2Title"), body: t("how2Body") },
              { icon: Coins, title: t("how3Title"), body: t("how3Body") },
            ].map(({ icon: Icon, title, body }) => (
              <div key={title}>
                <Icon className="h-6 w-6 text-emerald-700" aria-hidden />
                <h3 className="mt-3 font-semibold text-slate-900">{title}</h3>
                <p className="mt-1 text-sm text-slate-600">{body}</p>
              </div>
            ))}
          </div>
        </section>
      </div>
    </div>
  );
}

function SectionHeader({ title, href, query, linkLabel }: { title: string; href: "/search" | "/brands"; query?: Record<string, string>; linkLabel: string }) {
  return (
    <div className="flex items-end justify-between">
      <h2 className="text-xl font-semibold text-slate-900">{title}</h2>
      <Link href={query ? { pathname: href, query } : href} className="inline-flex items-center gap-1 text-sm text-slate-600 hover:text-slate-900">
        {linkLabel} <ArrowRight className="h-4 w-4" aria-hidden />
      </Link>
    </div>
  );
}
