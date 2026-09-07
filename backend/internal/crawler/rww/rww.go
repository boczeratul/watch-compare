// Package rww indexes https://rwwwatch.com (RWW Watch, Hong Kong; HKD).
//
// The storefront under /app/web/ is a jQuery app whose data comes from one JSON endpoint
// (verified 2026-09): POST /app/php/data.php with {"action": …, "data": {…}}.
//   - felistCategory {catid:"", lang:"zh"} lists the top-level categories; the four whose name
//     contains 手錶 (new/used Rolex, new/used other brands) are crawled, which leaves out bags.
//   - felistproduct {catid, lang:"zh", curr:"HKD", recordperpage:100, page, sort:"seq, moddt desc",
//     sortby:""} pages a category (ttlpage in the envelope). lang and curr must be real values
//     or the query matches nothing.
//   - Records: linkid (id), prname "Daytona系列 40mm 全新 Rolex 126500LN 黑面 (2026年)(尖沙咀店)",
//     prdesc HTML with 型號 / 配件 / 狀態 / 銷售地點 lines, minfinprice / price ("HK$ 242800", "" when
//     the price is on request), img[{path, filename, thumbnail}]. Sold pieces stay listed with an
//     "(已售)" prefix and are skipped.
//
// robots.txt allows /app/web/ only; the data endpoint is what that page itself calls.
package rww

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

const (
	baseURL  = "https://rwwwatch.com"
	dataURL  = baseURL + "/app/php/data.php"
	pageSize = 100
)

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "rww" }

// CategoryResponse mirrors felistCategory.
type CategoryResponse struct {
	Records []struct {
		CatID   string `json:"catid"`
		CatName string `json:"catname"`
		Status  string `json:"status"`
	} `json:"records"`
}

// ListResponse mirrors felistproduct.
type ListResponse struct {
	Records       []Record `json:"records"`
	Page          int      `json:"page"`
	TotalPages    int      `json:"ttlpage"`
	RecordPerPage int      `json:"recordperpage"`
	Status        string   `json:"status"`
}

// Record is one product.
type Record struct {
	LinkID      string `json:"linkid"`
	Name        string `json:"prname"`
	Desc        string `json:"prdesc"`
	Price       string `json:"price"`       // "HK$ 242800" or ""
	MinFinPrice string `json:"minfinprice"` // numeric string, "0" when unpriced
	MaxFinPrice string `json:"maxfinprice"`
	ModDT       string `json:"moddt"`
	Img         []struct {
		Path      string `json:"path"`
		Filename  string `json:"filename"`
		Thumbnail string `json:"thumbnail"`
	} `json:"img"`
	Tag []struct {
		Desc string `json:"gdesc"`
	} `json:"tag"`
}

var (
	soldRe    = regexp.MustCompile(`已售|已售出|sold`)
	seriesRe  = regexp.MustCompile(`^\s*(.+?)系列`)
	usedPctRe = regexp.MustCompile(`二手\s*(\d{2,3})\s*%\s*新`)
	storeRe   = regexp.MustCompile(`[（(]([^()（）]*店)[)）]`)
	tagRe     = regexp.MustCompile(`<[^>]+>`)
	modelRe   = regexp.MustCompile(`型號[:：]\s*([^\n]+)`)
	refTokRe  = regexp.MustCompile(`[A-Z0-9][A-Z0-9./-]{3,}`)
	descLine  = regexp.MustCompile(`(配件|狀態|銷售地點)[:：]\s*([^\n]+)`)
)

// Crawl discovers the watch categories and pages through each.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	body, err := post(ctx, env, map[string]any{"action": "felistCategory", "data": map[string]any{"catid": "", "lang": "zh"}})
	if err != nil {
		return fmt.Errorf("categories: %w", err)
	}
	cats, err := ParseCategories(body)
	if err != nil {
		return fmt.Errorf("categories: %w", err)
	}
	if len(cats) == 0 {
		return fmt.Errorf("no watch categories in felistCategory (site changed?)")
	}
	env.Log.Info().Int("categories", len(cats)).Msg("rww categories")
	seen := map[string]bool{}
	for _, cat := range cats {
		for page := 1; page <= env.MaxPages; page++ {
			body, err := post(ctx, env, map[string]any{"action": "felistproduct", "data": map[string]any{
				"page": page, "catid": cat, "lang": "zh", "curr": "HKD", "recordperpage": pageSize,
				"sort": "seq, moddt desc", "sortby": "",
			}})
			if err != nil {
				return fmt.Errorf("category %s page %d: %w", cat, page, err)
			}
			res, err := ParseList(body)
			if err != nil {
				return fmt.Errorf("category %s page %d: %w", cat, page, err)
			}
			if len(res.Records) == 0 {
				break
			}
			for _, r := range res.Records {
				l, ok := ToListing(r)
				if !ok || seen[l.ExternalID] {
					continue
				}
				seen[l.ExternalID] = true
				if err := emit(l); err != nil {
					return err
				}
			}
			if page >= res.TotalPages || len(res.Records) < pageSize {
				break
			}
		}
	}
	return nil
}

