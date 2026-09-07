// Package kenwatches crawls https://kenwatches.com (Ken's Watches 名錶廊, Hong Kong; HKD).
//
// Site structure (verified 2026-09): Next.js with server-side rendering; the list page embeds
// its data in <script id="__NEXT_DATA__">, so no JSON API is needed.
//   - List: /watch?page=<n> (20 items per page, newest first; pageProps.count is the total)
//   - Items (pageProps.watchItems[]): _id (product page /watch/<_id>), itemId, watchBrand.name,
//     model, modelRefNumber, sizeRemark "(39mm)", watchMovement.description, year, status STOCK,
//     conditions [BOX, PAPER, NEW], photos (relative to the kenwatches-storage bucket), store.name,
//     watchSize (id resolved through pageProps.filterOption.watchSizes), and the price: cash
//     (HKD) with listedPrice as the fallback; both null means price on request.
package kenwatches

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

const (
	baseURL  = "https://kenwatches.com"
	photoURL = "https://storage.googleapis.com/kenwatches-storage/"
	pageSize = 20
)

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "kenwatches" }

var nextDataRe = regexp.MustCompile(`(?s)<script[^>]*id="__NEXT_DATA__"[^>]*>(.*?)</script>`)

// Localized is the {"en-US": …, "zh-TW": …} shape the site uses for names.
type Localized map[string]string

