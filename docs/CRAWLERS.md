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
| `999watch` | HTML (alapower shop, same markup as RD Watch), brand categories `/product.asp?cat=N&page=P` (24 per page) | TWD | ⚠️ no captured fixture: 999watch.com is unreachable from the dev sandbox; 24/24 Rolex cards with ID, title, 編號 and price seen through a page reader (2026-10-20) | 鴻昇名錶精品中心, Taichung (package `crawler/hongsheng`; Go packages cannot start with a digit); the 手錶配件及錶帶 category is skipped |
| `369rolexwatch` | HTML (alapower shop; plain-text "售價 : 368,000元", spacer-gif card images, `index=` pager), brand categories `/product.asp?cat=N` | TWD | ✅ live crawl 2026-10-31: 26/26 pieces (the whole stock) with price, brand, photo; 25 with reference | 金宸名錶, Taichung (package `crawler/jinchen`); session pager like goodtimezone, though no category currently has a second page |
| `dzwatch` | HTML (alapower shop, same markup as RD Watch), brand categories `/product.asp?cat=N&page=P` (24 per page) | TWD | ✅ live crawl 2026-10-31: 232 pieces, all priced with photos, 218 with reference (warranty dates like "2019/12" in titles are not taken as references) | 大眾名錶, Kaohsiung; jewellery (GIA 鑽石, 玉石, 珍珠, 珊瑚, 飾品), 手錶配件 and 其他 categories are skipped |
| `goodtimezone` | HTML (alapower shop; lowercase `product_open.asp?id=`, `/product/product_BIG/` photos, `index=` pager) | TWD | ✅ live crawl 2026-10-31: 270 pieces (137 Rolex over 10 pages), all priced | 好時計, Taipei; the pager links are `product.asp?index=P` without `cat=` because the category lives in the ASP session (`cat=N&index=P` returns an empty list), so page 1 opens the category and later pages reuse its cookie |
| `egps` | HTML (alapower shop, `products.asp` / `products_open.asp` variant), only `products.asp?cat=2&type=open` (品牌男錶) and `cat=1` (品牌女錶), `&page=P` | TWD | ✅ live crawl 2026-10-31: 691 pieces, all priced, 608 with reference | 永生／永久名錶, Taichung (two shops); pages are Big5; every card has a `td.shopping_Price` holding the store phone ahead of the real `span.shopping_Price`; shop notices posted as products ("…公告") are skipped; mostly new-card pieces at market price |
| `xintai` | HTML (custom PHP), `/goods.php?Ty=1&currentPage=N` (腕錶類, 8 per page) | TWD | ✅ live crawl 2026-10-31: 38 pieces, 37 priced; card text has no diameter | 鑫泰當鋪 (89675555.com), Tucheng, New Taipei; pawn shop, so condition defaults to good; "(已售出)" pieces with the 999,999 placeholder price are skipped; listings can be months old, stock is confirmed over LINE |
| `watchstore` | HTML (classic ASP on iPAGE), every piece on the home page list `/?keywords=&larcode=&newsclass=&page=N` | TWD | ✅ live crawl 2026-10-31: 132 watches; 8 accessories (watch boxes, travel cases, straps) are dropped by the exclusion rules | 名錶窩, Taipei; thumbnails (`_S.jpg`) only |
| `commitwatch` | HTML (Shopify theme), `/collections/all?sort_by=created-descending&page=N` (48 cards per page, newest first) | JPY | ✅ 22/22 for-sale cards on the 2026-10-20 capture with price + 税抜, brand, 型番 reference, condition, 2 images | Commit Ginza, Tokyo; `/products.json` is oldest-first and capped at 25000 while the shop keeps 27000+ sold pieces, so it never reaches current stock; SOLD OUT / RESERVED cards are skipped and the crawl stops after 10 pages in a row with nothing for sale |
| `sevenhours` | HTML (Shopserve), all items via `/SHOP/list.php?Search=&Type=01&PAGE=N`, `<link rel=next>` | JPY | ✅ 40/40 cards with price, 税抜 price from the card, brand + reference from the title, image | 7hours, Kobe; a few hundred pieces, mostly new/unused |
| `lips` | HTML (EC-CUBE 4), in-stock watches newest first `/ec/products/list?category_id=1&stock_flg[]=1&orderby=2&pageno=N` | JPY | ✅ 40/40 cards with price, brand, 型番 reference, grade (新品 / SA / A ランク), 付属 box+papers, image | Brand Shop LIPS; ~1.1k in stock of ~43k catalogued, so the stock filter is essential |
| `allu` | **JSON API** `POST /jp/ja/api/v5/items/search?page=N&limit=30` with a category filter (c-79 = 腕時計) | JPY | ✅ 30/30 items with price, brand, reference, rank (N…C/J), image, seller | ALLU (Valuence) marketplace; tens of thousands of watches, so `CRAWL_MAX_PAGES` × 30 newest per run; house account "ALLU公式" = dealer, others = private |
| `ishida` | HTML (futureshop), pre-owned list newest first `/c/bestvintage/v_brand?page=N&sort=latest` (20 per page, `<link rel=next>`) | JPY | ✅ 20/20 cards with price, brand, reference (last line of the name), 【中古】 grade, movement/gender marks, gallery images | BEST ISHIDA, Tokyo; ~5.2k pieces / 260 pages, so `CRAWL_MAX_PAGES` bounds a run |
| `rww` | **JSON endpoint** `POST /app/php/data.php` (`felistCategory` → the four 手錶 categories, `felistproduct` 100 per page with `lang:"zh"`, `curr:"HKD"`) | HKD | ✅ 100/100 records with price, brand, reference (型號 line), grade from the name (全新 / 全新未用品 / 二手 N%新), diameter, year, images | RWW Watch, Hong Kong (尖沙咀 / 銅鑼灣 / 旺角 shops); sold pieces stay listed as "(已售)" and are skipped; robots.txt allows `/app/web/` only |
| `watchfinderhk` | HTML (Magento 2), `/watches?p=N` (42 per page, `<link rel=next>`) | HKD | ✅ 42/42 cards with price (`data-price-amount`), brand / series / model from card data attributes, year, box and papers icons, image | Watchfinder & Co., Hong Kong; ~5.5k pieces / 131 pages, so the whole catalogue fits in the default page budget; condition is only on the product page, so it stays unknown |
| `wristcheck` | **Algolia index** `prod_product_listings` (public search key from the page), 100 hits per page, `filters=NOT status:sold` | HKD | ✅ 100/100 hits with price, brand, reference, condition (WATCH_UNWORN / WATCH_WORN + grade out of 10), box/papers, year purchased, diameter, gender, images | Wristcheck, Hong Kong with a New York showroom: `location` decides HK or US per listing (marketplace rule); ~1.8k available of ~7.7k indexed |
| `kenwatches` | HTML (Next.js), `/watch?page=N` with the list in `__NEXT_DATA__` (20 per page) | HKD | ✅ 20/20 items with brand, reference, movement, gender, diameter, year, box/papers, photos; price from `cash` with `listedPrice` as fallback, both null = price on request | Ken's Watches 名錶廊, Hong Kong (Wan Chai / Central / Causeway Bay / TST shops); ~740 pieces / 38 pages |
| `rasin` | HTML (Shopserve), `/SHOP/list.php?Search=&Type=01&PAGE=N` (40 per page, default order) | JPY | ✅ 40/40 cards with price, brand + reference + grade + gender from the title, image | GINZA RASIN, Tokyo/Osaka; ~27k published pieces of which the in-stock ones come first and the sold ones follow at ¥0, so the crawl stops at the first page without a priced card (do not use `SortKey=new`, it lists sold pieces first) |
| `quark` | HTML, stock search `/rolex_search/result.html?…&p=N&sort=NEW&snum=100` (zero-based, 100 per page) | JPY | ✅ 100/100 items with price (3-year-warranty FAIR price; the 5-year one is kept in attributes), reference, category 新品/中古品, year, stock shop | ロレックス専門店クォーク (Quark), Rolex only, ~2.4k pieces; the origin answers 502 now and then, which the Fetcher retries |
| `bellemonde` | HTML (MakeShop), `/view/category/all_items?sort=order&page=N` (50 per page) | JPY | ✅ 34 in-stock of 50 cards; price, brand + reference from the title, 【美品】【中古】 markers → condition, 保証書/箱 → papers/box, image | 腕時計専門店ベルモンド, Tokyo; ~4.5k published but almost all sold, so the crawl stops after two pages without an in-stock piece |
| `kitamura` | HTML (Nuxt SSR), `/watch/buy/search/?page=N&sort=used_update_dt&pageSize=90` (90 per page, `.el-pager` numbers) | JPY | ✅ 90/90 cards with price, brand, 型番 reference, 状態 grade (AA/A/AB/B), shop, image | カメラのキタムラ's pre-owned watch counters; ~360 pieces / 4 pages; blocked from Google Cloud, so pages go through Browserless when `CRAWL_RENDER_SERVICE_URL` is set |
| `watchandtime` | HTML (ASP, Big5), brand categories `products_list.asp?id=N&page=P` from `index.asp` (24 per page) plus one product page per item | TWD | ✅ 24/24 cards with price and image; the product page adds the description title, reference, material, movement, gender, year, photos | 黃忠政名錶交流中心 (James Huang Vintage Watches), Taiwan; cards carry only brand + stock code, so every product page is fetched (a few hundred requests a night); the 鑽石／配件 category is skipped |
| `jdpawn` | HTML (ASP.NET MVC), `/luxury/Stuffs/Watch?page=N` (16 per page, "共13頁" in the pager); each piece is a quick-view modal on the list page | TWD | ✅ 16/16 with price, brand, 型式 reference/model, 機芯 movement, 錶徑 diameter, 附註 box/papers, store city; photo from `/luxury/StuffPictures/<code>.jpg` | 久大御典品, a Taichung pawn-shop group; ~200 pieces; pages weigh 4–5 MB because photos are inlined as base64, so the Fetcher body limit is 16 MB |
| `bbc0804` | HTML (iyp.tw static site), watch categories from the `/goods.html` nav (one page each, no pagination) plus one product page per piece | TWD | ✅ 27/27 cards with price ("$64.8萬"), brand, 型號 reference, 錶徑, papers year from the title; product page adds box/papers, material, movement, condition note, photos | 仁川當舖, a pawn shop in Rende, Tainan; ~80 pieces; jewellery, gold, scooter, phone and bag categories are skipped |
| `watchexchange` | **Reddit JSON API**, `/r/Watchexchange/new` (100 posts per page, `after` cursor, ≤10 pages: Reddit stops a listing at ~1000) | per post (USD default; C$, A$, €, £, CAD/EUR/… recognised) | ⚠️ fixture only: reddit.com is unreachable from the dev sandbox; optional `REDDIT_CLIENT_ID/SECRET`; blocked from Google Cloud: set `REDDIT_CLIENT_ID/SECRET` (OAuth works from cloud IPs); without them the public JSON goes through Browserless when `CRAWL_RENDER_SERVICE_URL` is set, but Browserless also gets 403 (2026-10-20) | peer-to-peer marketplace of private sellers; only WTS / WTS/WTT posts (flair, else `[WTS]` in the title), skipping WTB, Sold, removed and stickied posts; price from the title, else the body's asking line; photos from Reddit galleries / previews, imgur albums kept in `attributes.album`; `CONUS` / "located in US" sets US, otherwise location stays empty |
| `ebay` | **Browse API** (OAuth client credentials) | market currency | needs `EBAY_CLIENT_ID/SECRET` | per-brand aspect filter in Wristwatches (31387); free developer keys at developer.ebay.com |

