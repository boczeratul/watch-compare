/**
 * Typed client for the Go API. All calls run on the server (RSC / route handlers), so the
 * API base URL never reaches the browser.
 */
import "server-only";

const API_URL = process.env.API_URL ?? "http://localhost:8080";

export type Condition = "new" | "unworn" | "very_good" | "good" | "fair" | "poor" | "unknown";
export type Movement = "automatic" | "manual" | "quartz" | "unknown";
export type Gender = "men" | "women" | "unisex" | "unknown";
export type DialColor =
  | "black" | "white" | "silver" | "blue" | "green" | "grey" | "champagne" | "gold" | "brown" | "red"
  | "pink" | "salmon" | "purple" | "yellow" | "orange" | "mother_of_pearl" | "meteorite" | "cream" | "bronze";

export interface Listing {
  id: number;
  source: string;
  sourceName: string;
  externalId: string;
  url: string;
  title: string;
  brand?: string;
  brandName?: string;
  model?: string;
  referenceNumber?: string;
  condition: Condition;
  year?: number;
  caseDiameterMm?: number;
  caseMaterial?: string;
  dialColor?: DialColor;
  movement: Movement;
  gender: Gender;
  hasBox?: boolean;
  hasPapers?: boolean;
  price?: number;
  currency?: string;
  priceUsd?: number;
  shippingPrice?: number;
  locationCountry?: string;
  locationCity?: string;
  sellerName?: string;
  sellerType?: string;
  imageUrls: string[];
  description?: string;
  attributes?: Record<string, unknown>;
  isActive: boolean;
  firstSeenAt: string;
  lastSeenAt: string;
}

export interface FacetValue { key: string; label: string; count: number }
export interface Facets {
  brands: FacetValue[];
  sources: FacetValue[];
  conditions: FacetValue[];
  movements: FacetValue[];
  countries: FacetValue[];
  dialColors: FacetValue[];
  years: FacetValue[];
  priceMinUsd?: number;
  priceMaxUsd?: number;
}
export interface SearchResult {
  items: Listing[];
  total: number;
  page: number;
  perPage: number;
  totalPages: number;
  facets: Facets;
}
export interface Brand { id: number; slug: string; name: string; listingCount: number }
export interface Source { id: number; key: string; name: string; baseUrl: string; country: string; currency: string; enabled: boolean }
export interface ExchangeRate { quote: string; rate: number; fetchedAt: string }
export interface RatesResponse { base: "USD"; supported: string[]; items: ExchangeRate[] }
export interface PricePoint { price?: number; currency: string; priceUsd?: number; observedAt: string }
export interface Stats {
  activeListings: number;
  brands: number;
  lastCrawlAt: string | null;
  sources: { key: string; name: string; count: number }[];
}

export type SortKey = "relevance" | "price_asc" | "price_desc" | "newest" | "oldest" | "year_desc" | "year_asc" | "size_asc" | "size_desc";
export const SORT_KEYS: SortKey[] = ["relevance", "newest", "price_asc", "price_desc", "year_desc", "year_asc", "size_asc", "size_desc"];

export type SearchParams = Record<string, string | string[] | undefined>;

class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

/**
 * The frontend (Vercel) and the API (Cloud Run) deploy independently, so during any deploy window
 * the running API can predate a field the UI already reads. Postgres also has no way to express
 * "empty array" for a Go nil slice, which marshals to JSON null. Both produce runtime crashes in
 * components that spread or map these values, so every response is normalized here at the
 * boundary: after this, components can assume each array exists and is iterable.
 */
const EMPTY_FACETS: Facets = { brands: [], sources: [], conditions: [], movements: [], countries: [], dialColors: [], years: [] };

function arr<T>(v: T[] | null | undefined): T[] {
  return Array.isArray(v) ? v : [];
}

function normalizeListing(l: Listing): Listing {
  return { ...l, imageUrls: arr(l.imageUrls) };
}

function normalizeSearchResult(r: SearchResult): SearchResult {
  return {
    ...r,
    items: arr(r.items).map(normalizeListing),
    facets: { ...EMPTY_FACETS, ...(r.facets ?? {}) },
  };
}

function normalizeItems<T>(r: { items: T[] }): { items: T[] } {
  return { ...r, items: arr(r.items) };
}

async function get<T>(path: string, params?: Record<string, string | undefined>, revalidate = 60): Promise<T> {
  const url = new URL(path, API_URL);
  for (const [k, v] of Object.entries(params ?? {})) {
    if (v !== undefined && v !== "") url.searchParams.set(k, v);
  }
  const res = await fetch(url, { next: { revalidate }, headers: { Accept: "application/json" } });
  if (!res.ok) {
    throw new ApiError(res.status, `${res.status} ${res.statusText} for ${url.pathname}`);
  }
  return (await res.json()) as T;
}

function first(v: string | string[] | undefined): string | undefined {
  return Array.isArray(v) ? v[0] : v;
}

/** Translate Next.js searchParams into API query params (1:1 names, validated upstream). */
export function toApiParams(sp: SearchParams, currency: string): Record<string, string | undefined> {
  const keys = ["q", "brand", "model", "ref", "source", "condition", "movement", "gender", "country", "dial",
    "price_min", "price_max", "year_min", "year_max", "diameter_min", "diameter_max", "box", "papers", "sort", "page", "per_page"];
  const out: Record<string, string | undefined> = { currency };
  for (const k of keys) out[k] = first(sp[k]);
  return out;
}

export const api = {
  searchListings: (params: Record<string, string | undefined>) =>
    get<SearchResult>("/api/v1/listings", params, 60).then(normalizeSearchResult),
  getListing: (id: string | number) => get<Listing>(`/api/v1/listings/${id}`, undefined, 120).then(normalizeListing),
  similar: (id: string | number) =>
    get<{ items: Listing[] }>(`/api/v1/listings/${id}/similar`, undefined, 120).then((r) => ({ items: arr(r.items).map(normalizeListing) })),
  priceHistory: (id: string | number) => get<{ items: PricePoint[] }>(`/api/v1/listings/${id}/price-history`, undefined, 300).then(normalizeItems),
  brands: () => get<{ items: Brand[] }>("/api/v1/brands", undefined, 600).then(normalizeItems),
  sources: () => get<{ items: Source[] }>("/api/v1/sources", undefined, 3600).then(normalizeItems),
  rates: () => get<RatesResponse>("/api/v1/rates", undefined, 600).then((r) => ({ ...r, items: arr(r.items), supported: arr(r.supported) })),
  stats: () => get<Stats>("/api/v1/stats", undefined, 300).then((r) => ({ ...r, sources: arr(r.sources) })),
};

export { ApiError };

/** Wrap a fetch so a backend outage degrades to an empty state instead of a 500 page. */
export async function safe<T>(p: Promise<T>, fallback: T): Promise<T> {
  try {
    return await p;
  } catch (err) {
    console.error("[api]", err instanceof Error ? err.message : err);
    return fallback;
  }
}
