// Package jackroad crawls https://www.jackroad.co.jp (Japan, JPY, Shift_JIS pages).
//
// Site structure (verified 2026-09): ebisumart-style ASP.NET shop.
//   - Landing page: /shop/r/rjw/ links every watch brand category as /shop/r/rjw<code>/
//   - Brand list:   /shop/r/rjw<code>/?p=<n>   (60 cards per page)
//   - Cards:        div.list_type_03_ ul > li with
//     a.goods_name_[href="/shop/g/g<id>/"], img.lazyload[data-src],
//     .state_box_ (中古/新品/未使用), .icon_product_info_ span (メンズ/レディース/在庫あり),
//     .brand-en, .brand-ja, .item_name_, .item_diameter_detail_search_ span,
//     .item_movement_detail_search_ span, .item_no_ span (型番), .item_id_ span, .price_main_
package jackroad

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
	baseURL    = "https://www.jackroad.co.jp"
	landingURL = baseURL + "/shop/r/rjw/"
)

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "jackroad" }

var (
	idRe       = regexp.MustCompile(`/shop/g/g([A-Za-z0-9_-]+)/`)
	brandCatRe = regexp.MustCompile(`^/shop/r/(rjw[a-z0-9]+)/$`)
)

// Crawl discovers every brand category from the landing page and pages through each.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	landing, err := env.Fetcher.Doc(ctx, landingURL)
	if err != nil {
		return fmt.Errorf("landing: %w", err)
	}
	cats := BrandCategories(landing)
	if len(cats) == 0 {
		return fmt.Errorf("no brand categories on landing page (markup changed?)")
	}
	env.Log.Info().Int("categories", len(cats)).Msg("jackroad brand categories discovered")

	seen := map[string]bool{}
	for _, cat := range cats {
		for page := 1; page <= env.MaxPages; page++ {
			u := fmt.Sprintf("%s/shop/r/%s/", baseURL, cat)
			if page > 1 {
				u += fmt.Sprintf("?p=%d", page)
			}
			doc, err := env.Fetcher.Doc(ctx, u)
			if err != nil {
				if crawler.IsNotFound(err) {
					break
				}
				return fmt.Errorf("%s page %d: %w", cat, page, err)
			}
			items := ParseList(doc)
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
			if fresh == 0 || len(items) < 60 {
				break
			}
		}
	}
	return nil
}

// BrandCategories extracts watch brand category keys (rjwrx, rjwom, ...) from a page's navigation.
func BrandCategories(doc *goquery.Document) []string {
	seen := map[string]bool{}
	var out []string
	doc.Find(`a[href^="/shop/r/rjw"]`).Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		m := brandCatRe.FindStringSubmatch(href)
		if m == nil || m[1] == "rjw" || seen[m[1]] {
			return
		}
		seen[m[1]] = true
		out = append(out, m[1])
	})
	return out
}

// ParseList extracts listings from a list page. Exported for fixture tests.
func ParseList(doc *goquery.Document) []model.Listing {
	var out []model.Listing
	doc.Find("div.list_type_03_ ul > li, div.list_type_03_ li").Each(func(_ int, li *goquery.Selection) {
		a := li.Find("a.goods_name_").First()
		href, _ := a.Attr("href")
		m := idRe.FindStringSubmatch(href)
		if m == nil {
			return
		}
		l := model.Listing{
			ExternalID:      m[1],
			URL:             crawler.AbsURL(doc, href),
			Currency:        "JPY",
			SellerType:      "dealer",
			SellerName:      "Jackroad",
			LocationCountry: "JP",
			LocationCity:    "Tokyo",
		}
		brandEn := crawler.Text(li.Find(".brand-en"))
		brandJa := crawler.Text(li.Find(".brand-ja"))
		itemName := crawler.Text(li.Find(".item_name_"))
		l.Model = itemName
		l.Title = normalize.CleanText(strings.TrimSpace(firstNonEmpty(brandEn, brandJa) + " " + itemName))
		if b := normalize.DetectBrand(brandEn, brandJa, itemName); b != nil {
			l.BrandSlug, l.BrandName = b.Slug, b.Name
			if brandEn == "" {
				l.Title = normalize.CleanText(b.Name + " " + itemName)
			}
		}
		if src, ok := li.Find("img").First().Attr("data-src"); ok && src != "" {
			l.ImageURLs = []string{toLarge(crawler.AbsURL(doc, src))}
		} else if src, ok := li.Find("img").First().Attr("src"); ok && !strings.HasPrefix(src, "data:") {
			l.ImageURLs = []string{crawler.AbsURL(doc, src)}
		}
		l.ReferenceNumber = crawler.Text(li.Find(".item_no_ span"))
		l.CaseDiameterMM = normalize.ParseDiameter(crawler.Text(li.Find(".item_diameter_detail_search_")))
		l.Movement = normalize.MovementType(crawler.Text(li.Find(".item_movement_detail_search_")))
		state := crawler.Text(li.Find(".state_box_"))
		l.Condition = normalize.Condition(state)
		info := crawler.Text(li.Find(".icon_product_info_"))
		l.Gender = normalize.GenderType(info)
		if p, ok := normalize.ParsePrice(crawler.Text(li.Find(".price_main_"))); ok {
			l.Price = &p
		}
		attrs := map[string]any{"state": state, "info": info}
		if id := crawler.Text(li.Find(".item_id_ span")); id != "" {
			attrs["goodsId"] = id
		}
		if in := crawler.Text(li.Find(".in_stock")); in != "" {
			attrs["arrived"] = in
		}
		l.Attributes, _ = json.Marshal(attrs)
		if strings.Contains(info, "売約") || strings.Contains(state, "SOLD") {
			return // sold: skip so it gets deactivated
		}
		out = append(out, l)
	})
	return out
}

// toLarge swaps the 160px thumbnail (/img/goods/S/<id>.jpg) for the product-page main image
// (/img/goods/1/<id>.jpg), verified to exist for every goods id (2026-09).
func toLarge(u string) string {
	return strings.Replace(u, "/img/goods/S/", "/img/goods/1/", 1)
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if strings.TrimSpace(x) != "" {
			return x
		}
	}
	return ""
}
