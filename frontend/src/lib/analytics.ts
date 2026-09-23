"use client";

/**
 * Product analytics (Amplitude). Two events, shared with the iOS app so dashboards line up:
 *   search        — a results page was shown (query, filters, sort, page, result count)
 *   view_listing  — a listing detail page was opened
 * Autocapture is limited to sessions: no page-view, click or form tracking. The key is a public
 * browser key; override it per environment with NEXT_PUBLIC_AMPLITUDE_API_KEY (empty disables).
 */
import * as amplitude from "@amplitude/analytics-browser";

export const AMPLITUDE_API_KEY = process.env.NEXT_PUBLIC_AMPLITUDE_API_KEY ?? "0f8cd19c77fc0325b6bdd8807e466e13";

let started = false;

export function startAnalytics(): boolean {
  if (started) return true;
  if (typeof window === "undefined" || !AMPLITUDE_API_KEY) return false;
  amplitude.init(AMPLITUDE_API_KEY, {
    autocapture: { sessions: true, pageViews: false, formInteractions: false, fileDownloads: false, elementInteractions: false },
    serverZone: "US",
  });
  started = true;
  return true;
}

export type SearchEvent = {
  query?: string;
  brand?: string;
  model?: string;
  ref?: string;
  source?: string;
  condition?: string;
  movement?: string;
  gender?: string;
  country?: string;
  dial?: string;
  price_min?: string;
  price_max?: string;
  year_min?: string;
  year_max?: string;
  diameter_min?: string;
  diameter_max?: string;
  box?: string;
  papers?: string;
  sort?: string;
  page?: number;
  result_count: number;
  currency: string;
  locale: string;
};

export type ViewListingEvent = {
  listing_id: number;
  source: string;
  brand?: string;
  model?: string;
  ref?: string;
  condition: string;
  price_usd?: number;
  currency: string;
  locale: string;
};

export function trackSearch(e: SearchEvent) {
  if (startAnalytics()) amplitude.track("search", { platform: "web", ...e });
}

export function trackViewListing(e: ViewListingEvent) {
  if (startAnalytics()) amplitude.track("view_listing", { platform: "web", ...e });
}
