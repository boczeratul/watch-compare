// Package watchfinderhk crawls https://www.watchfinder.hk (Watchfinder & Co., Hong Kong; HKD).
//
// Site structure (verified 2026-09): Magento 2 storefront.
//   - List: /watches?p=<n> (42 cards per page, ~5.5k pieces / 131 pages, <link rel="next"> while
//     more exist; the whole catalogue fits in the default CRAWL_MAX_PAGES)
//   - Cards: a.product-card[data-product-sku][data-product-brand][data-product-series]
//     [data-product-model][data-product-image] with .product-card__id__{brand,series,model-number},
//     .product-card__specs__box-papers__item "Box <span class=icon-yes|icon-no>",
//     .product-card__specs__year-location__item__value (year),
//     [data-price-type="finalPrice"][data-price-amount] (HKD), .product-card__usp-badges__item
//   - robots.txt disallows filtered URLs (?filter*); plain pagination is allowed.
package watchfinderhk

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

const baseURL = "https://www.watchfinder.hk"

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "watchfinderhk" }

// Crawl pages through the full watch list.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	seen := map[string]bool{}
	for page := 1; page <= env.MaxPages; page++ {
		u := baseURL + "/watches"
		if page > 1 {
			u = fmt.Sprintf("%s/watches?p=%d", baseURL, page)
		}
		doc, err := env.Fetcher.Doc(ctx, u)
		if err != nil {
			return fmt.Errorf("page %d: %w", page, err)
		}
		items := ParseList(doc)
		if len(items) == 0 {
			if page == 1 {
				return fmt.Errorf("no items on first page (markup changed?)")
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
		if fresh == 0 || !HasNext(doc) {
			return nil
		}
	}
	return nil
}

// HasNext reports whether the page links to a following page.
func HasNext(doc *goquery.Document) bool {
	return doc.Find(`link[rel="next"]`).Length() > 0
}

// ParseList extracts listings from a list page. Exported for fixture tests.
func ParseList(doc *goquery.Document) []model.Listing {
	var out []model.Listing
	seen := map[string]bool{}
	doc.Find("a.product-card[data-product-sku]").Each(func(_ int, a *goquery.Selection) {
		sku, _ := a.Attr("data-product-sku")
		href, _ := a.Attr("href")
		if sku == "" || href == "" || seen[sku] {
			return
		}
		brand := attrOrText(a, "data-product-brand", ".product-card__id__brand")
		series := attrOrText(a, "data-product-series", ".product-card__id__series")
		ref := attrOrText(a, "data-product-model", ".product-card__id__model-number")
		title := normalize.CleanText(strings.Join([]string{brand, series, ref}, " "))
		if title == "" {
			return
		}
		seen[sku] = true
		l := model.Listing{
			ExternalID:      sku,
			URL:             crawler.AbsURL(doc, href),
			Title:           title,
			Model:           series,
			ReferenceNumber: strings.ToUpper(ref),
			Currency:        "HKD",
			Condition:       model.ConditionUnknown, // pre-owned specialist; the grade is only on the product page
			SellerType:      "dealer",
			SellerName:      "Watchfinder & Co.",
			LocationCountry: "HK",
			LocationCity:    "Hong Kong",
		}
		if b := normalize.DetectBrand(brand, title); b != nil {
			l.BrandSlug, l.BrandName = b.Slug, b.Name
		}
		if amt, ok := a.Find(`[data-price-type="finalPrice"]`).First().Attr("data-price-amount"); ok {
			if v, err := strconv.ParseFloat(amt, 64); err == nil && v > 0 {
				l.Price = &v
			}
		}
		if l.Price == nil {
			if p, ok := normalize.ParsePrice(crawler.Text(a.Find(".price").First())); ok {
				l.Price = &p
			}
		}
		if img, ok := a.Attr("data-product-image"); ok && img != "" {
			l.ImageURLs = []string{largeImage(img)}
		} else if src, ok := a.Find("img").First().Attr("src"); ok && src != "" {
			l.ImageURLs = []string{largeImage(src)}
		}
		a.Find(".product-card__specs__box-papers__item").Each(func(_ int, it *goquery.Selection) {
			yes := it.Find(".icon-yes").Length() > 0
			no := it.Find(".icon-no").Length() > 0
			if !yes && !no {
				return
			}
			v := yes
			switch label := strings.ToLower(crawler.Text(it)); {
			case strings.Contains(label, "box"):
				l.HasBox = &v
			case strings.Contains(label, "paper"):
				l.HasPapers = &v
			}
		})
		if y := crawler.Text(a.Find(".product-card__specs__year-location__item__value").First()); y != "" {
			l.Year = normalize.ParseYear(y)
		}
		attrs := map[string]any{}
		var badges []string
		a.Find(".product-card__usp-badges__item").Each(func(_ int, b *goquery.Selection) {
			if t := crawler.Text(b); t != "" {
				badges = append(badges, t)
			}
		})
		if len(badges) > 0 {
			attrs["badges"] = badges
		}
		if id, ok := a.Attr("data-product-model-id"); ok {
			attrs["modelId"] = id
		}
		l.Attributes, _ = json.Marshal(attrs)
		out = append(out, l)
	})
	return out
}

func attrOrText(a *goquery.Selection, attr, selector string) string {
	if v, ok := a.Attr(attr); ok && strings.TrimSpace(v) != "" {
		return normalize.CleanText(v)
	}
	return crawler.Text(a.Find(selector).First())
}

// largeImage drops the thumbnail crop parameters so the CDN returns the full photo.
func largeImage(u string) string {
	if i := strings.Index(u, "?"); i > 0 {
		return u[:i]
	}
	return u
}
