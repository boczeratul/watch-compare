# Crawlers

Every adapter implements `crawler.Source`:

```go
type Source interface {
    Key() string                                   // matches sources.key
    Crawl(ctx context.Context, env *Env, emit Emit) error
}
```

Adapters only *parse*; the `Runner` does normalization, FX conversion, persistence and
deactivation. Parsers are covered by fixture tests using real HTML captured in `testdata/`.

| Source | Method | Currency | Verified 2026‑09 | Notes |
|--------|--------|----------|------------------|-------|
| `jackroad` | HTML, Shift_JIS, brand categories `/shop/r/rjw<code>/?p=N` discovered from `/shop/r/rjw/` (174 brands) | JPY | ✅ 30/30 cards parsed with price, brand, reference, diameter, movement, image; live dry-run OK | ebisumart shop; brand from Japanese name |
| `watchnian` | HTML, `/shop/r/rwatch_pN/` | JPY | ✅ 60/60 cards; condition from dealer rank (S/A/B/C) | use bare host `watchnian.com` (www returned CloudFront 502) |
| `hourstack` | HTML, brand categories `/product.asp?cat=N&page=P` | TWD | ✅ cards parsed; reference/diameter/year extracted from title | `HOURSTACK_FETCH_DETAILS=true` also fetches product pages for material/diameter (slower) |
| `ebay` | **Browse API** (OAuth client credentials) | market currency | needs `EBAY_CLIENT_ID/SECRET` | per-brand aspect filter in Wristwatches (31387); free developer keys at developer.ebay.com |
| `chrono24` | HTML via render service or proxy | listing currency | ❌ plain requests get HTTP 403 | requires `CRAWL_RENDER_SERVICE_URL` or `CRAWL_PROXY_URL`; fails fast otherwise |

Image URLs: Jackroad thumbnails (`/img/goods/S/<id>.jpg`) are upgraded to the product-page image
(`/img/goods/1/<id>.jpg`); Watchnian only serves the small `/S/` variant for the main image;
Hourstack links `product/product_big/*.JPG` directly. We never download or store images.

## Politeness and reliability

* Per-host rate limit (`CRAWL_RATE_LIMIT_MS`, default 1.5 s) using a token bucket; 3 retries with
  exponential backoff on 429/5xx/network errors.
* `CRAWL_MAX_PAGES` caps list pages per source (or per brand/category).
* Deduplication per run on `external_id`; upserts are idempotent so re-running is safe.
* A source crawl that errors is marked `failed`/`partial` in `crawl_runs` and **does not**
  deactivate anything. Deactivation only happens after a complete pass.

## Running

```bash
make crawl-dry ARGS="-sources=jackroad"                     # print 25 parsed listings, no DB writes
make crawl ARGS="-sources=hourstack,watchnian -max-pages=3" # partial crawl
go run ./cmd/crawler -sources=ebay                          # needs eBay credentials
```

Useful flags: `-dry-run`, `-max-pages`, `-skip-fx`, `-timeout`. Production runs use the defaults
from the Cloud Run Job environment.

## Adding a marketplace

1. Capture a list page: `curl -A "$UA" <url> > internal/crawler/<key>/testdata/list.html`
   (use `iconv` for non-UTF-8 sites; the Fetcher decodes charsets automatically at runtime).
2. Implement `ParseList(doc) []model.Listing` and a `Crawl` that pages until no new IDs appear.
3. Fill what the site gives you (`ExternalID`, `URL`, `Title`, `Price`, `Currency`, `ImageURLs`,
   condition, etc.). Leave the rest empty — `EnrichFromTitle` fills brand, reference, diameter,
   year, box/papers heuristically.
4. Add a fixture test asserting that most cards have price, brand and image.
5. Insert the source row in a new migration (`INSERT INTO sources …`) and register the adapter in
   `cmd/crawler/main.go`.
6. Add brand aliases in `normalize/brands.go` for any names in the site's language.

## Chrono24 options

Chrono24 fronts its site with bot protection; a data-centre IP receives 403 immediately.
Supported ways to run the adapter, in order of preference:

1. **Licensed feed** — Chrono24 offers partner/affiliate data. If you have one, replace the HTML
   parser with the feed client; the rest of the pipeline stays the same.
2. **Rendering service** (`CRAWL_RENDER_SERVICE_URL`) — a headless browser endpoint you operate
   (e.g. Browserless/Playwright on Cloud Run) or a commercial rendering proxy. The adapter calls
   `<url>?url=<page>` and expects rendered HTML.
3. **Forward proxy** (`CRAWL_PROXY_URL`) — a residential proxy for plain fetches.

Review Chrono24's terms of use and robots.txt before enabling any of these.

## eBay setup

1. Create an app at https://developer.ebay.com → production keyset.
2. Store the Client ID / Secret in Secret Manager (`EBAY_CLIENT_ID`, `EBAY_CLIENT_SECRET`).
3. Optionally set `EBAY_MARKETPLACES=EBAY_US,EBAY_GB,EBAY_DE` (currency filter follows the market).

The default Browse API quota (5,000 calls/day) covers 25 brands × 1 marketplace × up to 50 pages;
lower `CRAWL_MAX_PAGES` for the eBay job or trim `ebay.Brands` if you add marketplaces.
