// Package alapower parses the classic ASP shop system built by alapower.com.tw that several
// Taiwanese watch dealers use (Hourstack, RD Watch). Site adapters only supply the base URL,
// seller name and any site-specific text cleanup.
//
// Structure (verified 2026-09):
//   - Brand categories: /product.asp?cat=<n>          (links on the home page nav)
//   - Pagination:       /product.asp?cat=<n>&page=<p>
//   - Product cards:    a fixed-width <td> per product containing a[href^="product_open.asp?ID="],
//     img[src*="product/product_big/"], a.a_table_list_txt (brand line and/or
//     title), "編號 :XXX" and span.shopping_Price (TWD, with or without commas)
//   - Product page:     /product_open.asp?ID=<id> with 款式/材質/錶徑/功能/附件 rows (td.a_txt_744)
package alapower

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

// Site describes one dealer running the platform.
type Site struct {
	BaseURL      string // with trailing slash
	SellerName   string
	FetchDetails bool     // also fetch product pages for material/diameter (slower)
	Boilerplate  []string // markers that start a marketing suffix to strip from titles
}

var (
	idRe   = regexp.MustCompile(`product_open\.asp\?ID=(\d+)`)
	catRe  = regexp.MustCompile(`product\.asp\?cat=(\d+)`)
	snRe   = regexp.MustCompile(`編號\s*[:：]\s*([A-Za-z0-9-]+)`)
	pageRe = regexp.MustCompile(`page=(\d+)`)
)

// Crawl walks every brand category page by page.
func (s Site) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	home, err := env.Fetcher.Doc(ctx, s.BaseURL)
	if err != nil {
		return fmt.Errorf("home: %w", err)
	}
	type cat struct{ id, name string }
	var cats []cat
	seenCat := map[string]bool{}
	home.Find(`a[href*="product.asp?cat="]`).Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		m := catRe.FindStringSubmatch(href)
		if m == nil || seenCat[m[1]] {
			return
		}
		seenCat[m[1]] = true
		cats = append(cats, cat{m[1], strings.Trim(crawler.Text(a), "‧· ")})
	})
	if len(cats) == 0 {
		return fmt.Errorf("no categories found on home page (markup changed?)")
	}
	env.Log.Info().Int("categories", len(cats)).Msg("categories discovered")

	seen := map[string]bool{}
	for _, c := range cats {
		for page := 1; page <= env.MaxPages; page++ {
			u := fmt.Sprintf("%sproduct.asp?cat=%s&page=%d", s.BaseURL, c.id, page)
			doc, err := env.Fetcher.Doc(ctx, u)
			if err != nil {
				return fmt.Errorf("category %s page %d: %w", c.id, page, err)
			}
			items := s.ParseList(doc, c.name)
			newOnPage := 0
			for _, l := range items {
				if seen[l.ExternalID] {
					continue
				}
				seen[l.ExternalID] = true
				newOnPage++
				if s.FetchDetails {
					s.enrich(ctx, env, &l)
				}
				if err := emit(l); err != nil {
					return err
				}
			}
			if newOnPage == 0 || !hasNextPage(doc, page) {
				break
			}
		}
	}
	return nil
}

func hasNextPage(doc *goquery.Document, current int) bool {
	next := false
	doc.Find(`a[href*="page="]`).Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		if m := pageRe.FindStringSubmatch(href); m != nil {
			var p int
			fmt.Sscanf(m[1], "%d", &p) //nolint:errcheck
			if p > current {
				next = true
			}
		}
	})
	return next
}

