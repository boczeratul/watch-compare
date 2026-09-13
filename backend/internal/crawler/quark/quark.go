// Package quark crawls https://www.909.co.jp (ロレックス専門店クォーク / Quark, a Rolex specialist
// with 18 shops across Japan; JPY).
//
// Site structure (verified 2026-09): custom PHP site. The stock search lists everything:
//   - /rolex_search/result.html?btn_val=kensaku_s&freeword_s=&p=<n>&sort=NEW&snum=100
//     (p is zero-based, 100 items per page, ~2.4k items; "#serch_count .num" is the total; sold
//     pieces linger with "SOLD OUT" in place of the price and are skipped;
//     snum above 100 answers 502, and the origin sometimes answers 502 on its own, which the
//     Fetcher retries)
//   - Items: a.list_item[href="../rolex_used/model/<model>_<ref>_<no>/" | "../rolex_catalog/…html"]
//     with .ttl_cat .cate (新品 / 中古品 / アンティーク), .list_spec1 "ロレックス<br>デイトナ<br>
//     Ref.116595RBOW<br>パヴェダイヤ", .list_spec2 "年式：2018年製<br>シリアル：…<br>商品番号：230621<br>
//     在庫店舗：デイトナサロン", .list_3y (3-year-warranty price, the listed one) and .list_5y
//     (5-year-warranty price or "－"), img (relative)
package quark

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
	baseURL  = "https://www.909.co.jp"
	pageSize = 100
)

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "quark" }

var (
	brRe    = regexp.MustCompile(`(?i)<br\s*/?>`)
	tagRe   = regexp.MustCompile(`<[^>]+>`)
	refRe   = regexp.MustCompile(`(?i)^Ref\.?\s*(.+)$`)
	yearRe  = regexp.MustCompile(`(\d{4})年`)
	itemRe  = regexp.MustCompile(`商品番号[:：]\s*([A-Za-z0-9_-]+)`)
	storeRe = regexp.MustCompile(`在庫店舗[:：]\s*(.+)$`)
	totalRe = regexp.MustCompile(`([0-9,]+)`)
	// "../stocklist_vtg/model/submariner_5513_275353/" → 275353
	hrefNoRe = regexp.MustCompile(`_(\d{4,})/?$`)
)

// Crawl pages through the stock search until the total is reached.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	seen := map[string]bool{}
	for page := 0; page < env.MaxPages; page++ {
		u := fmt.Sprintf("%s/rolex_search/result.html?btn_val=kensaku_s&freeword_s=&p=%d&sort=NEW&snum=%d", baseURL, page, pageSize)
		doc, err := env.Fetcher.Doc(ctx, u)
		if err != nil {
			return fmt.Errorf("page %d: %w", page, err)
		}
		items := ParseList(doc)
		if len(items) == 0 {
			if page == 0 {
				return fmt.Errorf("no items on first page (markup changed?)")
			}
			return nil
		}
		for _, l := range items {
			if seen[l.ExternalID] {
				continue
			}
			seen[l.ExternalID] = true
			if err := emit(l); err != nil {
				return err
			}
		}
		if total := Total(doc); total > 0 && (page+1)*pageSize >= total {
			return nil
		}
		if len(items) < pageSize/2 {
			return nil
		}
	}
	return nil
}

// Total returns the result count printed in the sidebar, or 0.
func Total(doc *goquery.Document) int {
	m := totalRe.FindStringSubmatch(crawler.Text(doc.Find("#serch_count .num").First()))
	if m == nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.ReplaceAll(m[1], ",", ""))
	return n
}