// En returns the English value, or any value when English is missing.
func (l Localized) En() string {
	if v := strings.TrimSpace(l["en-US"]); v != "" {
		return v
	}
	for _, v := range l {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// Option is an entry of a filterOption lookup table.
type Option struct {
	ID          string    `json:"_id"`
	Name        Localized `json:"name"`
	Description Localized `json:"description"`
}

// Page mirrors pageProps of /watch.
type Page struct {
	CurrentPage int    `json:"currentPage"`
	Count       int    `json:"count"`
	WatchItems  []Item `json:"watchItems"`
	FilterOpt   struct {
		WatchSizes     []Option `json:"watchSizes"`
		WatchMovements []Option `json:"watchMovements"`
		Stores         []Option `json:"stores"`
	} `json:"filterOption"`
}

// Item is one stock record.
type Item struct {
	ID             string                   `json:"_id"`
	ItemID         int64                    `json:"itemId"`
	Type           string                   `json:"type"`
	Status         string                   `json:"status"`
	IsPublished    bool                     `json:"isPublished"`
	Conditions     []string                 `json:"conditions"`
	Behaviours     []string                 `json:"behaviours"`
	Photos         []string                 `json:"photos"`
	MainPhoto      string                   `json:"mainPhoto"`
	WatchBrand     struct{ Name Localized } `json:"watchBrand"`
	Model          Localized                `json:"model"`
	ModelRefNumber string                   `json:"modelRefNumber"`
	SizeRemark     Localized                `json:"sizeRemark"`
	Remark         Localized                `json:"remark"`
	WatchSize      json.RawMessage          `json:"watchSize"` // id string or {_id,name}
	WatchMovement  json.RawMessage          `json:"watchMovement"`
	Year           string                   `json:"year"`
	PaymentType    string                   `json:"paymentType"`
	Cash           *float64                 `json:"cash"`
	ListedPrice    *float64                 `json:"listedPrice"`
	PriceRemark    *string                  `json:"priceRemark"`
	Material       *string                  `json:"material"`
	Store          struct {
		Name Localized `json:"name"`
	} `json:"store"`
	PublishedDate string `json:"publishedDate"`
}

// Crawl pages through the list until the total is reached.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	seen := map[string]bool{}
	for page := 1; page <= env.MaxPages; page++ {
		u := baseURL + "/watch"
		if page > 1 {
			u = fmt.Sprintf("%s/watch?page=%d", baseURL, page)
		}
		body, err := env.Fetcher.Get(ctx, u)
		if err != nil {
			return fmt.Errorf("page %d: %w", page, err)
		}
		p, err := ParsePage(body)
		if err != nil {
			return fmt.Errorf("page %d: %w", page, err)
		}
		if len(p.WatchItems) == 0 {
			if page == 1 {
				return fmt.Errorf("no items on first page (page data changed?)")
			}
			return nil
		}
		for _, it := range p.WatchItems {
			l, ok := ToListing(it, p)
			if !ok || seen[l.ExternalID] {
				continue
			}
			seen[l.ExternalID] = true
			if err := emit(l); err != nil {
				return err
			}
		}
		if len(p.WatchItems) < pageSize || page*pageSize >= p.Count {
			return nil
		}
	}
	return nil
}

// ParsePage extracts pageProps from a rendered list page. Exported for fixture tests.
func ParsePage(body []byte) (*Page, error) {
	m := nextDataRe.FindSubmatch(body)
	if m == nil {
		return nil, fmt.Errorf("no __NEXT_DATA__ script in page")
	}
	var env struct {
		Props struct {
			PageProps Page `json:"pageProps"`
		} `json:"props"`
	}
	if err := json.Unmarshal(m[1], &env); err != nil {
		return nil, fmt.Errorf("decode __NEXT_DATA__: %w", err)
	}
	return &env.Props.PageProps, nil
}

// ToListing converts an item; ok is false for non-watches and unpublished or sold stock.
func ToListing(it Item, p *Page) (model.Listing, bool) {
	if it.ID == "" || (it.Type != "" && it.Type != "WATCHES") || (it.Status != "" && it.Status != "STOCK") || (!it.IsPublished && it.Status == "") {
		return model.Listing{}, false
	}
	brand := it.WatchBrand.Name.En()
	modelName := it.Model.En()
	title := normalize.CleanText(strings.Join([]string{brand, modelName, it.ModelRefNumber, it.SizeRemark.En()}, " "))
	if title == "" {
		return model.Listing{}, false
	}
	l := model.Listing{
		ExternalID:      it.ID,
		URL:             baseURL + "/watch/" + it.ID,
		Title:           title,
		Model:           normalize.CleanText(modelName),
		ReferenceNumber: strings.ToUpper(strings.TrimSpace(it.ModelRefNumber)),
		Currency:        "HKD",
		SellerType:      "dealer",
		SellerName:      "Ken's Watches",
		LocationCountry: "HK",
		LocationCity:    "Hong Kong",
		Condition:       model.ConditionGood, // a pre-owned dealer: NEW is flagged explicitly below
	}
	if b := normalize.DetectBrand(brand, title); b != nil {
		l.BrandSlug, l.BrandName = b.Slug, b.Name
	}
	switch {
	case it.Cash != nil && *it.Cash > 0:
		l.Price = it.Cash
	case it.ListedPrice != nil && *it.ListedPrice > 0:
		l.Price = it.ListedPrice
	}
	box, papers := false, false
	for _, c := range it.Conditions {
		switch strings.ToUpper(c) {
		case "BOX":
			box = true
		case "PAPER", "PAPERS":
			papers = true
		case "NEW", "BRAND_NEW":
			l.Condition = model.ConditionNew
		case "UNWORN", "UNUSED":
			l.Condition = model.ConditionUnworn
		}
	}
	l.HasBox, l.HasPapers = &box, &papers
	if y, err := strconv.Atoi(strings.TrimSpace(it.Year)); err == nil && y >= 1900 && y <= 2100 {
		l.Year = &y
	}
	l.CaseDiameterMM = normalize.ParseDiameter(it.SizeRemark.En())
	if mv := optionName(it.WatchMovement, p.FilterOpt.WatchMovements); mv != "" {
		l.Movement = normalize.MovementType(mv)
	}
	if sz := optionName(it.WatchSize, p.FilterOpt.WatchSizes); sz != "" {
		l.Gender = normalize.GenderType(sz) // Men / Ladies / Boy / Jumbo …
	}
	if it.Material != nil && *it.Material != "" {
		l.CaseMaterial = *it.Material
	}
	for _, ph := range it.Photos {
		if ph != "" && len(l.ImageURLs) < 8 {
			l.ImageURLs = append(l.ImageURLs, photoURL+strings.TrimPrefix(ph, "/"))
		}
	}
	attrs := map[string]any{"itemId": it.ItemID, "conditions": it.Conditions, "paymentType": it.PaymentType, "publishedDate": it.PublishedDate}
	if st := it.Store.Name.En(); st != "" {
		attrs["store"] = st
	}
	if r := it.Remark.En(); r != "" {
		attrs["remark"] = r
	}
	if it.PriceRemark != nil && *it.PriceRemark != "" {
		attrs["priceRemark"] = *it.PriceRemark
	}
	if it.ListedPrice != nil && it.Cash != nil && *it.ListedPrice > *it.Cash {
		attrs["listPrice"] = *it.ListedPrice
	}
	l.Attributes, _ = json.Marshal(attrs)
	return l, true
}

// optionName resolves a field that is either an option id or an embedded option object.
func optionName(raw json.RawMessage, table []Option) string {
	if len(raw) == 0 {
		return ""
	}
	var id string
	if err := json.Unmarshal(raw, &id); err != nil {
		var o Option
		if json.Unmarshal(raw, &o) != nil {
			return ""
		}
		if n := o.Name.En(); n != "" {
			return n
		}
		if d := o.Description.En(); d != "" {
			return d
		}
		id = o.ID
	}
	for _, o := range table {
		if o.ID == id {
			if n := o.Name.En(); n != "" {
				return n
			}
			return o.Description.En()
		}
	}
	return ""
}
