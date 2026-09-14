// Package jdpawn crawls the watch section of https://www.jdpawn.com.tw (久大御典品, a pawn shop
// group around Taichung; TWD).
//
// Site structure (verified 2026-09): ASP.NET MVC, UTF-8, server-rendered.
//   - List: /luxury/Stuffs/Watch?page=<n> (16 pieces per page, ~200 pieces / 13 pages;
//     .pagination .PagedList-pageCountAndLocation reads "第1頁/共13頁"). Each page embeds every
//     photo as a base64 data URI and weighs 4–5 MB, hence the Fetcher's body limit.
//   - Each piece is a Bootstrap modal div.modal.quick-view#mdl<recid> holding div.media-body:
//     h2 "CARTIER錶", h3 "126,000", h4 "商品編號：<span>T848</span>", h4 "商品所在店：<span
//     title='電話:…地址:41266 台中市大里區…'>民生", and #details<n> li lines 型式 / 機芯 / 錶徑 /
//     功能 / 附註 (盒證 = box + papers) / 錶殼 / 錶帶 / 錶扣 / 錶況 / 錶面 / 新錶訂價 / 參考市價.
//   - Product page: /luxury/Stuffs/SingleProduct/<code>; photo: /luxury/StuffPictures/<code>.jpg
package jdpawn

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

const (
	baseURL  = "https://www.jdpawn.com.tw"
	pageSize = 16
)

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "jdpawn" }

var (
	pagesRe   = regexp.MustCompile(`共\s*(\d+)\s*頁`)
	cityRe    = regexp.MustCompile(`地址[:：]\s*\d*\s*([^\s<]{2,3}[市縣])`)
	labelRe   = regexp.MustCompile(`^([^：:]{1,6})[：:]\s*(.*)$`)
	diaRe     = regexp.MustCompile(`(\d{2,3}(?:\.\d+)?)\s*mm`)
	cityNames = map[string]string{"台中市": "Taichung", "臺中市": "Taichung", "台北市": "Taipei", "臺北市": "Taipei", "新北市": "New Taipei", "桃園市": "Taoyuan", "新竹市": "Hsinchu", "台南市": "Tainan", "臺南市": "Tainan", "高雄市": "Kaohsiung", "彰化縣": "Changhua", "南投縣": "Nantou"}
)

// Crawl pages through the watch list until the last page.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	seen := map[string]bool{}
	for page := 1; page <= env.MaxPages; page++ {
		u := baseURL + "/luxury/Stuffs/Watch"
		if page > 1 {
			u = fmt.Sprintf("%s?page=%d", u, page)
		}
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
		if fresh == 0 || page >= Pages(doc) {
			return nil
		}
	}
	return nil
}

// Pages returns the page count printed in the pager, or 1.
func Pages(doc *goquery.Document) int {
	if m := pagesRe.FindStringSubmatch(crawler.Text(doc.Find(".PagedList-pageCountAndLocation").First())); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil && n > 0 {
			return n
		}
	}
	return 1
}

// ParseList extracts the pieces from a list page's quick-view modals. Exported for fixture tests.
func ParseList(doc *goquery.Document) []model.Listing {
	var out []model.Listing
	seen := map[string]bool{}
	doc.Find("div.modal.quick-view").Each(func(_ int, modal *goquery.Selection) {
		body := modal.Find(".media-body").First()
		if body.Length() == 0 {
			return
		}
		code := ""
		body.Find("h4").Each(func(_ int, h *goquery.Selection) {
			if strings.Contains(crawler.Text(h), "商品編號") {
				code = strings.TrimSpace(crawler.Text(h.Find("span").First()))
			}
		})
		if code == "" || seen[code] {
			return
		}
		fields := map[string]string{}
		modal.Find(".tab-pane li").Each(func(_ int, li *goquery.Selection) {
			t := crawler.Text(li)
			if m := labelRe.FindStringSubmatch(t); m != nil {
				if _, dup := fields[m[1]]; !dup {
					fields[m[1]] = strings.TrimSpace(m[2])
				}
			}
		})
		brandLine := strings.TrimSuffix(strings.TrimSpace(strings.ReplaceAll(crawler.Text(body.Find("h2").First()), " ", " ")), "錶")
		title := normalize.CleanText(fields["型式"])
		if title == "" {
			title = normalize.CleanText(brandLine)
		}
		if title == "" {
			return
		}
		seen[code] = true
		l := model.Listing{
			ExternalID:      code,
			URL:             baseURL + "/luxury/Stuffs/SingleProduct/" + code,
			Title:           title,
			Currency:        "TWD",
			SellerType:      "dealer",
			SellerName:      "久大御典品",
			LocationCountry: "TW",
			Condition:       model.ConditionGood, // a pawn shop: everything is pre-owned
			ImageURLs:       []string{baseURL + "/luxury/StuffPictures/" + code + ".jpg"},
		}
		if b := normalize.DetectBrand(brandLine, title); b != nil {
			l.BrandSlug, l.BrandName = b.Slug, b.Name
		}
		if p, ok := normalize.ParsePrice(crawler.Text(body.Find("h3").First())); ok {
			l.Price = &p
		}
		if mv := fields["機芯"]; mv != "" {
			l.Movement = normalize.MovementType(mv)
			if l.Movement == model.MovementUnknown && strings.Contains(mv, "上鍊") {
				l.Movement = model.MovementManual
			}
		}
		if m := diaRe.FindStringSubmatch(fields["錶徑"]); m != nil { // "42.00mm x 42.00mm" → the first figure
			if v, err := strconv.ParseFloat(m[1], 64); err == nil && v >= 15 && v <= 70 {
				l.CaseDiameterMM = &v
			}
		}
		if note := fields["附註"]; note != "" {
			box, papers := strings.Contains(note, "盒"), strings.Contains(note, "證") || strings.Contains(note, "卡")
			l.HasBox, l.HasPapers = &box, &papers
		}
		if c := fields["錶殼"]; c != "" {
			l.CaseMaterial = normalize.CleanText(strings.TrimSuffix(strings.TrimSuffix(strings.Fields(c)[0], "材質錶殼"), "錶殼"))
		}
		if dial := fields["錶面"]; dial != "" {
			l.DialColor = normalize.DialColor(dial)
		}
		attrs := map[string]any{}
		for k, v := range fields {
			switch k {
			case "型式", "機芯", "錶徑", "功能", "附註", "錶殼", "錶帶", "錶扣", "錶況", "錶面", "錶節總數", "新錶訂價", "參考市價":
				attrs[k] = v
			}
		}
		var descParts []string
		for _, k := range []string{"錶況", "功能", "錶殼", "錶帶", "錶扣"} {
			if v := fields[k]; v != "" {
				descParts = append(descParts, k+"："+v)
			}
		}
		l.Description = strings.Join(descParts, " ")
		body.Find("h4").Each(func(_ int, h *goquery.Selection) {
			if !strings.Contains(crawler.Text(h), "商品所在店") {
				return
			}
			sp := h.Find("span[title]").First()
			if store := strings.TrimSpace(strings.ReplaceAll(strings.Split(crawler.Text(sp), " ")[0], " ", "")); store != "" {
				attrs["store"] = store
			}
			if t, ok := sp.Attr("title"); ok {
				if m := cityRe.FindStringSubmatch(t); m != nil {
					if en, ok := cityNames[m[1]]; ok {
						l.LocationCity = en
					} else {
						l.LocationCity = m[1]
					}
				}
			}
		})
		l.Attributes, _ = json.Marshal(attrs)
		out = append(out, l)
	})
	return out
}