Image URLs: Jackroad thumbnails (`/img/goods/S/<id>.jpg`) are upgraded to the product-page image
(`/img/goods/1/<id>.jpg`); Watchnian only serves the small `/S/` variant for the main image;
Hourstack links `product/product_big/*.JPG` directly. We never download or store images.

## Sources that need configuration

`ebay` implements `crawler.Preflighter`. When its credentials are missing
the Runner logs a warning, writes a `skipped` row to `crawl_runs` (visible at `/api/v1/crawls`) and
carries on with the other sources; the job still exits 0. Silence the warning with
`CRAWL_SOURCES=<list>` or `UPDATE sources SET enabled = false WHERE key = '…'`.

## Japanese prices

`jackroad`, `watchnian`, `commitwatch`, `lips`, `allu`, `ishida`, `rasin`, `quark`, `bellemonde` and `kitamura` all list tax-included prices; `sevenhours` prints its own 税抜 figure, which the adapter parses. The Runner fills
`PriceExclTax` for any source whose country is `JP` (see `crawler.ApplyTaxFreePrice`), and
`crawler.ComparisonPrice` decides what gets converted to USD. A new Japanese source therefore needs
no tax handling of its own. If a site publishes its own 税抜 figure, parse it into `PriceExclTax`
and the derivation will leave it alone.

