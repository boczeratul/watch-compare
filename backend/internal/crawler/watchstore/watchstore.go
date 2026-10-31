// Package watchstore crawls https://www.watchstore.tw (名錶窩, Taipei; TWD).
//
// Site structure (seen 2026-10 through a page reader only; no raw HTML fixture, see
// docs/CRAWLERS.md): classic ASP on iPAGE hosting ("Powered By iPAGE").
//   - List: the home page lists every piece, newest first:
//     /?keywords=&larcode=&newsclass=&page=<n> (8 pages in 2026-10). Brand pages
//     (/sublist.asp?LarCode=<name>) are not needed.
//   - Card: a[href*="prod_detail.asp?prodnum=<n>"] around a /product/<file>_S.jpg photo and the
//     full title, plus a "價格：$845,000" price line.
//   - The menu also has 鑽石, 手錶配件, 貼膜區 and 名錶維修 sections; non-watch pieces that reach
//     the home page list are dropped by the runner's exclusion rules or have no brand.
package watchstore

import (
	"context"
	"fmt"
	"regexp"

	"github.com/PuerkitoBio/goquery"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

const baseURL = "https://www.watchstore.tw/"

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "watchstore" }

var (
	idRe    = regexp.MustCompile(`(?i)prod_detail\.asp\?prodnum=(\d+)`)
	priceRe = regexp.MustCompile(`價格\s*[:：]?\s*(?:NT)?\$?\s*([\d,]{4,})`)
	photoRe = regexp.MustCompile(`(?i)/?product/[^/]+\.(jpe?g|png|gif|webp)`)
)

// Crawl pages through the home page list until a page brings no new piece.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	seen := map[string]bool{}
	for page := 1; page <= env.MaxPages; page++ {
		u := fmt.Sprintf("%s?keywords=&larcode=&newsclass=&page=%d", baseURL, page)
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
		if fresh == 0 {
			return nil
		}
	}
	return nil
}

// ParseList extracts the pieces from a list page. Exported for fixture tests.
func ParseList(doc *goquery.Document) []model.Listing {
	var out []model.Listing
	for _, c := range crawler.LinkCards(doc, idRe) {
		title := normalize.CleanText(crawler.LongestLinkText(c.Card, idRe, c.ID))
		text := crawler.Text(c.Card)
		if title == "" {
			// the title may sit in a plain cell next to the links: use the card text before the
			// price label
			if i := priceRe.FindStringIndex(text); i != nil {
				title = normalize.CleanText(text[:i[0]])
			}
		}
		if title == "" {
			continue
		}
		l := model.Listing{
			ExternalID:      c.ID,
			URL:             fmt.Sprintf("%sprod_detail.asp?prodnum=%s", baseURL, c.ID),
			Title:           title,
			Currency:        "TWD",
			SellerType:      "dealer",
			SellerName:      "名錶窩",
			LocationCountry: "TW",
			LocationCity:    "Taipei",
			ImageURLs:       crawler.CardImages(doc, c.Card, photoRe),
		}
		if len(l.ImageURLs) > 1 {
			l.ImageURLs = l.ImageURLs[:1]
		}
		if m := priceRe.FindStringSubmatch(text); m != nil {
			if p, ok := normalize.ParsePrice(m[1]); ok {
				l.Price = &p
			}
		}
		if b := normalize.DetectBrand(title); b != nil {
			l.BrandSlug, l.BrandName = b.Slug, b.Name
		}
		l.Condition = normalize.TitleConditionZH(title, model.ConditionVeryGood)
		out = append(out, l)
	}
	return out
}
