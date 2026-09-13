// Package kitamura crawls the pre-owned watch shop of https://shop.kitamura.jp (カメラのキタムラ,
// a Japanese camera retailer with watch counters; JPY).
//
// Site structure (verified 2026-09): Nuxt with server-side rendering, so the list is plain HTML.
//   - List, newest first: /watch/buy/search/?page=<n>&sort=used_update_dt&pageSize=90
//     (90 is the largest page size offered; ~360 pieces / 4 pages; .el-pager li.number holds the
//     page numbers)
//   - Cards: .data-item with a[href="/watch/buy/item/<id>/"] img, a.title
//     "ロレックス(ROLEX) エクスプローラーII ブラック 16570 ステンレススティール", .detail .type lines
//     "型番：16570", "状態：A" (AA / A / AB / B dealer grades), "取扱店舗：東京・中野サンモール店",
//     .price "1,348,000 円(税込)"
package kitamura

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

const (
	baseURL  = "https://shop.kitamura.jp"
	pageSize = 90
)

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "kitamura" }

var idRe = regexp.MustCompile(`/watch/buy/item/(\d+)/`)

// Crawl pages through the list until the last page.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	seen := map[string]bool{}
	for page := 1; page <= env.MaxPages; page++ {
		u := fmt.Sprintf("%s/watch/buy/search/?page=%d&sort=used_update_dt&pageSize=%d", baseURL, page, pageSize)
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
		if fresh == 0 || page >= LastPage(doc) {
			return nil
		}
	}
	return nil
}

// LastPage returns the highest page number in the pager, or 1.
func LastPage(doc *goquery.Document) int {
	last := 1
	doc.Find(".el-pager li.number").Each(func(_ int, li *goquery.Selection) {
		if n, err := strconv.Atoi(crawler.Text(li)); err == nil && n > last {
			last = n
		}
	})
	return last
}

// ParseList extracts listings from a search page. Exported for fixture tests.
func ParseList(doc *goquery.Document) []model.Listing {
	var out []model.Listing
	seen := map[string]bool{}
	doc.Find(".data-item").Each(func(_ int, item *goquery.Selection) {
		a := item.Find(`a[href*="/watch/buy/item/"]`).First()
		href, _ := a.Attr("href")
		m := idRe.FindStringSubmatch(href)
		if m == nil || seen[m[1]] {
			return
		}
		title := normalize.CleanText(crawler.Text(item.Find("a.title").First()))
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
			SellerName:      "Kitamura",
			LocationCountry: "JP",
			Condition:       model.ConditionGood, // pre-owned shop; the grade below refines it
		}
		if b := normalize.DetectBrand(title); b != nil {
			l.BrandSlug, l.BrandName = b.Slug, b.Name
		}
		attrs := map[string]any{}
		item.Find(".detail .type").Each(func(_ int, d *goquery.Selection) {
			t := crawler.Text(d)
			switch {
			case strings.HasPrefix(t, "型番"):
				if ref := strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(t, "型番"), "：: ")); ref != "" {
					l.ReferenceNumber = strings.ToUpper(ref)
				}
			case strings.HasPrefix(t, "状態"):
				grade := strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(t, "状態"), "：: "))
				attrs["grade"] = grade
				l.Condition = gradeToCondition(grade)
			case strings.HasPrefix(t, "取扱店舗"):
				shop := strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(t, "取扱店舗"), "：: "))
				attrs["shop"] = shop
				// "東京・中野サンモール店" → the prefecture / city before the interpunct
				if i := strings.Index(shop, "・"); i > 0 {
					l.LocationCity = shop[:i]
				}
			}
		})
		if p, ok := normalize.ParsePrice(crawler.Text(item.Find(".price").First())); ok {
			l.Price = &p
		}
		if src, ok := item.Find(".watch-img img").First().Attr("src"); ok && src != "" {
			l.ImageURLs = []string{crawler.AbsURL(doc, src)}
		}
		l.Attributes, _ = json.Marshal(attrs)
		out = append(out, l)
	})
	return out
}

// gradeToCondition maps Kitamura's grades (AA best, then A, AB, B, C) onto the scale.
func gradeToCondition(g string) model.Condition {
	switch strings.ToUpper(strings.TrimSpace(g)) {
	case "N", "S":
		return model.ConditionNew
	case "AA", "SA":
		return model.ConditionVeryGood
	case "A", "AB":
		return model.ConditionGood
	case "B", "BC":
		return model.ConditionFair
	case "C", "D":
		return model.ConditionPoor
	}
	return model.ConditionGood
}
