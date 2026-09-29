# Architecture

## Goals

* One nightly pass over five marketplaces; results queryable and sortable like Chrono24.
* Cheap to run: one small Cloud Run service, one job, one Cloud SQL instance, a Vercel Hobby/Pro project.
* No image storage: we keep the original image URLs only.
* Currency and language are per-visitor settings, applied at render time.

## Backend (`backend/`)

```
cmd/api        HTTP server (Cloud Run service)
cmd/crawler    one crawl cycle (Cloud Run job)
cmd/migrate    apply / roll back embedded SQL migrations
internal/
  alerts/      push alerts: new-find matching after each crawl, notification copy (5 languages)
  api/         chi router, handlers, rates cache, alert endpoints
  config/      env-based configuration
  crawler/     Fetcher (rate limit, retries, charset), Source interface, Runner (persistence)
  crawler/<site>/  one adapter per marketplace (+ fixture tests using real captured HTML/JSON)
  crawler/alapower/ shared parser for the ASP shop platform used by Hourstack and RD Watch
  db/          pgx pool, embedded migrations (advisory-locked on one connection, idempotent;
               applied on start unless AUTO_MIGRATE=false, which production sets on both workloads)
  fx/          USD-based exchange rates
  push/        OneSignal REST client (APNs + Web Push)
  search/      query-parameter parsing shared by the listings endpoint and alerts
  model/       domain types
  normalize/   brand dictionary (EN/繁中/日本語 aliases), price/ref/diameter parsing, condition mapping
  repository/  all SQL: upsert with price history, search with facets, similar listings
```

### Data model

* `listings` — one row per `(source_id, external_id)`. `dial_color` is a canonical key (black,
  blue, champagne, mother_of_pearl, …) detected from multilingual titles at crawl time and
  refreshed on every upsert; `has_box` / `has_papers` come from phrases such as 原廠盒單, 有盒單,
  無盒單, 箱・保証書あり, "full set". `price` is the price as listed and `price_excl_tax` the
  tax-free (税抜) amount for Japanese sources. `price_usd` is denormalized for cross-source
  sorting and range filters; a generated `tsvector` column powers full-text search (brand and
  reference weighted A, model B, title C). Sold/vanished listings are soft-deleted via `is_active`.
* `price_history` — a row whenever a listing's price changes; drives the price chart and "deal" badges.
* `brands` — canonical brand with denormalized `listing_count`.
* `exchange_rates` — `1 USD = rate QUOTE`, refreshed at the start of every crawl.

### Japanese consumption tax

Japan requires consumer prices to be displayed tax-included (総額表示義務), so every Japanese source
lists 税込 prices, while an exporting buyer pays the tax-free amount. The crawler stores both:
`price` as listed and `price_excl_tax` derived as `ceil(price / 1.1)`. That formula is not an
approximation — Jackroad publishes its own TAXFREE figure on product pages, and ceil matched it on
14 of 14 sampled products where round-half matched only 7. `price_usd`, which drives all sorting
and price-range filters, is computed from the tax-free amount when one exists, so a Tokyo listing
is not overstated by 10% against a Munich one. The detail page shows both figures; cards and the
comparison table show the tax-free basis so what is displayed agrees with the sort order.
An adapter that parses a published tax-free price keeps it; the derivation only fills gaps.
* `crawl_runs` — audit log per source per run (counts, status, error).
* `alerts` — saved searches for push notifications: the device's random `subscriber_id` (its
  OneSignal external_id), a display name, the criteria as a `/api/v1/listings` query string
  (including the currency its price bounds are in) and `checked_at`, the watermark for new finds.

### Crawl cycle

1. Refresh FX rates (keyless provider, configurable).
2. For each enabled source (bounded concurrency, per-host rate limiting, 3 retries with backoff):
   emit listings → normalize (brand, reference, condition, movement, gender, box/papers, diameter,
   year) → convert price to USD → upsert (price history if changed).
3. If the source crawl completed, deactivate its listings not seen for `CRAWL_DEACTIVATE_STALE_AFTER`
   (72h default) so one blocked night does not empty the index.
4. Refresh brand counts and write the `crawl_runs` row.
5. Push alerts (skipped without OneSignal credentials, or with `-skip-alerts`): for each alert,
   active listings matching its criteria with `first_seen_at` after its `checked_at` are its new
   finds. One notification per alert names the listing (or the count and the cheapest), links to
   it on the web and carries `listingId`/`query` for the iOS app. `checked_at` advances to the
   notifier's start time only after a successful send, so a OneSignal outage retries the same
   finds next night instead of dropping them. This also runs after a partial crawl.

### Push alerts

Clients have no accounts. The web app and the iOS app each generate a random subscriber id
(localStorage / UserDefaults), log in to OneSignal with it as `external_id`, and send it in the
`X-Subscriber-ID` header to `GET/POST /api/v1/alerts` and `DELETE /api/v1/alerts/{id}`. The web
calls these from Next.js server actions, so CORS stays GET-only. Alerts must have at least one
criterion (a query that matches everything is rejected with 422); a device can keep 20.

### Search

`repository.SearchListings` builds one parameterized `WHERE` clause and runs three things:
count + price bounds, the page itself, and five facet aggregations in a single pgx batch.
Sort keys map to indexed columns.

Text search is multilingual by construction, because most inventory is titled in Japanese or
Traditional Chinese while buyers search in any of four languages:

1. `websearch_to_tsquery('simple', q)` against the weighted `tsvector` (brand and reference
   weight A, model B, title C) — matches Latin words and reference numbers such as `116610LN`.
2. The query is **canonicalized** (`normalize.CanonicalizeQuery`): brand and model aliases in
   Japanese/Chinese are rewritten to English (`勞力士 水鬼` → `Rolex Submariner`) and OR-ed in.
   At crawl time the same dictionary sets `listings.model` to the English collection name
   (`サブマリーナー` → `Submariner`), so a Japanese listing is found by an English query and vice versa.
3. PostgreSQL's `simple` parser drops CJK characters entirely, so any non-ASCII query word is
   also matched with `title ILIKE '%word%'` (all words required) using a `pg_trgm` GIN index
   (migration 0002). This covers dealer-specific wording the dictionary does not know.

## Frontend (`frontend/`)

* Next.js 16 App Router, all data fetched in Server Components from the Go API (`API_URL` is
  server-only), with `revalidate` windows per endpoint. Backend outages degrade to empty states.
* Analytics: Amplitude (`src/lib/analytics.ts`, `<Analytics>` in the layout) with two events,
  `search` on every results page and `view_listing` on every listing page — the same names and
  properties the iOS app sends. Session autocapture only; `NEXT_PUBLIC_AMPLITUDE_API_KEY=` (empty)
  disables it for an environment.
* `next-intl` with `[locale]` routing (`en`, `zh-TW`, `zh-CN`, `ja`, `de`; default locale un-prefixed).
* Currency is a cookie (`wc_currency`, set through a Server Action); defaults per locale
  (en→USD, zh-TW→TWD, zh-CN→CNY, ja→JPY, de→EUR). `<Price>` converts `price_usd` client-side with the rates
  provided by the layout and always shows the seller's original price when it differs.
* Listing images are hot-linked with `referrerPolicy="no-referrer"`; the Next image optimizer is
  disabled so no third-party image passes through Vercel.
* Filters live in the URL (`/search?brand=rolex&price_max=5000&sort=price_asc`), so every result
  page is shareable and cacheable.

## Non-goals / future work

* Authentication, saved searches and price alerts (the schema already tracks price history).
* Chrono24 without a licensed feed or rendering proxy (see CRAWLERS.md).
* Image proxying/caching.
