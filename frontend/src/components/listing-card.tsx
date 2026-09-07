"use client";

import { useTranslations } from "next-intl";
import { MapPin } from "lucide-react";
import { Link } from "@/i18n/navigation";
import type { Listing } from "@/lib/api";
import { Price } from "./price";
import { SourceBadge } from "./source-badge";

export function ListingCard({ listing }: { listing: Listing }) {
  const tc = useTranslations("condition");
  const td = useTranslations("dial");
  const img = listing.imageUrls[0];
  return (
    <Link href={`/listing/${listing.id}`} className="group flex h-full flex-col overflow-hidden rounded-lg border border-slate-200 bg-white transition hover:border-slate-400 hover:shadow-md">
      <div className="relative aspect-square w-full bg-slate-100">
        {img ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={img} alt={listing.title} loading="lazy" decoding="async" referrerPolicy="no-referrer" className="h-full w-full object-contain p-3 transition group-hover:scale-[1.03]" />
        ) : (
          <div className="flex h-full items-center justify-center text-slate-300">—</div>
        )}
        <SourceBadge source={listing.source} name={listing.sourceName} className="absolute left-2 top-2" />
      </div>
      <div className="flex flex-1 flex-col gap-1 p-3">
        <p className="text-xs font-semibold uppercase tracking-wide text-slate-500">{listing.brandName ?? " "}</p>
        <h3 className="line-clamp-2 text-sm font-medium text-slate-900" title={listing.title}>
          {listing.title}
        </h3>
        <p className="text-xs text-slate-500">
          {[listing.referenceNumber, listing.year, listing.caseDiameterMm ? `${listing.caseDiameterMm} mm` : null, listing.dialColor ? td(listing.dialColor) : null].filter(Boolean).join(" · ") || " "}
        </p>
        <div className="mt-auto flex items-end justify-between gap-2 pt-2">
          {/* priceUsd is computed from the tax-free amount, so show that as the original too;
              otherwise a Japanese card would display a number that contradicts the sort order. */}
          <Price usd={listing.priceUsd} original={listing.priceExclTax ?? listing.price} originalCurrency={listing.currency} />
          <span className="rounded bg-slate-100 px-1.5 py-0.5 text-[11px] text-slate-600">{tc(listing.condition)}</span>
        </div>
        {(listing.locationCountry || listing.locationCity) && (
          <p className="flex items-center gap-1 text-[11px] text-slate-500">
            <MapPin className="h-3 w-3" aria-hidden />
            {[listing.locationCity, listing.locationCountry].filter(Boolean).join(", ")}
          </p>
        )}
      </div>
    </Link>
  );
}
