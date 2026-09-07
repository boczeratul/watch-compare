// Package ishida crawls the BEST VINTAGE (pre-owned) section of https://ishida-watch.com
// (BEST ISHIDA, Tokyo; JPY).
//
// Site structure (verified 2026-09): futureshop, UTF-8.
//   - List, newest first: /c/bestvintage/v_brand?page=<n>&sort=latest (20 cards per page,
//     ~5.2k pieces / 260 pages; <link rel="next"> while more exist, so CRAWL_MAX_PAGES bounds a run)
//   - Cards: article.fs-c-productListItem[data-product-id] with a[href="/c/bestvintage/<no>-h<nn>"],
//     img.fs-c-productListItem__image__image[data-layzr] (size=m) and the gallery
//     .fs-c-productImageModalCarousel__figure__image[data-src] (size=xl),
//     .fs-c-productName__name "【中古】CARTIER <br>SANTOS … <br>カルティエ <br>サントス … <br>WHSA0015"
//     (English brand, English model, Japanese brand, Japanese model, reference),
//     .fs-c-productMark__label (⾃動巻き / 手巻き / クォーツ, メンズ / レディース),
//     .fs-c-price__value (税込)
package ishida

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

const (
	baseURL  = "https://ishida-watch.com"
	pageSize = 20
)

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "ishida" }

var (
	idRe    = regexp.MustCompile(`/c/bestvintage/([A-Za-z0-9_-]+)$`)
	gradeRe = regexp.MustCompile(`^【([^】]+)】\s*`)
	brRe    = regexp.MustCompile(`(?i)<br\s*/?>`)
	tagRe   = regexp.MustCompile(`<[^>]+>`)
	refRe   = regexp.MustCompile(`^[A-Z0-9][A-Z0-9./-]{2,}$`)
)

// Crawl pages through the vintage list until the last page.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	seen := map[string]bool{}
	for page := 1; page <= env.MaxPages; page++ {
		u := fmt.Sprintf("%s/c/bestvintage/v_brand?page=%d&sort=latest", baseURL, page)
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
	return doc.Find(`link[rel="next"], .fs-c-pagination__item--next[href]`).Length() > 0
}

// ParseList extracts listings from a list page. Exported for fixture tests.
func ParseList(doc *goquery.Document) []model.Listing {
	var out []model.Listing
	seen := map[string]bool{}
	doc.Find("article.fs-c-productListItem").Each(func(_ int, art *goquery.Selection) {
		a := art.Find(`.fs-c-productName a[href], a[href*="/c/bestvintage/"]`).First()
		href, _ := a.Attr("href")
		m := idRe.FindStringSubmatch(href)
		if m == nil || seen[m[1]] {
			return
		}
		if art.Find(".fs-c-productListItem__outOfStock, .fs-c-productStock--outOfStock").Length() > 0 {
			return
		}
		lines := nameLines(art.Find(".fs-c-productName__name").First())
		if len(lines) == 0 {
			lines = nameLines(art.Find(".fs-c-productName__copy").First())
		}
		if len(lines) == 0 {
			return
		}
		grade := ""
		if gm := gradeRe.FindStringSubmatch(lines[0]); gm != nil {
			grade = gm[1]
			lines[0] = strings.TrimSpace(gradeRe.ReplaceAllString(lines[0], ""))
		}
		ref := ""
		if last := strings.ToUpper(lines[len(lines)-1]); len(lines) > 1 && refRe.MatchString(last) {
			ref = last
			lines = lines[:len(lines)-1]
		}
		title := normalize.CleanText(strings.Join(append(lines, ref), " "))
		if title == "" {
			return
		}
		seen[m[1]] = true
		l := model.Listing{
			ExternalID:      m[1],
			URL:             crawler.AbsURL(doc, href),
			Title:           title,
			ReferenceNumber: ref,
			Currency:        "JPY",
			SellerType:      "dealer",
			SellerName:      "BEST ISHIDA",
			LocationCountry: "JP",
			LocationCity:    "Tokyo",
			Condition:       normalize.Condition(grade),
		}
		if b := normalize.DetectBrand(lines...); b != nil {
			l.BrandSlug, l.BrandName = b.Slug, b.Name
		}
		if len(lines) >= 2 {
			l.Model = lines[1]
		}
		if p, ok := normalize.ParsePrice(crawler.Text(art.Find(".fs-c-productPrice--selling .fs-c-price__value").First())); ok {
			l.Price = &p
		}
		var marks []string
		art.Find(".fs-c-productMark__label").Each(func(_ int, sp *goquery.Selection) {
			// the shop types 自 as the Kangxi radical ⾃ (U+2F8A) in "⾃動巻き"
			t := strings.ReplaceAll(crawler.Text(sp), "⾃", "自")
			if t == "" {
				return
			}
			marks = append(marks, t)
			if mv := normalize.MovementType(t); mv != model.MovementUnknown && l.Movement == "" {
				l.Movement = mv
			}
			if g := normalize.GenderType(t); g != model.GenderUnknown && l.Gender == "" {
				l.Gender = g
			}
		})
		art.Find(".fs-c-productImageModalCarousel__figure__image").Each(func(_ int, img *goquery.Selection) {
			src, _ := img.Attr("data-src")
			if src == "" {
				src, _ = img.Attr("data-lazy")
			}
			if src != "" && len(l.ImageURLs) < 6 {
				l.ImageURLs = append(l.ImageURLs, src)
			}
		})
		if len(l.ImageURLs) == 0 {
			if src, ok := art.Find("img.fs-c-productListItem__image__image").First().Attr("data-layzr"); ok && src != "" {
				l.ImageURLs = []string{src}
			}
		}
		attrs := map[string]any{"grade": grade, "marks": marks}
		if pid, ok := art.Attr("data-product-id"); ok {
			attrs["productId"] = pid
		}
		l.Attributes, _ = json.Marshal(attrs)
		out = append(out, l)
	})
	return out
}

// nameLines splits a <br>-separated product name into trimmed, non-empty lines.
func nameLines(s *goquery.Selection) []string {
	raw, err := s.Html()
	if err != nil {
		return nil
	}
	var out []string
	for _, part := range brRe.Split(raw, -1) {
		t := normalize.CleanText(html.UnescapeString(tagRe.ReplaceAllString(part, " ")))
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}
