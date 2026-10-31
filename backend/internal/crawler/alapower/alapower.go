// Package alapower parses the classic ASP shop system built by alapower.com.tw (歐耶資訊) that
// many Taiwanese watch dealers use (Hourstack, RD Watch, 鴻昇名錶, 金宸名錶, 大眾名錶, 好時計,
// 永生／永久). Site adapters only supply the base URL, seller name and the template variant.
//
// Structure (verified 2026-09):
//   - Brand categories: /product.asp?cat=<n>          (links on the home page nav)
//   - Pagination:       /product.asp?cat=<n>&page=<p>
//   - Product cards:    a fixed-width <td> per product containing a[href^="product_open.asp?ID="],
//     img[src*="product/product_big/"], a.a_table_list_txt (brand line and/or
//     title), "編號 :XXX" and span.shopping_Price (TWD, with or without commas)
//   - Product page:     /product_open.asp?ID=<id> with 款式/材質/錶徑/功能/附件 rows (td.a_txt_744)
//
// Variants seen 2026-10 (through a page reader only, see docs/CRAWLERS.md): the list path can be
// products.asp + products_open.asp (egps), the id parameter lowercase (product_open.asp?id=, with
// /product/product_BIG/ photos: goodtimezone), the pager can use index=<p> instead of page=<p>
// (369rolexwatch, goodtimezone) and the price can be plain "售價 : 368,000元" text without
// span.shopping_Price. Cards are therefore located from their product link, not from a class.
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
	// SkipCategories are substrings of menu entries that hold no watches (straps, accessories);
	// those categories are not crawled.
	SkipCategories []string
	// ListPath is the category list script: "product.asp" (default) or "products.asp".
	ListPath string
	// CategoryQuery is appended to every category URL (egps needs "&type=open").
	CategoryQuery string
	// PageParams are the pager query parameters sent for page > 1; default ["page"]. Sites whose
	// pager links use index= get both, so whichever one the script reads takes effect.
	PageParams []string
	// Categories, when set, replaces home-page discovery (sites whose home page also links
	// jewellery categories under the same scheme).
	Categories []Category
	// SkipTitles are title substrings of cards that are not for sale (shop announcements).
	SkipTitles []string
}

// Category is one brand / gender category of the shop.
type Category struct{ ID, Name string }

var (
	idRe     = regexp.MustCompile(`(?i)products?_open\.asp\?id=(\d+)`)
	catRe    = regexp.MustCompile(`(?i)products?\.asp\?cat=(\d+)`)
	snRe     = regexp.MustCompile(`編號\s*[:：]\s*([A-Za-z0-9-]+)`)
	priceRe  = regexp.MustCompile(`(?:售價|特價|價格)\s*[:：]?\s*(?:NT\$|NTD\.?|\$)?\s*([\d,]{4,})`)
	stateRe  = regexp.MustCompile(`錶況\s*[:：]\s*(\S+)`)
	diaRe    = regexp.MustCompile(`錶徑\s*[:：]\s*(\S+)`)
	photoRe  = regexp.MustCompile(`(?i)product/product_(big|small)/`)
	bigRe    = regexp.MustCompile(`(?i)product_big/`)
	pageNoRe = regexp.MustCompile(`(?:page|index)=(\d+)`)
)

func (s Site) listPath() string {
	if s.ListPath != "" {
		return s.ListPath
	}
	return "product.asp"
}

func (s Site) categoryURL(id string, page int) string {
	u := fmt.Sprintf("%s%s?cat=%s%s", s.BaseURL, s.listPath(), id, s.CategoryQuery)
	params := s.PageParams
	if len(params) == 0 {
		params = []string{"page"}
	}
	for _, p := range params {
		u += fmt.Sprintf("&%s=%d", p, page)
	}
	return u
}

// Crawl walks every brand category page by page.
func (s Site) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	cats := s.Categories
	if len(cats) == 0 {
		home, err := env.Fetcher.Doc(ctx, s.BaseURL)
		if err != nil {
			return fmt.Errorf("home: %w", err)
		}
		cats = s.DiscoverCategories(home, env)
	}
	if len(cats) == 0 {
		return fmt.Errorf("no categories found on home page (markup changed?)")
	}
	env.Log.Info().Int("categories", len(cats)).Msg("categories discovered")

	seen := map[string]bool{}
	for _, c := range cats {
		for page := 1; page <= env.MaxPages; page++ {
			doc, err := env.Fetcher.Doc(ctx, s.categoryURL(c.ID, page))
			if err != nil {
				return fmt.Errorf("category %s page %d: %w", c.ID, page, err)
			}
			items := s.ParseList(doc, c.Name)
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

// DiscoverCategories reads the brand categories from the home page menu, skipping straps,
// accessories and jewellery.
func (s Site) DiscoverCategories(home *goquery.Document, env *crawler.Env) []Category {
	var cats []Category
	seen := map[string]bool{}
	home.Find(`a[href]`).Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		m := catRe.FindStringSubmatch(href)
		if m == nil || seen[m[1]] {
			return
		}
		seen[m[1]] = true
		name := strings.Trim(crawler.Text(a), "‧· ")
		if s.skipCategory(name) {
			if env != nil {
				env.Log.Info().Str("category", name).Msg("skipping non-watch category")
			}
			return
		}
		if reason := normalize.ExclusionReason(name); reason != "" {
			// strap makers and the like have their own menu entry: skip the whole category
			if env != nil {
				env.Log.Info().Str("category", name).Str("reason", reason).Msg("skipping non-watch category")
			}
			return
		}
		cats = append(cats, Category{m[1], name})
	})
	return cats
}

