// Package bbc0804 crawls the forfeited-goods (流當品) watch categories of
// https://www.bbc0804.com.tw (仁川當舖, a pawn shop in Rende, Tainan; TWD).
//
// Site structure (verified 2026-09): an iyp.tw-hosted static site, UTF-8.
//   - /goods.html lists the categories in its side nav as <a href="rolex.html">勞力士男錶 運動款</a>;
//     watches are the categories whose name contains 錶 or a known brand (勞力士, 帝舵, OMEGA,
//     百達翡麗伯爵愛彼, LONGINES, ORIS, CHOPARD, 萬國 IWC, K金錶 鑽錶, 百大名錶…); jewellery, gold,
//     scooters, phones and bags are skipped. Each category is a single page (no pagination,
//     ~30 pieces at most; ~80 pieces in all).
//   - Cards: ul.product-list li a[href="product-detail-<id>.html"] img.photoSmall, h3
//     "勞力士 ROLEX 型號116503 Daytona 半金白迪 錶徑40mm 動力4130 保卡2021/JAN 國外AD $64.8萬",
//     span.desc "售價:$64.8萬" (萬 = ten thousand)
//   - Product page: #productDesc with labelled lines 型　　號 / 保卡年份 / 購買店家 / 錶　　徑 /
//     動力來源 / 材　　質 / 錶　　況 / 附　　件 (原盒1 保卡1 …), img.photoBig and extra photos.
package bbc0804

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

const baseURL = "https://www.bbc0804.com.tw/"

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "bbc0804" }

var (
	idRe      = regexp.MustCompile(`product-detail-(\d+)\.html`)
	catRe     = regexp.MustCompile(`^(?:\.\./)*([a-zA-Z0-9_-]+)\.html$`)
	priceRe   = regexp.MustCompile(`\$\s*([0-9][0-9,]*(?:\.\d+)?)\s*(萬)?`)
	tailRe    = regexp.MustCompile(`\s*\$\s*[0-9][0-9,]*(?:\.\d+)?\s*萬?\s*$`)
	refRe     = regexp.MustCompile(`型號\s*[:：]?\s*([A-Za-z0-9][A-Za-z0-9./-]*)`)
	yearRe    = regexp.MustCompile(`(?:保卡年份|保卡日期|保卡|年份)\s*[:：]?\s*((?:19|20)\d{2})`)
	labelRe   = regexp.MustCompile(`^([^：:]{1,6})[：:]\s*(.*)$`)
	nonWatch  = []string{"戒", "鍊", "鑽石", "珠", "翡翠", "黃金", "寶石", "機車", "手機", "包包", "其它", "墜", "貴金屬"}
	skipPages = map[string]bool{"goods": true, "about-us": true, "product": true, "knowledge": true, "news": true, "contact-us": true, "index": true}
)

// Category is a goods category page.
type Category struct {
	Slug string
	Name string
}

// Crawl reads every watch category and each product page.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	home, err := env.Fetcher.Doc(ctx, baseURL+"goods.html")
	if err != nil {
		return fmt.Errorf("goods page: %w", err)
	}
	cats := Categories(home)
	if len(cats) == 0 {
		return fmt.Errorf("no watch categories in the goods nav (markup changed?)")
	}
	env.Log.Info().Int("categories", len(cats)).Msg("bbc0804 categories")
	seen := map[string]bool{}
	for _, cat := range cats {
		doc, err := env.Fetcher.Doc(ctx, baseURL+cat.Slug+".html")
		if err != nil {
			return fmt.Errorf("category %s: %w", cat.Slug, err)
		}
		for _, l := range ParseList(doc, cat.Name) {
			if seen[l.ExternalID] {
				continue
			}
			seen[l.ExternalID] = true
			if detail, err := env.Fetcher.Doc(ctx, l.URL); err != nil {
				env.Log.Warn().Err(err).Str("id", l.ExternalID).Msg("product page failed; using list data")
			} else {
				ApplyDetail(&l, detail)
			}
			if err := emit(l); err != nil {
				return err
			}
		}
	}
	return nil
}

// Categories returns the watch categories linked from the goods page nav.
func Categories(doc *goquery.Document) []Category {
	var out []Category
	seen := map[string]bool{}
	doc.Find(`a[href$=".html"]`).Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		m := catRe.FindStringSubmatch(strings.TrimSpace(href))
		name := crawler.Text(a)
		if m == nil || name == "" || seen[m[1]] || skipPages[m[1]] || strings.Contains(href, "product-detail") {
			return
		}
		for _, bad := range nonWatch {
			if strings.Contains(name, bad) && !strings.Contains(name, "錶") {
				return
			}
		}
		if !strings.Contains(name, "錶") && normalize.DetectBrand(name) == nil {
			return
		}
		seen[m[1]] = true
		out = append(out, Category{Slug: m[1], Name: name})
	})
	return out
}

