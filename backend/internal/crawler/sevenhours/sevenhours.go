// Package sevenhours crawls https://7hours.jp (セブンアワーズ, Kobe; JPY).
//
// Site structure (verified 2026-09): Shopserve shop, UTF-8. The stock is small (a few hundred
// pieces, mostly new and unused), so the whole catalogue is read from the empty search:
//   - All items: /SHOP/list.php?Search=&Type=01&PAGE=<n>  (40 cards per page; a <link rel="next">
//     is present while more pages exist)
//   - Cards: section.column4 with p.itemThumb a[href="/SHOP/<id>.html"] img (llimg = large),
//     h2 a[title="【<id>】 BRAND ブランド <ref> <model> …"], p.price .selling_price (税込) and
//     .taxin "(税抜 ¥…)", span.badge img icon_new.png
package sevenhours

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

const baseURL = "https://7hours.jp"

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "sevenhours" }

var (
	idRe          = regexp.MustCompile(`/SHOP/(\d+)\.html`)
	titlePrefixRe = regexp.MustCompile(`^【\d+】\s*`)
	taxExclRe     = regexp.MustCompile(`税抜\s*[¥￥]?\s*([0-9][0-9,]*)`)
)

// Crawl pages through the all-items search until the last page.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	seen := map[string]bool{}
	for page := 1; page <= env.MaxPages; page++ {
		u := fmt.Sprintf("%s/SHOP/list.php?Search=&Type=01&PAGE=%d", baseURL, page)
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
	doc.Find("section.column4").Each(func(_ int, card *goquery.Selection) {
		a := card.Find(`a[href*="/SHOP/"]`).First()
		href, _ := a.Attr("href")
		m := idRe.FindStringSubmatch(href)
		if m == nil || seen[m[1]] {
			return
		}
		h2 := card.Find("h2 a").First()
		title, _ := h2.Attr("title")
		if title == "" {
			title = crawler.Text(h2)
		}
		title = normalize.CleanText(titlePrefixRe.ReplaceAllString(title, ""))
		if title == "" {
			return
		}
		seen[m[1]] = true
		l := model.Listing{
			ExternalID:      m[1],
			URL:             crawler.AbsURL(doc, href),
			Title:           title,
			Currency:        "JPY",
			SellerType:      "dealer",
			SellerName:      "7hours",
			LocationCountry: "JP",
			LocationCity:    "Kobe",
		}
		if p, ok := normalize.ParsePrice(crawler.Text(card.Find(".selling_price").First())); ok {
			l.Price = &p
		}
		if tm := taxExclRe.FindStringSubmatch(crawler.Text(card.Find(".taxin").First())); tm != nil {
			if p, ok := normalize.ParsePrice(tm[1]); ok {
				l.PriceExclTax = &p
			}
		}
		if src, ok := card.Find("p.itemThumb img").First().Attr("src"); ok && src != "" {
			l.ImageURLs = []string{crawler.AbsURL(doc, src)}
		}
		// The shop sells new and unused pieces; the title states the grade only when it deviates.
		switch {
		case strings.Contains(title, "未使用"):
			l.Condition = model.ConditionUnworn
		case strings.Contains(title, "中古"):
			l.Condition = model.ConditionGood
		case strings.Contains(title, "新品"):
			l.Condition = model.ConditionNew
		}
		attrs := map[string]any{}
		card.Find("span.badge img").Each(func(_ int, img *goquery.Selection) {
			if src, _ := img.Attr("src"); strings.Contains(src, "icon_new") {
				attrs["badge"] = "new-arrival"
			}
		})
		l.Attributes, _ = json.Marshal(attrs)
		out = append(out, l)
	})
	return out
}
