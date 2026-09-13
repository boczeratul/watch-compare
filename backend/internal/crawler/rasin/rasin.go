// Package rasin crawls https://www.rasin.co.jp (GINZA RASIN, Tokyo and Osaka; JPY).
//
// Site structure (verified 2026-09): Shopserve shop, UTF-8, same theme as 7hours.
//   - All items: /SHOP/list.php?Search=&Type=01&PAGE=<n> (40 cards per page). The default order
//     lists pieces in stock first and then ~25k sold ones, which stay published with a ¥0 price
//     and "在庫切れ" on their page; the crawl stops at the first page without a priced card.
//     (SortKey=new is not used: it surfaces the sold pieces first.)
//   - Cards: section.column4 with p.itemThumb a[href="/SHOP/<code>.html"] img (llimg = large),
//     h2 a[title="ロレックス デイトジャスト 116200 ブラック/バー ランダムシリアル 中古 メンズ"]
//     (brand, model, reference, dial, serial range, grade 中古/新品/未使用, gender), .selling_price
//     (税込), span.badge img icon_new.png for new arrivals
package rasin

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

const baseURL = "https://www.rasin.co.jp"

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "rasin" }

var idRe = regexp.MustCompile(`/SHOP/([A-Za-z0-9_-]+)\.html`)

// Crawl pages through the list until the in-stock pieces run out.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	seen := map[string]bool{}
	for page := 1; page <= env.MaxPages; page++ {
		u := fmt.Sprintf("%s/SHOP/list.php?Search=&Type=01&PAGE=%d", baseURL, page)
		doc, err := env.Fetcher.Doc(ctx, u)
		if err != nil {
			return fmt.Errorf("page %d: %w", page, err)
		}
		items, total := ParseList(doc)
		if total == 0 {
			if page == 1 {
				return fmt.Errorf("no cards on first page (markup changed?)")
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
		// a page made only of sold (¥0) pieces means the in-stock part of the list is over
		if fresh == 0 || len(items) == 0 || !HasNext(doc) {
			return nil
		}
	}
	return nil
}

// HasNext reports whether the page links to a following page.
func HasNext(doc *goquery.Document) bool {
	return doc.Find(`link[rel="next"]`).Length() > 0
}

// ParseList extracts the in-stock listings from a list page and the number of cards on it,
// sold cards (price ¥0) included. Exported for fixture tests.
func ParseList(doc *goquery.Document) ([]model.Listing, int) {
	var out []model.Listing
	seen := map[string]bool{}
	total := 0
	doc.Find("section.column4").Each(func(_ int, card *goquery.Selection) {
		a := card.Find(`a[href*="/SHOP/"]`).First()
		href, _ := a.Attr("href")
		m := idRe.FindStringSubmatch(href)
		if m == nil || seen[m[1]] {
			return
		}
		total++
		h2 := card.Find("h2 a").First()
		title, _ := h2.Attr("title")
		if title == "" {
			title = crawler.Text(h2)
		}
		title = normalize.CleanText(title)
		if title == "" {
			return
		}
		price, ok := normalize.ParsePrice(crawler.Text(card.Find(".selling_price").First()))
		if !ok {
			return // ¥0: sold, the page says 在庫切れ
		}
		seen[m[1]] = true
		l := model.Listing{
			ExternalID:      m[1],
			URL:             crawler.AbsURL(doc, href),
			Title:           title,
			Price:           &price,
			Currency:        "JPY",
			SellerType:      "dealer",
			SellerName:      "GINZA RASIN",
			LocationCountry: "JP",
			LocationCity:    "Tokyo",
			Condition:       condition(title),
		}
		if src, ok := card.Find("p.itemThumb img").First().Attr("src"); ok && src != "" {
			l.ImageURLs = []string{crawler.AbsURL(doc, src)}
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
	return out, total
}

// condition reads the grade word Rasin puts near the end of every title.
func condition(title string) model.Condition {
	switch {
	case strings.Contains(title, "未使用"):
		return model.ConditionUnworn
	case strings.Contains(title, "新品"):
		return model.ConditionNew
	case strings.Contains(title, "アンティーク"):
		return model.ConditionFair
	case strings.Contains(title, "中古"):
		return model.ConditionGood
	}
	return model.ConditionUnknown
}