// ParseList extracts the cards of a category page. Exported for fixture tests.
func ParseList(doc *goquery.Document, categoryName string) []model.Listing {
	var out []model.Listing
	seen := map[string]bool{}
	doc.Find(`ul.product-list li a[href*="product-detail-"]`).Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		m := idRe.FindStringSubmatch(href)
		if m == nil || seen[m[1]] {
			return
		}
		raw := normalize.CleanText(strings.ReplaceAll(crawler.Text(a.Find("h3").First()), " ", " "))
		if raw == "" {
			raw, _ = a.Attr("title")
			raw = normalize.CleanText(raw)
		}
		title := strings.TrimSpace(tailRe.ReplaceAllString(raw, ""))
		if title == "" {
			return
		}
		seen[m[1]] = true
		l := model.Listing{
			ExternalID:      m[1],
			URL:             crawler.AbsURL(doc, href),
			Title:           title,
			Currency:        "TWD",
			SellerType:      "dealer",
			SellerName:      "仁川當舖",
			LocationCountry: "TW",
			LocationCity:    "Tainan",
			Condition:       model.ConditionGood, // a pawn shop: pre-owned unless the title says 全新
		}
		if strings.Contains(title, "全新") {
			l.Condition = model.ConditionNew
		}
		if b := normalize.DetectBrand(title, categoryName); b != nil {
			l.BrandSlug, l.BrandName = b.Slug, b.Name
		}
		if rm := refRe.FindStringSubmatch(title); rm != nil {
			l.ReferenceNumber = strings.ToUpper(rm[1])
		}
		if ym := yearRe.FindStringSubmatch(title); ym != nil {
			if y, err := strconv.Atoi(ym[1]); err == nil {
				l.Year = &y
			}
		}
		l.CaseDiameterMM = normalize.ParseDiameter(title)
		if p, ok := ParsePrice(crawler.Text(a.Find("span.desc").First())); ok {
			l.Price = &p
		} else if p, ok := ParsePrice(raw); ok {
			l.Price = &p
		}
		if src, ok := a.Find("img").First().Attr("src"); ok && src != "" {
			l.ImageURLs = []string{stripQuery(crawler.AbsURL(doc, src))}
		}
		attrs := map[string]any{"category": categoryName}
		// titles carry hints such as "保卡2022/JUN" or "裸錶無單"; the detail page refines them
		l.HasBox, l.HasPapers = normalize.DetectBoxPapers(title)
		l.Attributes, _ = json.Marshal(attrs)
		out = append(out, l)
	})
	return out
}

// ParsePrice reads "$64.8萬" (648,000) or "$58000".
func ParsePrice(s string) (float64, bool) {
	m := priceRe.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", ""), 64)
	if err != nil || v <= 0 {
		return 0, false
	}
	if m[2] == "萬" {
		v *= 10000
	}
	return v, true
}

// ApplyDetail fills box/papers, material, movement, condition notes and photos from a product
// page. Exported for fixture tests.
func ApplyDetail(l *model.Listing, doc *goquery.Document) {
	desc := doc.Find("#productDesc").First()
	var lines []string
	desc.Find("div[dir], p").Each(func(_ int, d *goquery.Selection) {
		if d.Children().Filter("div[dir], p").Length() > 0 {
			return // container of other lines
		}
		if t := normalize.CleanText(strings.ReplaceAll(strings.ReplaceAll(crawler.Text(d), " ", " "), "　", "")); t != "" {
			lines = append(lines, t)
		}
	})
	if len(lines) == 0 {
		lines = strings.Split(strings.ReplaceAll(crawler.Text(desc), "　", ""), "\n")
	}
	fields := map[string]string{}
	for _, line := range lines {
		if m := labelRe.FindStringSubmatch(line); m != nil {
			key := strings.ReplaceAll(m[1], " ", "")
			if _, dup := fields[key]; !dup {
				fields[key] = strings.TrimSpace(m[2])
			}
		}
	}
	var attrs map[string]any
	_ = json.Unmarshal(l.Attributes, &attrs)
	if attrs == nil {
		attrs = map[string]any{}
	}
	pick := func(keys ...string) string {
		for _, k := range keys {
			if v := fields[k]; v != "" {
				return v
			}
		}
		return ""
	}
	if acc := pick("附件", "配件"); acc != "" {
		l.HasBox, l.HasPapers = normalize.DetectAccessories(acc) // "原盒1 保單1 說明書2 吊牌1"
		attrs["accessories"] = acc
	}
	if cond := pick("錶況", "目前錶況"); cond != "" {
		attrs["conditionNote"] = cond
		l.Description = strings.TrimSpace("錶況：" + cond)
	}
	if mat := pick("材質"); mat != "" {
		l.CaseMaterial = mat
	}
	if mv := pick("動力來源", "動力"); mv != "" {
		attrs["movement"] = mv
		if m := normalize.MovementType(mv); m != model.MovementUnknown {
			l.Movement = m
		}
	}
	if l.ReferenceNumber == "" {
		if ref := pick("型號"); ref != "" {
			if f := strings.Fields(ref); len(f) > 0 {
				l.ReferenceNumber = strings.ToUpper(f[0])
			}
		}
	}
	if l.Year == nil {
		if y := normalize.ParseYear(pick("保卡年份", "保卡日期", "年份")); y != nil {
			l.Year = y
		}
	}
	if shop := pick("購買店家", "購買地點"); shop != "" {
		attrs["boughtFrom"] = shop
	}
	var photos []string
	if src, ok := doc.Find("img.photoBig").First().Attr("src"); ok && src != "" {
		photos = append(photos, stripQuery(crawler.AbsURL(doc, src)))
	}
	desc.Find("img[src]").Each(func(_ int, img *goquery.Selection) {
		src, _ := img.Attr("src")
		if src != "" && len(photos) < 8 {
			photos = append(photos, stripQuery(crawler.AbsURL(doc, src)))
		}
	})
	if len(photos) > 0 {
		l.ImageURLs = photos
	}
	l.Attributes, _ = json.Marshal(attrs)
}

func stripQuery(u string) string {
	if i := strings.Index(u, "?"); i > 0 {
		return u[:i]
	}
	return u
}
