// Package housekihiroba crawls https://housekihiroba.jp (宝石広場, Shibuya, Tokyo; JPY).
//
// Site structure (verified 2026-09): ecbeing shop, Shift_JIS (decoded by the Fetcher).
//   - Landing: /shop/c/cwatch/ links every watch brand as /shop/c/c01<code>/ (20 brands)
//   - Brand list, newest first: /shop/c/c01<code>_ssd/ and /shop/c/c01<code>_ssd_p<n>/
//     (32 cards per page, <a rel="next"> while more exist; Rolex alone is >200 pages, so
//     CRAWL_MAX_PAGES bounds each brand and the sort keeps the newest pieces on the first pages)
//   - Cards: div.goods_ div.StyleT_Item_ with a.goods_name_[href="/shop/g/g<code>/"]
//     img.lazy[data-original] (only the /S/ size exists), .brand_ .name_ .ref_number_,
//     .icon_wrap_ img.auto_[alt=新品|未使用|USED|極美品], .type_ (メンズ), .case_ (サイズ：40.0mm),
//     .move_ (ムーブ：自動巻き), .variation_ (材　質：…), .code_, .price_ ("<s>old</s> 1,750,000円（税込）")
package housekihiroba

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
	baseURL    = "https://housekihiroba.jp"
	landingURL = baseURL + "/shop/c/cwatch/"
)

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "housekihiroba" }

var (
	idRe       = regexp.MustCompile(`/shop/g/g([A-Za-z0-9_-]+)/`)
	brandCatRe = regexp.MustCompile(`^/shop/c/(c01[a-z]{2,3})/$`)
)

// Crawl discovers the brand categories and pages through each, newest first.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	landing, err := env.Fetcher.Doc(ctx, landingURL)
	if err != nil {
		return fmt.Errorf("landing: %w", err)
	}
	cats := BrandCategories(landing)
	if len(cats) == 0 {
		return fmt.Errorf("no brand categories on %s (markup changed?)", landingURL)
	}
	env.Log.Info().Int("brands", len(cats)).Msg("housekihiroba categories")
	seen := map[string]bool{}
	for _, cat := range cats {
		for page := 1; page <= env.MaxPages; page++ {
			u := fmt.Sprintf("%s/shop/c/%s_ssd/", baseURL, cat)
			if page > 1 {
				u = fmt.Sprintf("%s/shop/c/%s_ssd_p%d/", baseURL, cat, page)
			}
			doc, err := env.Fetcher.Doc(ctx, u)
			if err != nil {
				if crawler.IsNotFound(err) {
					break
				}
				return fmt.Errorf("%s page %d: %w", cat, page, err)
			}
			items := ParseList(doc)
			if len(items) == 0 {
				break
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
				break
			}
		}
	}
	return nil
}

// BrandCategories returns the watch brand category codes linked from the landing page.
func BrandCategories(doc *goquery.Document) []string {
	var out []string
	seen := map[string]bool{}
	doc.Find(`a[href^="/shop/c/c01"]`).Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		if m := brandCatRe.FindStringSubmatch(href); m != nil && !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	})
	return out
}

// HasNext reports whether the goods list links to a following page.
func HasNext(doc *goquery.Document) bool {
	return doc.Find(`.navipage_next_ a[rel="next"], a[rel="next"]`).Length() > 0
}

// ParseList extracts listings from a category page. Exported for fixture tests.
func ParseList(doc *goquery.Document) []model.Listing {
	var out []model.Listing
	seen := map[string]bool{}
	doc.Find("div.goods_ div.StyleT_Item_").Each(func(_ int, item *goquery.Selection) {
		a := item.Find("a.goods_name_").First()
		href, _ := a.Attr("href")
		m := idRe.FindStringSubmatch(href)
		if m == nil || seen[m[1]] {
			return
		}
		if item.Find(".soldout_, .soldout").Length() > 0 {
			return
		}
		brand := crawler.Text(item.Find(".brand_").First())
		name := crawler.Text(item.Find(".name_").First())
		ref := crawler.Text(item.Find(".ref_number_").First())
		title := normalize.CleanText(strings.Join([]string{brand, name, ref}, " "))
		if title == "" {
			return
		}
		seen[m[1]] = true
		l := model.Listing{
			ExternalID:      m[1],
			URL:             crawler.AbsURL(doc, href),
			Title:           title,
			Model:           name,
			Currency:        "JPY",
			SellerType:      "dealer",
			SellerName:      "Housekihiroba",
			LocationCountry: "JP",
			LocationCity:    "Tokyo",
		}
		if b := normalize.DetectBrand(brand, name); b != nil {
			l.BrandSlug, l.BrandName = b.Slug, b.Name
		}
		if ref != "" {
			l.ReferenceNumber = strings.ToUpper(ref)
		}
		if alt, ok := item.Find(".icon_wrap_ img.auto_").First().Attr("alt"); ok {
			l.Condition = normalize.Condition(alt)
		}
		l.Gender = normalize.GenderType(crawler.Text(item.Find(".type_").First()))
		l.CaseDiameterMM = normalize.ParseDiameter(crawler.Text(item.Find(".case_").First()))
		l.Movement = normalize.MovementType(crawler.Text(item.Find(".move_").First()))
		if mat := strings.TrimSpace(strings.TrimPrefix(strings.ReplaceAll(crawler.Text(item.Find(".variation_").First()), "材 質：", ""), "材質：")); mat != "" {
			l.CaseMaterial = mat
		}
		price := item.Find(".price_").First().Clone()
		old := crawler.Text(price.Find("s"))
		price.Find("s").Remove()
		if p, ok := normalize.ParsePrice(crawler.Text(price)); ok {
			l.Price = &p
		}
		if src, ok := item.Find("img.lazy").First().Attr("data-original"); ok && src != "" {
			l.ImageURLs = []string{crawler.AbsURL(doc, src)}
		} else if src, ok := item.Find("img").First().Attr("src"); ok && src != "" && !strings.Contains(src, "ajax-loader") {
			l.ImageURLs = []string{crawler.AbsURL(doc, src)}
		}
		attrs := map[string]any{"code": crawler.Text(item.Find(".code_").First())}
		if old != "" {
			attrs["previousPrice"] = old
		}
		var icons []string
		item.Find(".icon_wrap_ img").Each(func(_ int, img *goquery.Selection) {
			if alt, _ := img.Attr("alt"); alt != "" {
				icons = append(icons, alt)
			}
		})
		if len(icons) > 0 {
			attrs["icons"] = icons
		}
		l.Attributes, _ = json.Marshal(attrs)
		out = append(out, l)
	})
	return out
}
