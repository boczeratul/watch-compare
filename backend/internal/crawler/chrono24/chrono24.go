// Package chrono24 indexes https://www.chrono24.com listing pages.
//
// Chrono24 fronts its site with bot protection: plain HTTP requests from a data-centre IP
// receive HTTP 403 (verified 2026-09). Two supported ways to run this adapter:
//
//  1. CRAWL_RENDER_SERVICE_URL – a headless-browser render endpoint (e.g. a Browserless /
//     Playwright service you operate, or a commercial rendering proxy). The adapter calls
//     `<url>?url=<page>` and expects the fully rendered HTML back.
//  2. CRAWL_PROXY_URL – a residential/forward proxy for plain fetches.
//
// Without either, the adapter fails fast with a clear error and the run is marked "failed"
// without touching existing rows. Review Chrono24's terms of use and robots.txt before
// enabling; the intended production path is an official data-partner feed.
//
// Page structure: /<brand-slug>/index.htm?pageSize=120&showpage=<n>&sortorder=5
// Cards: a.js-article-item[data-article-id][href*="--id<id>.htm"] with
//
//	.text-bold (title), .text-sm (details), [class*="price"], img[data-src|src],
//	and optional JSON-LD ItemList in <script type="application/ld+json">.
package chrono24

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

const baseURL = "https://www.chrono24.com"

// Brands crawled (Chrono24 brand slugs). Mirrors the eBay list so cross-platform comparison overlaps.
var Brands = []string{
	"rolex", "tudor", "omega", "patekphilippe", "audemarspiguet", "vacheronconstantin", "alangesoehne",
	"cartier", "breitling", "iwc", "jaeger-lecoultre", "panerai", "hublot", "tagheuer", "zenith", "breguet",
	"blancpain", "grandseiko", "seiko", "longines", "oris", "nomos", "sinn", "tissot", "hamilton",
}

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "chrono24" }

var (
	idRe    = regexp.MustCompile(`--id(\d+)\.htm`)
	priceRe = regexp.MustCompile(`(?i)(\$|€|£|CHF|¥|HK\$|S\$|A\$|C\$|NT\$|US\$|JPY|USD|EUR|GBP|TWD)\s*([0-9][0-9.,]*)`)
)

// Crawl walks each brand's index pages.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	if env.Cfg.RenderServiceURL == "" && env.Cfg.ProxyURL == "" {
		return fmt.Errorf("chrono24 requires CRAWL_RENDER_SERVICE_URL or CRAWL_PROXY_URL (site blocks plain requests with 403)")
	}
	seen := map[string]bool{}
	for _, brand := range Brands {
		for page := 1; page <= env.MaxPages; page++ {
			u := fmt.Sprintf("%s/%s/index.htm?pageSize=120&showpage=%d&sortorder=5", baseURL, brand, page)
			doc, err := env.Fetcher.DocRendered(ctx, u)
			if err != nil {
				return fmt.Errorf("%s page %d: %w", brand, page, err)
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
			if fresh == 0 {
				break
			}
		}
	}
	return nil
}

// ParseList extracts listings from a Chrono24 result page. Exported for fixture tests.
func ParseList(doc *goquery.Document) []model.Listing {
	var out []model.Listing
	seen := map[string]bool{}
	doc.Find(`a[href*="--id"]`).Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		m := idRe.FindStringSubmatch(href)
		if m == nil || seen[m[1]] {
			return
		}
		card := a
		if card.Find("img").Length() == 0 {
			card = a.Closest(".article-item-container, .rcard, article, li, div")
		}
		title := firstText(card, ".text-bold, .article-title, h3, [class*='title']")
		if title == "" {
			title, _ = a.Attr("title")
		}
		if title == "" {
			return
		}
		seen[m[1]] = true
		l := model.Listing{
			ExternalID: m[1],
			URL:        crawler.AbsURL(doc, href),
			Title:      normalize.CleanText(title),
			Condition:  model.ConditionUnknown,
			SellerType: "dealer",
		}
		details := firstText(card, ".text-sm, .article-subtitle, [class*='subtitle']")
		l.Description = details
		if cur, p, ok := parsePrice(crawler.Text(card)); ok {
			l.Price, l.Currency = &p, cur
		}
		card.Find("img").Each(func(_ int, img *goquery.Selection) {
			src, _ := img.Attr("data-src")
			if src == "" {
				src, _ = img.Attr("src")
			}
			if src != "" && !strings.HasPrefix(src, "data:") && len(l.ImageURLs) < 6 {
				l.ImageURLs = append(l.ImageURLs, upscale(crawler.AbsURL(doc, src)))
			}
		})
		text := strings.ToLower(crawler.Text(card))
		switch {
		case strings.Contains(text, "unworn"):
			l.Condition = model.ConditionUnworn
		case strings.Contains(text, "new"):
			l.Condition = model.ConditionNew
		case strings.Contains(text, "very good"):
			l.Condition = model.ConditionVeryGood
		case strings.Contains(text, "pre-owned") || strings.Contains(text, "good"):
			l.Condition = model.ConditionGood
		}
		if strings.Contains(text, "private seller") {
			l.SellerType = "private"
		}
		if loc := firstText(card, "[class*='location'], .article-item-location"); loc != "" {
			l.LocationCity = loc
		}
		attrs := map[string]any{"details": details}
		l.Attributes, _ = json.Marshal(attrs)
		out = append(out, l)
	})
	return out
}

func firstText(s *goquery.Selection, selector string) string {
	return crawler.Text(s.Find(selector).First())
}

func parsePrice(text string) (string, float64, bool) {
	m := priceRe.FindStringSubmatch(text)
	if m == nil {
		return "", 0, false
	}
	p, ok := normalize.ParsePrice(m[2])
	if !ok {
		return "", 0, false
	}
	cur := map[string]string{"$": "USD", "US$": "USD", "€": "EUR", "£": "GBP", "CHF": "CHF", "¥": "JPY", "JPY": "JPY", "HK$": "HKD", "S$": "SGD", "A$": "AUD", "C$": "CAD", "NT$": "TWD", "USD": "USD", "EUR": "EUR", "GBP": "GBP", "TWD": "TWD"}[strings.ToUpper(m[1])]
	if cur == "" {
		cur = map[string]string{"$": "USD", "€": "EUR", "£": "GBP", "¥": "JPY"}[m[1]]
	}
	return cur, p, true
}

// upscale swaps Chrono24's thumbnail size suffix for a larger rendition when present.
func upscale(u string) string {
	return strings.NewReplacer("-Square-", "-ExtraLarge-", "-Small-", "-ExtraLarge-", "-Medium-", "-ExtraLarge-").Replace(u)
}
