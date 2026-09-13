// Package watchandtime crawls https://www.watchandtime.com (黃忠政名錶交流中心 / James Huang
// Vintage Watches, Taiwan; TWD).
//
// Site structure (verified 2026-09): classic ASP site in Big5 (decoded by the Fetcher).
//   - Home: index.asp links every brand category as products_list.asp?id=<n> (31 categories;
//     the 鑽石／配件 one is jewellery and is skipped)
//   - Category: products_list.asp?id=<n>&page=<p> (24 cards per page; the header reads
//     "總數 74 筆．目前 1 /4頁")
//   - Cards: a[href="products_open.asp?id=<id>"] img[src="product/product_big/m-<n>.JPG"],
//     td.product_t (brand), td.product_t1 (stock code "M09222" and "售價 ：355000元")
//   - The card carries no model or reference, so every product page is fetched: its text has
//     "編號：M09222", "售價：NT$ 355000元", "CASE：不鏽鋼，Rolex Oyster Perpetual Date GMT-Master
//     16750 "Pepsi". 40mm.", "功能：男錶，Cal.3075 自動機芯，…", "附件狀況：附原裝…，1985 年製造，…"
//     and product/product_picture/*.jpg photos.
package watchandtime

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

const (
	baseURL  = "https://www.watchandtime.com/"
	pageSize = 24
)

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "watchandtime" }

var (
	catRe   = regexp.MustCompile(`products_list\.asp\?id=(\d+)`)
	idRe    = regexp.MustCompile(`products_open\.asp\?id=(\d+)`)
	countRe = regexp.MustCompile(`總數\s*(\d+)\s*筆[^0-9]*?(\d+)\s*/\s*(\d+)\s*頁`)
	fieldRe = regexp.MustCompile(`(?s)(編號|售價|CASE|功能|附件狀況)\s*[：:]\s*(.*?)\s*(?:編號|售價|CASE|功能|附件狀況)\s*[：:]|(?s)(附件狀況)\s*[：:]\s*(.*?)(?:│|回首頁|$)`)
	labelRe = regexp.MustCompile(`(編號|售價|CASE|功能|附件狀況)\s*[：:]`)
)

// Category is a brand category on the home page.
type Category struct {
	ID   string
	Name string
}

// Crawl walks every brand category page by page and fetches each product page.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	home, err := env.Fetcher.Doc(ctx, baseURL+"index.asp")
	if err != nil {
		return fmt.Errorf("home: %w", err)
	}
	cats := Categories(home)
	if len(cats) == 0 {
		return fmt.Errorf("no categories on the home page (markup changed?)")
	}
	env.Log.Info().Int("categories", len(cats)).Msg("watchandtime categories")
	seen := map[string]bool{}
	for _, cat := range cats {
		for page := 1; page <= env.MaxPages; page++ {
			u := fmt.Sprintf("%sproducts_list.asp?id=%s&page=%d", baseURL, cat.ID, page)
			doc, err := env.Fetcher.Doc(ctx, u)
			if err != nil {
				return fmt.Errorf("category %s page %d: %w", cat.ID, page, err)
			}
			items := ParseList(doc, cat.Name)
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
				if detail, err := env.Fetcher.Doc(ctx, l.URL); err != nil {
					env.Log.Warn().Err(err).Str("id", l.ExternalID).Msg("product page failed; using list data")
				} else {
					ApplyDetail(&l, detail)
				}
				if err := emit(l); err != nil {
					return err
				}
			}
			_, cur, last := Pagination(doc)
			if fresh == 0 || cur >= last || len(items) < pageSize {
				break
			}
		}
	}
	return nil
}

// Categories returns the brand categories linked from the home page, jewellery excluded.
func Categories(doc *goquery.Document) []Category {
	var out []Category
	seen := map[string]bool{}
	doc.Find(`a[href*="products_list.asp?id="]`).Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		m := catRe.FindStringSubmatch(href)
		name := crawler.Text(a)
		if m == nil || seen[m[1]] || name == "" {
			return
		}
		if strings.Contains(name, "鑽石") || strings.Contains(name, "配件") || strings.Contains(name, "珠寶") {
			return
		}
		seen[m[1]] = true
		out = append(out, Category{ID: m[1], Name: name})
	})
	return out
}

// Pagination reads "總數 74 筆．目前 1 /4頁" from a category page: total items, current page,
// last page. Zeroes mean the header was not found.
func Pagination(doc *goquery.Document) (total, current, last int) {
	m := countRe.FindStringSubmatch(crawler.Text(doc.Selection))
	if m == nil {
		return 0, 0, 0
	}
	total, _ = strconv.Atoi(m[1])
	current, _ = strconv.Atoi(m[2])
	last, _ = strconv.Atoi(m[3])
	return total, current, last
}