func post(ctx context.Context, env *crawler.Env, payload any) ([]byte, error) {
	b, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, dataURL, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", baseURL+"/app/web/product_list.php")
	resp, err := env.Fetcher.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("data.php: http %d", resp.StatusCode)
	}
	return body, nil
}

// ParseCategories returns the ids of the published categories whose name marks them as watches.
func ParseCategories(body []byte) ([]string, error) {
	var res CategoryResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	var out []string
	for _, c := range res.Records {
		if c.CatID == "" || c.CatID == "ALL" || (c.Status != "" && c.Status != "published") {
			continue
		}
		if strings.Contains(c.CatName, "手錶") || strings.Contains(c.CatName, "手表") {
			out = append(out, c.CatID)
		}
	}
	return out, nil
}

// ParseList decodes a felistproduct page. Exported for fixture tests.
func ParseList(body []byte) (*ListResponse, error) {
	var res ListResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &res, nil
}

// ToListing converts a record; ok is false for sold pieces.
func ToListing(r Record) (model.Listing, bool) {
	name := normalize.CleanText(r.Name)
	if r.LinkID == "" || name == "" || soldRe.MatchString(strings.ToLower(name)) {
		return model.Listing{}, false
	}
	l := model.Listing{
		ExternalID:      r.LinkID,
		URL:             baseURL + "/app/web/product_detail.php?linkid=" + r.LinkID,
		Title:           name,
		Currency:        "HKD",
		SellerType:      "dealer",
		SellerName:      "RWW Watch",
		LocationCountry: "HK",
		LocationCity:    "Hong Kong",
		Condition:       condition(name),
	}
	if m := seriesRe.FindStringSubmatch(name); m != nil {
		l.Model = strings.TrimSpace(m[1])
	}
	if v, err := strconv.ParseFloat(strings.TrimSpace(r.MinFinPrice), 64); err == nil && v > 0 {
		l.Price = &v
	} else if p, ok := normalize.ParsePrice(r.Price); ok {
		l.Price = &p
	}
	desc := normalize.CleanText(html.UnescapeString(tagRe.ReplaceAllString(strings.ReplaceAll(r.Desc, "</p>", "\n"), " ")))
	// the description ends in a payment-terms boilerplate introduced by "***"
	if i := strings.Index(desc, "***"); i > 0 {
		desc = strings.TrimSpace(desc[:i])
	}
	l.Description = desc
	modelLine := ""
	if m := modelRe.FindStringSubmatch(strings.ReplaceAll(desc, " 配件", "\n配件")); m != nil {
		modelLine = strings.TrimSpace(m[1])
	}
	if b := normalize.DetectBrand(modelLine, name); b != nil {
		l.BrandSlug, l.BrandName = b.Slug, b.Name
	}
	if modelLine != "" {
		// "Rolex 50529" / "Patek Philippe 5164R-001": the reference is the alphanumeric token after the brand
		up := strings.ToUpper(modelLine)
		for _, tok := range refTokRe.FindAllString(up, -1) {
			if strings.ContainsAny(tok, "0123456789") {
				l.ReferenceNumber = tok
				break
			}
		}
	}
	for _, im := range r.Img {
		if im.Path != "" && im.Filename != "" && len(l.ImageURLs) < 8 {
			l.ImageURLs = append(l.ImageURLs, im.Path+im.Filename)
		}
	}
	attrs := map[string]any{"modifiedAt": r.ModDT}
	if m := storeRe.FindStringSubmatch(name); m != nil {
		attrs["store"] = m[1]
	}
	for _, m := range descLine.FindAllStringSubmatch(strings.ReplaceAll(desc, " 配件", "\n配件"), -1) {
		key := map[string]string{"配件": "accessories", "狀態": "state", "銷售地點": "storeLocation"}[m[1]]
		attrs[key] = strings.TrimSpace(m[2])
		if m[1] == "配件" {
			l.HasBox, l.HasPapers = normalize.DetectBoxPapers(m[2])
		}
	}
	var tags []string
	for _, t := range r.Tag {
		if t.Desc != "" {
			tags = append(tags, t.Desc)
		}
	}
	if len(tags) > 0 {
		attrs["tags"] = tags
	}
	l.Attributes, _ = json.Marshal(attrs)
	return l, true
}

// condition reads the dealer's grade out of the name: 全新 (new), 全新未用品 (unworn), 二手 N%新.
func condition(name string) model.Condition {
	switch {
	case strings.Contains(name, "未用") || strings.Contains(name, "未使用"):
		return model.ConditionUnworn
	case strings.Contains(name, "全新"):
		return model.ConditionNew
	}
	if m := usedPctRe.FindStringSubmatch(name); m != nil {
		pct, _ := strconv.Atoi(m[1])
		switch {
		case pct >= 98:
			return model.ConditionVeryGood
		case pct >= 90:
			return model.ConditionGood
		default:
			return model.ConditionFair
		}
	}
	if strings.Contains(name, "二手") {
		return model.ConditionGood
	}
	return model.ConditionUnknown
}
