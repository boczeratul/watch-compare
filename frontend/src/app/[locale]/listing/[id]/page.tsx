import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { getLocale, getTranslations, setRequestLocale } from "next-intl/server";
import { ExternalLink, MapPin, Store, User } from "lucide-react";
import { Link } from "@/i18n/navigation";
import { api, ApiError, safe, type Listing } from "@/lib/api";
import { ImageGallery } from "@/components/image-gallery";
import { Price } from "@/components/price";
import { SourceBadge } from "@/components/source-badge";
import { CompareTable } from "@/components/compare-table";
import { PriceHistory } from "@/components/price-history";

type Props = { params: Promise<{ locale: string; id: string }> };

async function load(id: string): Promise<Listing | null> {
  if (!/^\d+$/.test(id)) return null;
  try {
    return await api.getListing(id);
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null;
    throw err;
  }
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { id } = await params;
  const l = await load(id);
  if (!l) return {};
  return {
    title: l.title,
    description: [l.brandName, l.model, l.referenceNumber, l.sourceName].filter(Boolean).join(" · "),
    openGraph: { images: l.imageUrls.slice(0, 1) },
  };
}

export default async function ListingPage({ params }: Props) {
  const { locale, id } = await params;
  setRequestLocale(locale);
  const listing = await load(id);
  if (!listing) notFound();
  const [t, tc, tm, tg, td, currentLocale, similar, history] = await Promise.all([
    getTranslations("listing"),
    getTranslations("condition"),
    getTranslations("movement"),
    getTranslations("gender"),
    getTranslations("dial"),
    getLocale(),
    safe(api.similar(id), { items: [] }),
    safe(api.priceHistory(id), { items: [] }),
  ]);
  const date = (s: string) => new Intl.DateTimeFormat(currentLocale, { dateStyle: "medium" }).format(new Date(s));
  const yesNo = (v?: boolean) => (v === undefined ? "—" : v ? t("yes") : t("no"));
  const cheaperOther = similar.items.find((s) => s.priceUsd !== undefined);
  const savePct = listing.priceUsd && cheaperOther?.priceUsd && cheaperOther.priceUsd > listing.priceUsd ? Math.round((1 - listing.priceUsd / cheaperOther.priceUsd) * 100) : 0;

  const specs: [string, React.ReactNode][] = [
    [t("brand"), listing.brandName ? <Link href={{ pathname: "/search", query: { brand: listing.brand ?? "" } }} className="underline">{listing.brandName}</Link> : "—"],
    [t("model"), listing.model || "—"],
    [t("reference"), listing.referenceNumber ? <Link href={{ pathname: "/search", query: { ref: listing.referenceNumber } }} className="underline">{listing.referenceNumber}</Link> : "—"],
    [t("condition"), tc(listing.condition)],
    [t("year"), listing.year ?? "—"],
    [t("diameter"), listing.caseDiameterMm ? `${listing.caseDiameterMm} mm` : "—"],
    [t("material"), listing.caseMaterial || "—"],
    [t("dialColor"), listing.dialColor ? <Link href={{ pathname: "/search", query: { dial: listing.dialColor, ...(listing.brand ? { brand: listing.brand } : {}) } }} className="underline">{td(listing.dialColor)}</Link> : "—"],
    [t("movement"), tm(listing.movement)],
    [t("gender"), tg(listing.gender)],
    [t("boxPapers"), `${yesNo(listing.hasBox)} / ${yesNo(listing.hasPapers)}`],
    [t("location"), [listing.locationCity, listing.locationCountry].filter(Boolean).join(", ") || "—"],
    [t("firstSeen"), date(listing.firstSeenAt)],
    [t("lastSeen"), date(listing.lastSeenAt)],
  ];

  return (
    <div className="mx-auto max-w-7xl px-4 py-6 sm:px-6">
      {!listing.isActive && (
        <p className="mb-4 rounded-md border border-amber-200 bg-amber-50 px-4 py-2 text-sm text-amber-900">{t("inactive", { source: listing.sourceName })}</p>
      )}
      <div className="grid gap-8 lg:grid-cols-[minmax(0,7fr)_minmax(0,5fr)]">
        <div>
          <ImageGallery images={listing.imageUrls} alt={listing.title} />
        </div>
        <div className="space-y-6">
          <div>
            <div className="flex items-center gap-2">
              <SourceBadge source={listing.source} name={listing.sourceName} />
              <span className="text-xs text-slate-500">
                {listing.sellerType === "private" ? <User className="inline h-3.5 w-3.5" aria-hidden /> : <Store className="inline h-3.5 w-3.5" aria-hidden />}{" "}
                {listing.sellerName && listing.sellerName !== listing.sourceName ? listing.sellerName : listing.sellerType === "private" ? t("private") : t("dealer")}
              </span>
            </div>
            <p className="mt-3 text-sm font-semibold uppercase tracking-wide text-slate-500">{listing.brandName}</p>
            <h1 className="mt-1 text-2xl font-bold text-slate-900">{listing.title}</h1>
            {(listing.locationCity || listing.locationCountry) && (
              <p className="mt-2 flex items-center gap-1 text-sm text-slate-500">
                <MapPin className="h-4 w-4" aria-hidden />
                {[listing.locationCity, listing.locationCountry].filter(Boolean).join(", ")}
              </p>
            )}
          </div>

          <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
            <Price usd={listing.priceUsd} original={listing.price} originalCurrency={listing.currency} size="lg" />
            {listing.shippingPrice !== undefined && (
              <p className="mt-1 text-xs text-slate-500">{t("shipping")}: <Price usd={undefined} original={listing.shippingPrice} originalCurrency={listing.currency} size="sm" showOriginal={false} /></p>
            )}
            {savePct > 0 && <p className="mt-2 text-sm font-medium text-emerald-700">{t("saveVs", { percent: savePct })}</p>}
            <a href={listing.url} target="_blank" rel="noopener noreferrer nofollow" className="mt-4 inline-flex h-11 w-full items-center justify-center gap-2 rounded-md bg-slate-900 text-sm font-semibold text-white hover:bg-slate-800">
              {t("viewOnSource", { source: listing.sourceName })}
              <ExternalLink className="h-4 w-4" aria-hidden />
            </a>
          </div>

          <section className="rounded-lg border border-slate-200 bg-white">
            <h2 className="border-b border-slate-200 px-4 py-3 text-base font-semibold text-slate-900">{t("details")}</h2>
            <dl className="divide-y divide-slate-100 text-sm">
              {specs.map(([k, v]) => (
                <div key={k} className="grid grid-cols-[minmax(0,2fr)_minmax(0,3fr)] gap-2 px-4 py-2">
                  <dt className="text-slate-500">{k}</dt>
                  <dd className="text-slate-900">{v}</dd>
                </div>
              ))}
            </dl>
          </section>
        </div>
      </div>

      <div className="mt-10 grid gap-6 lg:grid-cols-[minmax(0,7fr)_minmax(0,5fr)]">
        <CompareTable current={listing} others={similar.items} />
        <div className="space-y-6">
          <PriceHistory points={history.items} />
          {listing.description && (
            <section className="rounded-lg border border-slate-200 bg-white p-4">
              <h2 className="mb-2 text-base font-semibold text-slate-900">{t("description")}</h2>
              <p className="whitespace-pre-line text-sm text-slate-700">{listing.description}</p>
            </section>
          )}
        </div>
      </div>
    </div>
  );
}
