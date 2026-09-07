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
| `hourstack` | HTML (alapower shop), brand categories `/product.asp?cat=N&page=P` | TWD | ✅ cards parsed; reference/diameter/year extracted from title | shared parser in `crawler/alapower`; `ALAPOWER_FETCH_DETAILS=true` also fetches product pages for material/diameter (slower) |
| `rdwatch` | HTML (alapower shop, same markup as Hourstack; 230px cards, prices without separators) | TWD | ✅ 24/24 cards on the Rolex category; live dry-run OK (41 categories) | RD Watch, Taipei |
| `commitwatch` | **Shopify JSON feed** `/products.json?limit=250&page=N` | JPY | ✅ 30/30 with price, brand (from `vendor`), `Ref.` reference, images, condition grade from title; live dry-run OK | Commit Ginza, Tokyo; sold-out variants are skipped so they deactivate |
| `sevenhours` | HTML (Shopserve), all items via `/SHOP/list.php?Search=&Type=01&PAGE=N`, `<link rel=next>` | JPY | ✅ 40/40 cards with price, 税抜 price from the card, brand + reference from the title, image | 7hours, Kobe; a few hundred pieces, mostly new/unused |
| `lips` | HTML (EC-CUBE 4), in-stock watches newest first `/ec/products/list?category_id=1&stock_flg[]=1&orderby=2&pageno=N` | JPY | ✅ 40/40 cards with price, brand, 型番 reference, grade (新品 / SA / A ランク), 付属 box+papers, image | Brand Shop LIPS; ~1.1k in stock of ~43k catalogued, so the stock filter is essential |
| `allu` | **JSON API** `POST /jp/ja/api/v5/items/search?page=N&limit=30` with a category filter (c-79 = 腕時計) | JPY | ✅ 30/30 items with price, brand, reference, rank (N…C/J), image, seller | ALLU (Valuence) marketplace; tens of thousands of watches, so `CRAWL_MAX_PAGES` × 30 newest per run; house account "ALLU公式" = dealer, others = private |
| `housekihiroba` | HTML (ecbeing, Shift_JIS), brand categories `/shop/c/c01<code>/` from `/shop/c/cwatch/`, newest first `/shop/c/c01<code>_ssd_p<N>/` (32 per page) | JPY | ✅ 32/32 cards with price, brand, reference, condition icon, diameter, movement, material, image | 宝石広場, Shibuya; Rolex alone is >7k pieces (>200 pages), so `CRAWL_MAX_PAGES` bounds each brand |
| `ishida` | HTML (futureshop), pre-owned list newest first `/c/bestvintage/v_brand?page=N&sort=latest` (20 per page, `<link rel=next>`) | JPY | ✅ 20/20 cards with price, brand, reference (last line of the name), 【中古】 grade, movement/gender marks, gallery images | BEST ISHIDA, Tokyo; ~5.2k pieces / 260 pages, so `CRAWL_MAX_PAGES` bounds a run |
| `rww` | **JSON endpoint** `POST /app/php/data.php` (`felistCategory` → the four 手錶 categories, `felistproduct` 100 per page with `lang:"zh"`, `curr:"HKD"`) | HKD | ✅ 100/100 records with price, brand, reference (型號 line), grade from the name (全新 / 全新未用品 / 二手 N%新), diameter, year, images | RWW Watch, Hong Kong (尖沙咀 / 銅鑼灣 / 旺角 shops); sold pieces stay listed as "(已售)" and are skipped; robots.txt allows `/app/web/` only |
| `ebay` | **Browse API** (OAuth client credentials) | market currency | needs `EBAY_CLIENT_ID/SECRET` | per-brand aspect filter in Wristwatches (31387); free developer keys at developer.ebay.com |
| `chrono24` | HTML via Browserless (see [BROWSERLESS.md](BROWSERLESS.md)) | listing currency | ❌ plain requests get HTTP 403; selectors unverified until a rendered page is captured | marketplace: each card carries the seller's own country and city, so the DE on the sources row is never applied to a listing; skipped unless `CRAWL_RENDER_SERVICE_URL` or `CRAWL_PROXY_URL` is set; budget = `CHRONO24_BRANDS` (default Rolex, Omega, IWC, AP, PP) × `CHRONO24_MAX_PAGES` (default 3) |

Image URLs: Jackroad thumbnails (`/img/goods/S/<id>.jpg`) are upgraded to the product-page image
(`/img/goods/1/<id>.jpg`); Watchnian only serves the small `/S/` variant for the main image;
Hourstack links `product/product_big/*.JPG` directly. We never download or store images.

## Sources that need configuration

`ebay` and `chrono24` implement `crawler.Preflighter`. When their credentials / proxy are missing
the Runner logs a warning, writes a `skipped` row to `crawl_runs` (visible at `/api/v1/crawls`) and
carries on with the other sources; the job still exits 0. Silence the warning with
`CRAWL_SOURCES=<list>` or `UPDATE sources SET enabled = false WHERE key = '…'`.

## Japanese prices

`jackroad`, `watchnian`, `commitwatch`, `lips`, `allu`, `housekihiroba` and `ishida` all list tax-included prices; `sevenhours` prints its own 税抜 figure, which the adapter parses. The Runner fills
`PriceExclTax` for any source whose country is `JP` (see `crawler.ApplyTaxFreePrice`), and
`crawler.ComparisonPrice` decides what gets converted to USD. A new Japanese source therefore needs
no tax handling of its own. If a site publishes its own 税抜 figure, parse it into `PriceExclTax`
and the derivation will leave it alone.

## Non-watch items

Dealers list straps, buckles and bracelet links under their watch brands. `normalize.NonWatchBrands`
(currently Vagenari, a strap maker) and `normalize.NonWatchTitleMarkers` (currently 錶節, bracelet
links) name what is dropped: the Runner skips any listing whose title or model matches and counts
it as `excluded` in the run summary, and the alapower adapters skip a whole brand category whose
menu entry matches. Add to those two lists when a new part maker or part type shows up.

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

Two shortcuts exist: a shop built on the **alapower** ASP platform (Taiwanese dealers; look for
`product.asp?cat=` and `product_open.asp?ID=` links) only needs a 15-line adapter that fills
`alapower.Site{BaseURL, SellerName, Boilerplate}` — see `crawler/rdwatch`. A **Shopify** store
exposes `/products.json`; copy `crawler/commitwatch` and adjust the title/condition parsing.

Otherwise:

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
2. **Browserless** (`CRAWL_RENDER_SERVICE_URL` + `CRAWL_RENDER_SERVICE_TOKEN`) — either the hosted
   browserless.io Smart Scrape service or the official container on Cloud Run. Step-by-step setup, testing and
   budget notes: [BROWSERLESS.md](BROWSERLESS.md).
3. **Forward proxy** (`CRAWL_PROXY_URL`) — a residential proxy for plain fetches.

Review Chrono24's terms of use and robots.txt before enabling any of these.

## eBay setup

1. Create an app at https://developer.ebay.com → production keyset.
2. Store the Client ID / Secret in Secret Manager (`EBAY_CLIENT_ID`, `EBAY_CLIENT_SECRET`).
3. Optionally set `EBAY_MARKETPLACES=EBAY_US,EBAY_GB,EBAY_DE` (currency filter follows the market).

The default Browse API quota (5,000 calls/day) covers 25 brands × 1 marketplace × up to 50 pages;
lower `CRAWL_MAX_PAGES` for the eBay job or trim `ebay.Brands` if you add marketplaces.
