// Package xintai crawls the watch section of https://www.89675555.com (鑫泰當鋪 / 鑫泰動產質借中心,
// Tucheng, New Taipei; TWD), a pawn shop selling forfeited pieces (流當精品).
//
// Site structure (seen 2026-10 through a page reader only; no raw HTML fixture, see
// docs/CRAWLERS.md): custom PHP.
//   - List: /goods.php?Ty=1&currentPage=<n> (Ty=1 腕錶類; other Ty values are jewellery, cars,
//     scooters, 3C), 8 pieces per page, about 10 pages. The pager's "next" arrow is broken on
//     page 1, so the crawl stops when a page brings no new piece instead.
//   - Card: a[href*="goods_info.php?SerialNo=<n>"] around an /upload/products/<file>.jpg photo and
//     the title, plus a "NTD 365,000" price line.
//   - Sold pieces stay listed with "(已售出)" in the title and a 999,999 placeholder price; they
//     are skipped.
//   - Product page: /goods_info.php?SerialNo=<n>&Ty=1 with 型號 / 機芯 / 材質 / 錶徑 / 附件 lines.
//
// Many listings are months old, so they may be gone already; the shop confirms stock over LINE.
package xintai

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

const baseURL = "https://www.89675555.com/"

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "xintai" }

var (
	idRe    = regexp.MustCompile(`goods_info\.php\?SerialNo=(\d+)`)
	priceRe = regexp.MustCompile(`NT(?:D|\$)\.?\s*([\d,]{4,})`)
	photoRe = regexp.MustCompile(`(?i)upload/products/`)
)

// Crawl pages through the watch category until a page brings no new piece.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	seen := map[string]bool{}
	for page := 1; page <= env.MaxPages; page++ {
		u := fmt.Sprintf("%sgoods.php?Ty=1&currentPage=%d", baseURL, page)
		doc, err := env.Fetcher.Doc(ctx, u)
		if err != nil {
			return fmt.Errorf("page %d: %w", page, err)
		}
		cards := crawler.LinkCards(doc, idRe)
		if len(cards) == 0 {
			if page == 1 {
				return fmt.Errorf("no items on first page (markup changed?)")
			}
			return nil
		}
		fresh := 0
		for _, l := range ParseList(doc) {
			if seen[l.ExternalID] {
				continue
			}
			seen[l.ExternalID] = true
			fresh++
			if err := emit(l); err != nil {
				return err
			}
		}
		// sold pieces count as seen ids too, so a page of only sold pieces does not end the crawl
		for _, c := range cards {
			if !seen[c.ID] {
				seen[c.ID] = true
				fresh++
			}
		}
		if fresh == 0 {
			return nil
		}
	}
	return nil
}

// ParseList extracts the unsold pieces from a list page. Exported for fixture tests.
func ParseList(doc *goquery.Document) []model.Listing {
	var out []model.Listing
	for _, c := range crawler.LinkCards(doc, idRe) {
		title := normalize.CleanText(crawler.LongestLinkText(c.Card, idRe, c.ID))
		if title == "" {
			continue
		}
		if strings.Contains(title, "已售") {
			continue
		}
		title = strings.TrimSpace(strings.TrimPrefix(title, "鑫泰典精品"))
		l := model.Listing{
			ExternalID:      c.ID,
			URL:             fmt.Sprintf("%sgoods_info.php?SerialNo=%s&Ty=1", baseURL, c.ID),
			Title:           title,
			Currency:        "TWD",
			SellerType:      "dealer",
			SellerName:      "鑫泰當鋪",
			LocationCountry: "TW",
			LocationCity:    "New Taipei",
			ImageURLs:       crawler.CardImages(doc, c.Card, photoRe),
		}
		if len(l.ImageURLs) > 1 {
			l.ImageURLs = l.ImageURLs[:1]
		}
		if m := priceRe.FindStringSubmatch(crawler.Text(c.Card)); m != nil {
			if p, ok := normalize.ParsePrice(m[1]); ok && p != 999999 {
				l.Price = &p
			}
		}
		if b := normalize.DetectBrand(title); b != nil {
			l.BrandSlug, l.BrandName = b.Slug, b.Name
		}
		l.Condition = normalize.TitleConditionZH(title, model.ConditionGood) // a pawn shop: pre-owned by default
		l.Attributes, _ = json.Marshal(map[string]any{"category": "腕錶類"})
		out = append(out, l)
	}
	return out
}
