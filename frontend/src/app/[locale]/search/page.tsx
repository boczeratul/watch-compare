import type { Metadata } from "next";
import { Suspense } from "react";
import { cookies } from "next/headers";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { api, safe, toApiParams, type SearchParams, type SearchResult } from "@/lib/api";
import { CURRENCY_COOKIE, isCurrency } from "@/lib/settings";
import { defaultCurrencyForLocale, type Locale } from "@/i18n/routing";
import { FiltersSidebar } from "@/components/filters-sidebar";
import { SortSelect } from "@/components/sort-select";
import { ListingGrid } from "@/components/listing-grid";
import { Pagination } from "@/components/pagination";

const EMPTY: SearchResult = { items: [], total: 0, page: 1, perPage: 30, totalPages: 0, facets: { brands: [], sources: [], conditions: [], movements: [], countries: [], dialColors: [], years: [] } };

type Props = { params: Promise<{ locale: string }>; searchParams: Promise<SearchParams> };

export async function generateMetadata({ params, searchParams }: Props): Promise<Metadata> {
  const { locale } = await params;
  const sp = await searchParams;
  const t = await getTranslations({ locale, namespace: "search" });
  const q = typeof sp.q === "string" ? sp.q : undefined;
  return { title: q ? `${q} – ${t("title")}` : t("title"), robots: { index: !Object.keys(sp).length } };
}

export default async function SearchPage({ params, searchParams }: Props) {
  const { locale } = await params;
  setRequestLocale(locale);
  const sp = await searchParams;
  const t = await getTranslations("search");
  const jar = await cookies();
  const cookieCurrency = jar.get(CURRENCY_COOKIE)?.value;
  const currency = isCurrency(cookieCurrency) ? cookieCurrency : defaultCurrencyForLocale[locale as Locale];

  const result = await safe(api.searchListings(toApiParams(sp, currency)), EMPTY);
  const q = typeof sp.q === "string" ? sp.q.trim() : "";
  const from = result.total === 0 ? 0 : (result.page - 1) * result.perPage + 1;
  const to = Math.min(result.total, result.page * result.perPage);

  return (
    <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">{t("resultsFor", { count: result.total, query: q || "none" })}</h1>
          {result.total > 0 && <p className="text-sm text-slate-500">{t("showing", { from, to, total: result.total.toLocaleString(locale) })}</p>}
        </div>
        <div className="flex items-center gap-2">
          <Suspense>
            <FiltersSidebarMobile facets={result.facets} />
          </Suspense>
          <Suspense>
            <SortSelect />
          </Suspense>
        </div>
      </div>

      <div className="mt-6 flex gap-8">
        <Suspense>
          <FiltersSidebarDesktop facets={result.facets} />
        </Suspense>
        <div className="min-w-0 flex-1">
          {result.items.length === 0 ? (
            <div className="rounded-lg border border-dashed border-slate-300 p-12 text-center">
              <p className="text-base font-medium text-slate-800">{t("noResults")}</p>
              <p className="mt-1 text-sm text-slate-500">{t("noResultsHint")}</p>
            </div>
          ) : (
            <ListingGrid items={result.items} />
          )}
          <Suspense>
            <Pagination page={result.page} totalPages={result.totalPages} />
          </Suspense>
        </div>
      </div>
    </div>
  );
}

// FiltersSidebar renders both the desktop aside and the mobile trigger; we mount it once on
// desktop (inside the flex row) and once as the mobile trigger via CSS visibility toggles.
function FiltersSidebarDesktop({ facets }: { facets: SearchResult["facets"] }) {
  return (
    <div className="hidden lg:block">
      <FiltersSidebar facets={facets} />
    </div>
  );
}
function FiltersSidebarMobile({ facets }: { facets: SearchResult["facets"] }) {
  return (
    <div className="lg:hidden">
      <FiltersSidebar facets={facets} />
    </div>
  );
}
