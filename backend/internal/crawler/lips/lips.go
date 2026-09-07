// Package lips crawls https://lips-online.jp (Brand Shop LIPS; JPY).
//
// Site structure (verified 2026-09): EC-CUBE 4 shop, UTF-8. The watch category holds ~43k
// products including sold ones, so the crawl asks for in-stock pieces only, newest first:
//   - List: /ec/products/list?category_id=1&stock_flg[]=1&orderby=2&disp_number=40&pageno=<n>
//     (category 1 = 時計, ~1.1k in stock, 40 cards per page)
//   - Cards: li.ec-shelfGrid__item > a[href="https://lips-online.jp/ec/products/detail/<id>"] with
//     figure img, p.u-color--accent01 ("ROLEX(ロレックス)"), h2.c-card__heading (model name),
//     a table of 型番 / 素材 / 程度 / 付属 / 定価 rows, .price02-default (税込 price) and
//     ul.c-product-card__hashtags
package lips

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
	baseURL  = "https://lips-online.jp"
	pageSize = 40
)

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "lips" }

var (
	idRe = regexp.MustCompile(`/ec/products/detail/(\d+)`)
	// 型番 often carries the serial-letter range in parentheses: "116509G(V)", "214270(ﾗﾝﾀﾞﾑ)"
	refSuffixRe = regexp.MustCompile(`\s*[（(].*$`)
)

// Crawl pages through the in-stock watch list until a short page.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	seen := map[string]bool{}
	for page := 1; page <= env.MaxPages; page++ {
		u := fmt.Sprintf("%s/ec/products/list?category_id=1&stock_flg%%5B%%5D=1&orderby=2&disp_number=%d&pageno=%d", baseURL, pageSize, page)
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
		if fresh == 0 || len(items) < pageSize {
			return nil
		}
	}
	return nil
}

// ParseList extracts listings from a list page. Exported for fixture tests.
func ParseList(doc *goquery.Document) []model.Listing {
	var out []model.Listing
	seen := map[string]bool{}
	doc.Find("li.ec-shelfGrid__item").Each(func(_ int, li *goquery.Selection) {
		a := li.Find(`a[href*="/ec/products/detail/"]`).First()
		href, _ := a.Attr("href")
		m := idRe.FindStringSubmatch(href)
		if m == nil || seen[m[1]] {
			return
		}
		brandLine := crawler.Text(li.Find("p.u-color--accent01").First())
		brandEn := crawler.Text(li.Find(`p.u-color--accent01 span[lang="en"]`).First())
		name := crawler.Text(li.Find("h2.c-card__heading").First())
		rows := map[string]string{}
		li.Find("table tr").Each(func(_ int, tr *goquery.Selection) {
			k := strings.TrimRight(crawler.Text(tr.Find("th").First()), "：:")
			if k != "" {
				rows[k] = crawler.Text(tr.Find("td").First())
			}
		})
		serial := rows["型番"]
		ref := strings.TrimSpace(refSuffixRe.ReplaceAllString(serial, ""))
		title := normalize.CleanText(strings.Join([]string{brandEn, name, serial}, " "))
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
			SellerName:      "Brand Shop LIPS",
			LocationCountry: "JP",
			Condition:       normalize.Condition(strings.TrimSuffix(rows["程度"], "ランク")),
		}
		if b := normalize.DetectBrand(brandEn, brandLine); b != nil {
			l.BrandSlug, l.BrandName = b.Slug, b.Name
		}
		if ref != "" {
			l.ReferenceNumber = strings.ToUpper(ref)
		}
		if p, ok := normalize.ParsePrice(crawler.Text(li.Find(".price02-default").First())); ok {
			l.Price = &p
		}
		if src, ok := li.Find("figure img").First().Attr("src"); ok && src != "" {
			l.ImageURLs = []string{crawler.AbsURL(doc, src)}
		}
		if mat := rows["素材"]; mat != "" {
			l.CaseMaterial = mat
		}
		if acc := rows["付属"]; acc != "" {
			l.HasBox, l.HasPapers = normalize.DetectBoxPapers(acc)
			l.Description = "付属: " + acc
		}
		attrs := map[string]any{"grade": rows["程度"], "accessories": rows["付属"], "listPrice": rows["定価"], "modelNumber": serial}
		var tags []string
		li.Find("ul.c-product-card__hashtags a").Each(func(_ int, t *goquery.Selection) {
			if s := crawler.Text(t); s != "" {
				tags = append(tags, s)
			}
		})
		if len(tags) > 0 {
			attrs["tags"] = tags
		}
		l.Attributes, _ = json.Marshal(attrs)
		out = append(out, l)
	})
	return out
}