// ParseList extracts the cards of a category page. Titles are provisional (brand + stock code)
// until ApplyDetail adds the product page's description. Exported for fixture tests.
func ParseList(doc *goquery.Document, categoryName string) []model.Listing {
	var out []model.Listing
	seen := map[string]bool{}
	doc.Find(`a[href*="products_open.asp?id="]`).Each(func(_ int, a *goquery.Selection) {
		if a.Find("img").Length() == 0 {
			return // the brand-name link of the same card
		}
		href, _ := a.Attr("href")
		m := idRe.FindStringSubmatch(href)
		if m == nil || seen[m[1]] {
			return
		}
		card := a.ParentsFiltered("table").Eq(1)
		if card.Length() == 0 {
			card = a.Parent()
		}
		brand := crawler.Text(card.Find("td.product_t").First())
		code, priceText := "", ""
		card.Find("td.product_t1").Each(func(_ int, td *goquery.Selection) {
			t := crawler.Text(td)
			if strings.Contains(t, "售價") {
				priceText = t
			} else if code == "" {
				code = t
			}
		})
		seen[m[1]] = true
		l := model.Listing{
			ExternalID:      m[1],
			URL:             crawler.AbsURL(doc, href),
			Title:           normalize.CleanText(strings.TrimSpace(brand + " " + code)),
			Currency:        "TWD",
			SellerType:      "dealer",
			SellerName:      "James Huang Vintage Watches",
			LocationCountry: "TW",
			Condition:       model.ConditionGood, // a pre-owned and vintage dealer
		}
		if b := normalize.DetectBrand(brand, categoryName); b != nil {
			l.BrandSlug, l.BrandName = b.Slug, b.Name
		}
		if p, ok := normalize.ParsePrice(priceText); ok {
			l.Price = &p
		}
		if src, ok := a.Find("img").First().Attr("src"); ok && src != "" {
			l.ImageURLs = []string{crawler.AbsURL(doc, src)}
		}
		attrs := map[string]any{"category": categoryName}
		if code != "" {
			attrs["stockCode"] = code
		}
		l.Attributes, _ = json.Marshal(attrs)
		out = append(out, l)
	})
	return out
}

// ApplyDetail fills the title, material, movement, gender, year, description and photos from a
// product page. Exported for fixture tests.
func ApplyDetail(l *model.Listing, doc *goquery.Document) {
	text := crawler.Text(doc.Find("body"))
	fields := map[string]string{}
	locs := labelRe.FindAllStringIndex(text, -1)
	for i, loc := range locs {
		label := strings.TrimRight(strings.TrimSpace(text[loc[0]:loc[1]]), "：:")
		end := len(text)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		val := strings.TrimSpace(text[loc[1]:end])
		if label == "附件狀況" {
			// the last field runs into the footer navigation
			for _, stop := range []string{"│", "回首頁"} {
				if j := strings.Index(val, stop); j >= 0 {
					val = strings.TrimSpace(val[:j])
				}
			}
		}
		if _, dup := fields[label]; !dup {
			fields[label] = val
		}
	}
	var attrs map[string]any
	_ = json.Unmarshal(l.Attributes, &attrs)
	if attrs == nil {
		attrs = map[string]any{}
	}
	if c := fields["CASE"]; c != "" {
		desc := c
		// "不鏽鋼，Rolex Oyster Perpetual Date GMT-Master 16750 "Pepsi". 40mm." → material, then the watch
		// "不鏽鋼，Rolex Oyster … 16750" or "18K Rose Gold. Saxonia 842.032 …": a short material
		// prefix ends at the first comma, or at the first full stop when there is no comma
		if i := strings.IndexAny(c, "，,"); i > 0 && i < 30 {
			_, w := utf8.DecodeRuneInString(c[i:])
			l.CaseMaterial = strings.TrimSpace(c[:i])
			desc = strings.TrimSpace(c[i+w:])
		} else if i := strings.Index(c, ". "); i > 0 && i < 20 {
			l.CaseMaterial = strings.TrimSpace(c[:i])
			desc = strings.TrimSpace(c[i+2:])
		}
		if desc != "" {
			l.Title = normalize.CleanText(strings.Trim(desc, " ."))
		}
		attrs["case"] = c
	}
	if f := fields["功能"]; f != "" {
		attrs["functions"] = f
		if mv := normalize.MovementType(f); mv != model.MovementUnknown {
			l.Movement = mv
		}
		if g := normalize.GenderType(f); g != model.GenderUnknown {
			l.Gender = g
		}
	}
	if a := fields["附件狀況"]; a != "" {
		l.Description = normalize.CleanText(a)
		l.Year = normalize.ParseYear(a)
		l.HasBox, l.HasPapers = normalize.DetectBoxPapers(a)
		attrs["accessories"] = a
	}
	if l.Price == nil {
		if p, ok := normalize.ParsePrice(fields["售價"]); ok {
			l.Price = &p
		}
	}
	var photos []string
	doc.Find(`img[src*="product/product_picture/"], a[href*="product/product_picture/"]`).Each(func(_ int, s *goquery.Selection) {
		src, ok := s.Attr("src")
		if !ok {
			src, _ = s.Attr("href")
		}
		if src == "" || len(photos) >= 8 {
			return
		}
		u := crawler.AbsURL(doc, src)
		for _, p := range photos {
			if p == u {
				return
			}
		}
		photos = append(photos, u)
	})
	if len(photos) > 0 {
		l.ImageURLs = append(photos, l.ImageURLs...)
		if len(l.ImageURLs) > 8 {
			l.ImageURLs = l.ImageURLs[:8]
		}
	}
	l.Attributes, _ = json.Marshal(attrs)
}
