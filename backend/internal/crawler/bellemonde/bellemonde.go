// Package bellemonde crawls https://bellemonde.tokyo (腕時計専門店ベルモンド, Tokyo; JPY).
//
// Site structure (verified 2026-09): MakeShop, UTF-8.
//   - All items, newest first: /view/category/all_items?sort=order&page=<n> (50 per page,
//     "全4451件" in .bm_item_all_num). Sold pieces stay listed with .bm_item_icon_soldout and a
//     0円 price; only the first pages hold pieces in stock, so the crawl stops after two
//     consecutive pages without one.
//   - Cards: .bm_item_lists li with a[href="/view/item/<id>?category_page_id=all_items"], img,
//     .bm_item_lists_title a "【美品】【中古】IWC パイロット ウォッチ クロノグラフ 41 IW388101 保証書(…)"
//     (grade markers, brand in English and Japanese, model, reference, accessories),
//     .bm_price "220,000円（税込）", img[src*="/shopimages/bellemonde/icon<X>.gif"] flags
package bellemonde

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

const (
	baseURL  = "https://bellemonde.tokyo"
	pageSize = 50
)

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "bellemonde" }

var (
	idRe     = regexp.MustCompile(`/view/item/(\d+)`)
	iconRe   = regexp.MustCompile(`/icon([A-Za-z]+)\.gif`)
	markerRe = regexp.MustCompile(`【[^】]*】\s*`)
	totalRe  = regexp.MustCompile(`全\s*([0-9,]+)\s*件`)
)

// Crawl pages newest first until two consecutive pages hold nothing in stock.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	seen := map[string]bool{}
	empty := 0
	for page := 1; page <= env.MaxPages; page++ {
		u := fmt.Sprintf("%s/view/category/all_items?sort=order&page=%d", baseURL, page)
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
		if fresh == 0 {
			empty++
		} else {
			empty = 0
		}
		if empty >= 2 || total < pageSize {
			return nil
		}
	}
	return nil
}

// ParseList extracts the in-stock listings from a category page and the number of cards on it,
// sold ones included. Exported for fixture tests.
func ParseList(doc *goquery.Document) ([]model.Listing, int) {
	var out []model.Listing
	seen := map[string]bool{}
	total := 0
	doc.Find(".bm_item_lists li").Each(func(_ int, li *goquery.Selection) {
		a := li.Find(`a[href*="/view/item/"]`).First()
		href, _ := a.Attr("href")
		m := idRe.FindStringSubmatch(href)
		if m == nil || seen[m[1]] {
			return
		}
		total++
		if li.Find(".bm_item_icon_soldout, .bm_item_cart_soldout").Length() > 0 {
			return
		}
		raw := crawler.Text(li.Find(".bm_item_lists_title a").First())
		if raw == "" {
			return
		}
		price, ok := normalize.ParsePrice(crawler.Text(li.Find(".bm_price").First()))
		if !ok {
			return
		}
		seen[m[1]] = true
		markers := markerRe.FindAllString(raw, -1)
		title := normalize.CleanText(markerRe.ReplaceAllString(raw, ""))
		l := model.Listing{
			ExternalID:      m[1],
			URL:             baseURL + "/view/item/" + m[1],
			Title:           title,
			Price:           &price,
			Currency:        "JPY",
			SellerType:      "dealer",
			SellerName:      "Belle Monde",
			LocationCountry: "JP",
			LocationCity:    "Tokyo",
			Condition:       condition(strings.Join(markers, "")),
		}
		if src, ok := li.Find(".bm_item_lists_img img").First().Attr("src"); ok && src != "" {
			l.ImageURLs = []string{crawler.AbsURL(doc, src)}
		}
		l.HasBox, l.HasPapers = normalize.DetectBoxPapers(raw)
		attrs := map[string]any{}
		if len(markers) > 0 {
			attrs["markers"] = strings.Join(markers, "")
		}
		var icons []string
		li.Find(".bm_item_lists_title img").Each(func(_ int, img *goquery.Selection) {
			if src, _ := img.Attr("src"); src != "" {
				if im := iconRe.FindStringSubmatch(src); im != nil {
					icons = append(icons, im[1])
				}
			}
		})
		if len(icons) > 0 {
			attrs["icons"] = icons
		}
		l.Attributes, _ = json.Marshal(attrs)
		out = append(out, l)
	})
	return out, total
}

// Total returns the catalogue size printed on the page, or 0.
func Total(doc *goquery.Document) int {
	if m := totalRe.FindStringSubmatch(crawler.Text(doc.Find(".bm_item_all_num").First())); m != nil {
		n := 0
		for _, r := range m[1] {
			if r >= '0' && r <= '9' {
				n = n*10 + int(r-'0')
			}
		}
		return n
	}
	return 0
}

// condition maps the 【…】 markers at the start of a title onto the scale.
func condition(markers string) model.Condition {
	switch {
	case strings.Contains(markers, "未使用"):
		return model.ConditionUnworn
	case strings.Contains(markers, "新品"):
		return model.ConditionNew
	case strings.Contains(markers, "極美品"):
		return model.ConditionVeryGood
	case strings.Contains(markers, "美品"):
		return model.ConditionVeryGood
	case strings.Contains(markers, "中古"):
		return model.ConditionGood
	}
	return model.ConditionUnknown
}
