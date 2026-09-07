// Package hourstack crawls https://www.hourstack.com.tw (Taiwan, TWD).
//
// Site structure (verified 2026-09): classic ASP pages.
//   - Brand categories: /product.asp?cat=<n>   (links on the home page nav)
//   - Pagination:       /product.asp?cat=<n>&page=<p>
//   - Product cards:    <td width="260"> blocks with a[href^="product_open.asp?ID="],
//     img[src*="product/product_big/"], two a.a_table_list_txt (brand, title),
//     "編號 :XXX" and span.shopping_Price (TWD, no decimals)
//   - Product page:     /product_open.asp?ID=<id> with 款式/材質/錶徑/功能/附件 rows (td.a_txt_744)
package hourstack

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

const baseURL = "https://www.hourstack.com.tw/"

// Source implements crawler.Source.
type Source struct {
	FetchDetails bool // fetch product pages for material/diameter (slower)
}

// Key returns the source key.
func (Source) Key() string { return "hourstack" }

var (
	idRe   = regexp.MustCompile(`product_open\.asp\?ID=(\d+)`)
	catRe  = regexp.MustCompile(`product\.asp\?cat=(\d+)`)
	snRe   = regexp.MustCompile(`編號\s*[:：]\s*([A-Za-z0-9-]+)`)
	pageRe = regexp.MustCompile(`page=(\d+)`)
)

// Crawl walks every brand category page by page.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	home, err := env.Fetcher.Doc(ctx, baseURL)
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
		cats = append(cats, cat{m[1], crawler.Text(a)})
	})
	if len(cats) == 0 {
		return fmt.Errorf("no categories found on home page (markup changed?)")
	}
	env.Log.Info().Int("categories", len(cats)).Msg("hourstack categories discovered")

	seen := map[string]bool{}
	for _, c := range cats {
		for page := 1; page <= env.MaxPages; page++ {
			u := fmt.Sprintf("%sproduct.asp?cat=%s&page=%d", baseURL, c.id, page)
			doc, err := env.Fetcher.Doc(ctx, u)
			if err != nil {
				return fmt.Errorf("category %s page %d: %w", c.id, page, err)
			}
			items := s.parseList(doc, c.name)
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

// parseList extracts listings from a category page.
func (s Source) parseList(doc *goquery.Document, categoryName string) []model.Listing {
	var out []model.Listing
	doc.Find(`td[width="260"]`).Each(func(_ int, card *goquery.Selection) {
		link := card.Find(`a[href*="product_open.asp?ID="]`).First()
		href, _ := link.Attr("href")
		m := idRe.FindStringSubmatch(href)
		if m == nil {
			return
		}
		l := model.Listing{
			ExternalID: m[1],
			URL:        crawler.AbsURL(doc, href),
			Currency:   "TWD",
			SellerType: "dealer",
			SellerName: "Hourstack",
			Condition:  model.ConditionUnknown,
		}
		texts := card.Find("a.a_table_list_txt")
		brandLine := crawler.Text(texts.Eq(0))
		l.Title = stripBoilerplate(normalize.CleanText(crawler.Text(texts.Eq(1))))
		if l.Title == "" {
			l.Title = normalize.CleanText(brandLine)
		}
		if b := normalize.DetectBrand(brandLine, categoryName, l.Title); b != nil {
			l.BrandSlug, l.BrandName = b.Slug, b.Name
		}
		if img, ok := card.Find(`img[src*="product_big"]`).First().Attr("src"); ok {
			l.ImageURLs = []string{crawler.AbsURL(doc, img)}
		}
		if p, ok := normalize.ParsePrice(crawler.Text(card.Find(".shopping_Price").First())); ok {
			l.Price = &p
		}
		cardText := crawler.Text(card)
		attrs := map[string]any{"category": categoryName}
		if sn := snRe.FindStringSubmatch(cardText); sn != nil {
			attrs["stockNumber"] = sn[1]
		}
		l.Attributes = mustJSON(attrs)
		// Hourstack sells mostly pre-owned, with explicit markers for new pieces.
		switch {
		case strings.Contains(l.Title, "全新") || strings.Contains(l.Title, "未使用"):
			l.Condition = model.ConditionUnworn
		case strings.Contains(l.Title, "二手") || strings.Contains(l.Title, "中古"):
			l.Condition = model.ConditionGood
		default:
			l.Condition = model.ConditionVeryGood
		}
		if strings.Contains(l.Title, "女錶") || strings.Contains(categoryName, "女錶") {
			l.Gender = model.GenderWomen
		}
		l.Model = guessModel(l.Title, l.BrandName)
		out = append(out, l)
	})
	return out
}

// enrich fetches the product page for material / diameter / description.
func (s Source) enrich(ctx context.Context, env *crawler.Env, l *model.Listing) {
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

// boilerplateMarkers start the dealer's marketing suffix that Hourstack appends to many titles.
var boilerplateMarkers = []string{"誠摯邀請", "本店承諾", "歡迎洽詢", "歡迎來店"}

func stripBoilerplate(title string) string {
	for _, m := range boilerplateMarkers {
		if i := strings.Index(title, m); i > 0 {
			title = title[:i]
		}
	}
	return strings.TrimSpace(title)
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

func mustJSON(m map[string]any) []byte {
	b, _ := jsonMarshal(m)
	return b
}
