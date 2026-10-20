// Package commitwatch indexes https://commit-watch.co.jp (Commit Ginza, Tokyo; JPY).
//
// The shop runs on Shopify, but its /products.json feed is useless here: it is ordered oldest
// first, ignores availability filters, and Shopify stops it at page*limit = 25000 while the shop
// keeps over 27000 sold pieces (collection "soldout"), so current stock sits past the cap.
//
// Instead we read the storefront collection page newest first (verified 2026-10):
//   - GET /collections/all?filter.v.availability=1&sort_by=created-descending&page=N
//     (48 cards per page; the availability filter is ignored too, but the sort works)
//   - Cards: .product-item with p.brand_name (Japanese), p.model_name, span.dotadd "型番：<ref>",
//     .condition_for_list (未使用 / 中古 / ヴィンテージ), img alt = full title, and either
//     .taxinprice .price "¥15,510,000" + .notax "¥14,100,000（税抜）" or a status label
//     .lable_status_price.soldout / .reserved. In-stock cards carry the Shopify product id in the
//     cart form (input[name="product-id"]), which is the external id.
//
// Sold pieces stay in the list, so the crawl stops after soldOutRun consecutive pages without
// a single piece for sale.
package commitwatch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

const (
	baseURL = "https://commit-watch.co.jp"
	// Shopify caps collection paging at 25000 products as well (48 per page → 521 pages).
	maxPage    = 25000 / 48
	soldOutRun = 10
)

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "commitwatch" }

// Crawl walks the collection newest first until soldOutRun pages in a row have nothing for sale.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	seen := map[string]bool{}
	empty := 0
	for page := 1; page <= min(env.MaxPages, maxPage); page++ {
		u := fmt.Sprintf("%s/collections/all?filter.v.availability=1&sort_by=created-descending&page=%d", baseURL, page)
		doc, err := env.Fetcher.Doc(ctx, u)
		if err != nil {
			return fmt.Errorf("page %d: %w", page, err)
		}
		cards, items := ParseList(doc)
		if cards == 0 {
			if page == 1 {
				return fmt.Errorf("no products on first page (markup changed?)")
			}
			return nil
		}
		fresh := 0
		for _, l := range items {
			if seen[l.ExternalID] {
				continue
			}
			seen[l.ExternalID] = true
			fresh++
			if err := emit(l); err != nil {
				return err
			}
		}
		if fresh == 0 {
			if empty++; empty >= soldOutRun {
				return nil
			}
		} else {
			empty = 0
		}
	}
	return nil
}

// ParseList returns the number of product cards on a page and the listings for sale among them.
// Exported for fixture tests.
func ParseList(doc *goquery.Document) (cards int, out []model.Listing) {
	doc.Find(".product-item").Each(func(_ int, item *goquery.Selection) {
		a := item.Find(`a.product-item__title[href*="/products/"]`).First()
		href, ok := a.Attr("href")
		if !ok {
			return
		}
		cards++
		if item.Find(".lable_status_price").Length() > 0 { // SOLD OUT / RESERVED
			return
		}
		price, ok := normalize.ParsePrice(crawler.Text(item.Find(".taxinprice .price").First()))
		if !ok || price <= 0 {
			return
		}
		handle := strings.TrimPrefix(href, "/products/")
		id, _ := item.Find(`input[name="product-id"]`).Attr("value")
		if id == "" {
			id = handle
		}
		img := item.Find("img.product-item__primary-image").First()
		title, _ := img.Attr("alt")
		title = normalize.CleanText(strings.TrimSuffix(strings.TrimSpace(title), " "+handle))
		brand := crawler.Text(item.Find(".brand_name").First())
		modelName := crawler.Text(item.Find(".model_name").First())
		if title == "" {
			title = normalize.CleanText(brand + " " + modelName)
		}
		l := model.Listing{
			ExternalID:      id,
			URL:             baseURL + "/products/" + handle,
			Title:           title,
			Price:           &price,
			Currency:        "JPY",
			SellerType:      "dealer",
			SellerName:      "Commit Ginza",
			LocationCountry: "JP",
			LocationCity:    "Tokyo",
			Condition:       model.ConditionUnknown,
		}
		if excl, ok := normalize.ParsePrice(crawler.Text(item.Find(".notax .metafield-money").First())); ok && excl > 0 {
			l.PriceExclTax = &excl
		}
		if b := normalize.DetectBrand(brand, title); b != nil {
			l.BrandSlug, l.BrandName = b.Slug, b.Name
		}
		item.Find("span.dotadd").Each(func(_ int, sp *goquery.Selection) {
			if t := crawler.Text(sp); strings.HasPrefix(t, "型番") {
				ref := strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(t, "型番"), "：: "))
				if f := strings.Fields(ref); len(f) > 0 {
					l.ReferenceNumber = strings.ToUpper(f[0])
				}
			}
		})
		grade := crawler.Text(item.Find(".condition_for_list").First())
		switch {
		case strings.Contains(grade, "未使用"):
			l.Condition = model.ConditionUnworn
		case strings.Contains(grade, "新品"):
			l.Condition = model.ConditionNew
		case strings.Contains(grade, "中古"), strings.Contains(grade, "ヴィンテージ"):
			l.Condition = model.ConditionGood
		}
		if src, ok := img.Attr("src"); ok && src != "" {
			l.ImageURLs = append(l.ImageURLs, crawler.AbsURL(doc, src))
		}
		if src, ok := item.Find("img.product-item__secondary-image").First().Attr("src"); ok && src != "" {
			l.ImageURLs = append(l.ImageURLs, crawler.AbsURL(doc, src))
		}
		attrs := map[string]any{"handle": handle, "grade": grade, "model": modelName}
		if notes := crawler.Text(item.Find(".individual_characteristics").First()); strings.TrimSpace(notes) != "" {
			attrs["notes"] = notes
		}
		l.Attributes, _ = json.Marshal(attrs)
		out = append(out, l)
	})
	return cards, out
}