// ParseList extracts listings from a category page. Cards are located from their price element
// upwards so the platform's varying column widths (260px, 230px, …) do not matter.
func (s Site) ParseList(doc *goquery.Document, categoryName string) []model.Listing {
	var out []model.Listing
	seen := map[string]bool{}
	doc.Find(".shopping_Price").Each(func(_ int, price *goquery.Selection) {
		card := price.Closest(`td[valign="top"][width]`)
		if card.Length() == 0 {
			return
		}
		link := card.Find(`a[href*="product_open.asp?ID="]`).First()
		href, _ := link.Attr("href")
		m := idRe.FindStringSubmatch(href)
		if m == nil || seen[m[1]] {
			return
		}
		seen[m[1]] = true
		l := model.Listing{
			ExternalID: m[1],
			URL:        crawler.AbsURL(doc, href),
			Currency:   "TWD",
			SellerType: "dealer",
			SellerName: s.SellerName,
			Condition:  model.ConditionUnknown,
		}
		// The longest a.a_table_list_txt text is the title; a shorter one (if any) is the brand line.
		var brandLine string
		card.Find("a.a_table_list_txt").Each(func(_ int, a *goquery.Selection) {
			t := normalize.CleanText(crawler.Text(a))
			if len([]rune(t)) > len([]rune(l.Title)) {
				if l.Title != "" {
					brandLine = l.Title
				}
				l.Title = t
			} else if t != "" {
				brandLine = t
			}
		})
		l.Title = s.stripBoilerplate(l.Title)
		if l.Title == "" {
			return
		}
		if b := normalize.DetectBrand(brandLine, categoryName, l.Title); b != nil {
			l.BrandSlug, l.BrandName = b.Slug, b.Name
		}
		if img, ok := card.Find(`img[src*="product_big"]`).First().Attr("src"); ok {
			l.ImageURLs = []string{crawler.AbsURL(doc, img)}
		}
		if p, ok := normalize.ParsePrice(crawler.Text(price)); ok {
			l.Price = &p
		}
		attrs := map[string]any{"category": categoryName}
		if sn := snRe.FindStringSubmatch(crawler.Text(card)); sn != nil {
			attrs["stockNumber"] = sn[1]
		}
		l.Attributes, _ = json.Marshal(attrs)
		switch {
		case strings.Contains(l.Title, "全新") || strings.Contains(l.Title, "未使用"):
			l.Condition = model.ConditionUnworn
		case strings.Contains(l.Title, "二手") || strings.Contains(l.Title, "中古"):
			l.Condition = model.ConditionGood
		default:
			l.Condition = model.ConditionVeryGood // these dealers sell mostly pre-owned in top condition
		}
		if strings.Contains(l.Title, "女錶") || strings.Contains(categoryName, "女錶") {
			l.Gender = model.GenderWomen
		}
		l.Model = guessModel(l.Title, l.BrandName)
		out = append(out, l)
	})
	return out
}

func (s Site) stripBoilerplate(title string) string {
	for _, m := range s.Boilerplate {
		if i := strings.Index(title, m); i > 0 {
			title = title[:i]
		}
	}
	return strings.TrimSpace(title)
}

// enrich fetches the product page for material / diameter / description.
func (s Site) enrich(ctx context.Context, env *crawler.Env, l *model.Listing) {
	doc, err := env.Fetcher.Doc(ctx, l.URL)
	if err != nil {
		env.Log.Warn().Err(err).Str("id", l.ExternalID).Msg("detail fetch failed")
		return
	}
	doc.Find("td.a_txt_744").Each(func(i int, td *goquery.Selection) {
		label := crawler.Text(td)
		val := crawler.Text(td.Next())
		switch {
		case strings.HasPrefix(label, "款式"):
			if val != "" {
				l.Model = val
			}
		case strings.HasPrefix(label, "材質"):
			l.CaseMaterial = val
		case strings.HasPrefix(label, "錶徑"):
			l.CaseDiameterMM = normalize.ParseDiameter(val)
		case strings.HasPrefix(label, "功能"):
			l.Description = val
			l.Movement = normalize.MovementType(val)
		case strings.HasPrefix(label, "附件"):
			l.HasBox, l.HasPapers = normalize.DetectBoxPapers(val)
		}
	})
	var imgs []string
	doc.Find(`img[src*="product_big"]`).Each(func(_ int, img *goquery.Selection) {
		if src, ok := img.Attr("src"); ok {
			imgs = append(imgs, crawler.AbsURL(doc, src))
		}
	})
	if len(imgs) > 0 {
		l.ImageURLs = imgs
	}
}

// guessModel takes the Latin words after the brand and before the first reference-like token
// as the model name (e.g. "Navitimer B01 Chronograph").
func guessModel(title, brand string) string {
	var out []string
	for _, w := range strings.Fields(title) {
		lw := strings.ToLower(w)
		if brand != "" && strings.Contains(strings.ToLower(brand), lw) {
			continue
		}
		if !isLatin(w) {
			if len(out) > 0 {
				break
			}
			continue
		}
		if len(w) > 4 && normalize.ExtractReference(w) == strings.ToUpper(w) || strings.HasSuffix(lw, "mm") {
			break
		}
		if len(out) >= 4 {
			break
		}
		out = append(out, w)
	}
	return strings.Join(out, " ")
}

func isLatin(s string) bool {
	for _, r := range s {
		if r > 0x24F {
			return false
		}
	}
	return s != ""
}