func (s Site) skipCategory(name string) bool {
	for _, m := range s.SkipCategories {
		if strings.Contains(name, m) {
			return true
		}
	}
	return false
}

func hasNextPage(doc *goquery.Document, current int) bool {
	next := false
	doc.Find(`a[href*="page="], a[href*="index="]`).Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		if m := pageNoRe.FindStringSubmatch(href); m != nil {
			var p int
			fmt.Sscanf(m[1], "%d", &p) //nolint:errcheck
			if p > current {
				next = true
			}
		}
	})
	return next
}

// ParseList extracts listings from a category page. Cards are located from their product link
// (see crawler.LinkCards) so column widths, class names and id-parameter case do not matter.
func (s Site) ParseList(doc *goquery.Document, categoryName string) []model.Listing {
	var out []model.Listing
	for _, c := range crawler.LinkCards(doc, idRe) {
		card := c.Card
		text := crawler.Text(card)
		priceEl := card.Find(".shopping_Price").First()
		if priceEl.Length() == 0 && !strings.Contains(text, "售價") && !strings.Contains(text, "價格") {
			continue // a link in a sidebar or banner, not a product card
		}
		href, _ := card.Find("a[href]").FilterFunction(func(_ int, a *goquery.Selection) bool {
			h, _ := a.Attr("href")
			m := idRe.FindStringSubmatch(h)
			return m != nil && m[1] == c.ID
		}).First().Attr("href")
		l := model.Listing{
			ExternalID: c.ID,
			URL:        crawler.AbsURL(doc, href),
			Currency:   "TWD",
			SellerType: "dealer",
			SellerName: s.SellerName,
			Condition:  model.ConditionUnknown,
		}
		// The longest product-link text is the title; a shorter a.a_table_list_txt (if any) is the
		// brand line.
		l.Title = normalize.CleanText(crawler.LongestLinkText(card, idRe, c.ID))
		var brandLine string
		card.Find("a.a_table_list_txt").Each(func(_ int, a *goquery.Selection) {
			if t := normalize.CleanText(crawler.Text(a)); t != "" && t != l.Title {
				brandLine = t
			}
		})
		l.Title = s.stripBoilerplate(l.Title)
		if l.Title == "" || s.skipTitle(l.Title) {
			continue
		}
		if b := normalize.DetectBrand(brandLine, categoryName, l.Title); b != nil {
			l.BrandSlug, l.BrandName = b.Slug, b.Name
		}
		if imgs := crawler.CardImages(doc, card, photoRe); len(imgs) > 0 {
			l.ImageURLs = imgs[:1]
		}
		if priceEl.Length() > 0 {
			if p, ok := normalize.ParsePrice(crawler.Text(priceEl)); ok {
				l.Price = &p
			}
		} else if m := priceRe.FindStringSubmatch(text); m != nil {
			if p, ok := normalize.ParsePrice(m[1]); ok {
				l.Price = &p
			}
		}
		attrs := map[string]any{"category": categoryName}
		if sn := snRe.FindStringSubmatch(text); sn != nil {
			attrs["stockNumber"] = sn[1]
		}
		state := ""
		if m := stateRe.FindStringSubmatch(text); m != nil {
			state = m[1]
			attrs["condition"] = state
		}
		if m := diaRe.FindStringSubmatch(text); m != nil {
			l.CaseDiameterMM = normalize.ParseDiameter(m[1])
		}
		l.Attributes, _ = json.Marshal(attrs)
		switch {
		case normalize.Condition(state) != model.ConditionUnknown:
			l.Condition = normalize.Condition(state)
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
	}
	return out
}

func (s Site) skipTitle(title string) bool {
	for _, m := range s.SkipTitles {
		if strings.Contains(title, m) {
			return true
		}
	}
	return false
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
	doc.Find(`img[src]`).Each(func(_ int, img *goquery.Selection) {
		if src, _ := img.Attr("src"); bigRe.MatchString(src) {
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