## Box and papers

`normalize.DetectBoxPapers` reads free text (titles, prose descriptions) in English, Traditional and
Simplified Chinese and Japanese. It returns `nil` when the text says nothing, and explicit
negatives win over the positive words they contain. Watch-only wording means no box and no papers:
淨錶 / 凈錶 (Hong Kong dealers write both), 裸錶 (Taiwanese pawn shops), a standalone 單錶, 本体のみ,
付属品なし, "watch only". "一錶一紙" style phrases mean the watch plus exactly one accessory.

`normalize.DetectAccessories` is for a **structured accessories field** that lists everything
included, so whatever it does not mention is absent rather than unknown: RWW `配件` ("Full Set,
全齊", "淨錶", "一錶一紙"), Lips `付属` ("箱 保証書(2026.08)", "箱", "-"), 仁川當舖 `附件` ("原盒1 保單1
說明書2") and JD Pawn `附註`. An empty field stays unknown. Prose fields such as Watch & Time's
附件狀況 keep the free-text detector.

## Images

The Runner passes every listing's images through `crawler.PhotoURLs`, which drops SVGs and data
URIs: marketplaces decorate cards with badges, and an `<img>` sweep picks them up ahead of the
photo. Chrono24's "Certified" seal (`s.c24.media/images/default/certified/certified-filled.svg`)
is additionally rejected in its own parser together with everything under `/images/default/`.

## Non-watch items

Dealers list straps, buckles and bracelet links under their watch brands. `normalize.NonWatchBrands`
(currently Vagenari, a strap maker) and `normalize.NonWatchTitleMarkers` (bracelet links 錶節,
straps sold alone 原廠錶帶 / 副廠錶帶, boxes 錶盒 and travel cases / pouches) name what is dropped:
the Runner skips any listing whose title or model matches and counts it as `excluded` in the run
summary, and the alapower adapters skip a whole brand category whose menu entry matches. Add to
those two lists when a new part maker or part type shows up.

Taiwanese dealers use the same words when describing a whole watch ("錶節齊全", "少一節錶節",
"搭配Vagenari橡膠錶帶", "附原廠錶盒"), so a match does not count when the title also reads as a
watch (錶徑, 上鍊, 機芯, 保單/保卡, 面盤, 腕錶, a case diameter of 28mm or more, …) or when it sits
in a "comes with" phrase after 搭配 / 附 / 贈 / 送. A bare 錶帶 is not a marker: it usually
describes the watch's own strap ("快拆式錶帶", "不鏽鋼橡膠錶帶").

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
exposes `/products.json`; read the feed as `crawler/commitwatch` did before 2026-10 (see git history) and adjust the title/condition parsing.

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

## Reddit setup (r/Watchexchange)

Works without configuration through the public `www.reddit.com/r/Watchexchange/new.json`, but
Reddit throttles unauthenticated clients and often answers 403 to cloud IP ranges (Cloud Run). For
production, use the OAuth API:

1. Create a *script* app at https://www.reddit.com/prefs/apps (any redirect URI).
2. Set `REDDIT_CLIENT_ID` (the string under the app name) and `REDDIT_CLIENT_SECRET` on the crawler
   job, e.g. via Secret Manager and `--set-secrets` in `backend/cloudbuild.yaml`.

The adapter then gets an app-only token (client-credentials grant) and reads
`oauth.reddit.com/r/Watchexchange/new`; 10 requests a night are far below Reddit's free-tier limit.

## eBay setup

1. Create an app at https://developer.ebay.com → production keyset.
2. Store the Client ID / Secret in Secret Manager (`EBAY_CLIENT_ID`, `EBAY_CLIENT_SECRET`).
3. Optionally set `EBAY_MARKETPLACES=EBAY_US,EBAY_GB,EBAY_DE` (currency filter follows the market).

The default Browse API quota (5,000 calls/day) covers 25 brands × 1 marketplace × up to 50 pages;
lower `CRAWL_MAX_PAGES` for the eBay job or trim `ebay.Brands` if you add marketplaces.