// ParseList extracts listings from a result page. Exported for fixture tests.
func ParseList(doc *goquery.Document) []model.Listing {
	var out []model.Listing
	seen := map[string]bool{}
	doc.Find("a.list_item").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		if href == "" {
			return
		}
		spec1 := lines(a.Find(".list_spec1").First())
		spec2 := lines(a.Find(".list_spec2").First())
		if len(spec1) == 0 {
			return
		}
		id := ""
		for _, l := range spec2 {
			if m := itemRe.FindStringSubmatch(l); m != nil {
				id = m[1]
			}
		}
		if id == "" {
			// vintage pages carry no 商品番号 line; the stock number ends the page path instead
			if m := hrefNoRe.FindStringSubmatch(href); m != nil {
				id = m[1]
			} else {
				id = strings.Trim(strings.TrimSuffix(strings.TrimPrefix(href, "../"), ".html"), "/")
			}
		}
		if seen[id] {
			return
		}
		if strings.Contains(strings.ToUpper(crawler.Text(a.Find(".list_price").First())), "SOLD") {
			return // sold pieces stay in the search for a while with "SOLD OUT" in place of a price
		}
		ref := ""
		var parts []string
		for _, l := range spec1 {
			if m := refRe.FindStringSubmatch(l); m != nil {
				ref = strings.ToUpper(strings.TrimSpace(m[1]))
				parts = append(parts, ref)
				continue
			}
			parts = append(parts, l)
		}
		title := normalize.CleanText(strings.Join(parts, " "))
		if title == "" {
			return
		}
		seen[id] = true
		l := model.Listing{
			ExternalID:      id,
			URL:             crawler.AbsURL(doc, href),
			Title:           title,
			ReferenceNumber: ref,
			Currency:        "JPY",
			SellerType:      "dealer",
			SellerName:      "Quark",
			LocationCountry: "JP",
			Condition:       condition(crawler.Text(a.Find(".ttl_cat .cate").First())),
		}
		if len(spec1) >= 2 {
			l.Model = spec1[1]
		}
		if b := normalize.DetectBrand(spec1[0], title); b != nil {
			l.BrandSlug, l.BrandName = b.Slug, b.Name
		}
		if p, ok := normalize.ParsePrice(priceText(a.Find(".list_3y").First())); ok {
			l.Price = &p
		}
		if src, ok := a.Find("img").First().Attr("src"); ok && src != "" {
			l.ImageURLs = []string{crawler.AbsURL(doc, src)}
		}
		attrs := map[string]any{"category": crawler.Text(a.Find(".ttl_cat .cate").First())}
		for _, s := range spec2 {
			if m := yearRe.FindStringSubmatch(s); m != nil && strings.Contains(s, "年式") {
				if y, err := strconv.Atoi(m[1]); err == nil && y >= 1900 && y <= 2100 {
					l.Year = &y
				}
			}
			if m := storeRe.FindStringSubmatch(s); m != nil {
				l.LocationCity = normalize.CleanText(m[1])
				attrs["store"] = l.LocationCity
			}
			if strings.HasPrefix(s, "シリアル") {
				attrs["serial"] = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(s, "シリアル："), "シリアル:"))
			}
		}
		if len(spec1) >= 4 {
			attrs["dial"] = spec1[3]
		}
		if p, ok := normalize.ParsePrice(priceText(a.Find(".list_5y").First())); ok {
			attrs["price5yWarranty"] = p
		}
		l.Attributes, _ = json.Marshal(attrs)
		out = append(out, l)
	})
	return out
}

// priceText returns the amount part of a warranty price block ("３年保証 FAIR価格 ￥2,019,900").
func priceText(s *goquery.Selection) string {
	t := crawler.Text(s)
	if i := strings.LastIndexAny(t, "￥¥"); i >= 0 {
		return t[i:]
	}
	return ""
}

// lines splits a <br>-separated block into trimmed, non-empty lines.
func lines(s *goquery.Selection) []string {
	raw, err := s.Html()
	if err != nil {
		return nil
	}
	var out []string
	for _, part := range brRe.Split(raw, -1) {
		if t := normalize.CleanText(tagRe.ReplaceAllString(part, " ")); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func condition(cat string) model.Condition {
	switch {
	case strings.Contains(cat, "新品"):
		return model.ConditionNew
	case strings.Contains(cat, "アンティーク") || strings.Contains(cat, "ヴィンテージ"):
		return model.ConditionFair
	case strings.Contains(cat, "中古"):
		return model.ConditionGood
	}
	return model.ConditionUnknown
}
