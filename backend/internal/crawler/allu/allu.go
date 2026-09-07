// Package allu indexes https://allu-official.com (ALLU, the Valuence group's marketplace; JPY).
//
// The storefront is a Nuxt app whose market list is loaded from a same-origin JSON API
// (verified 2026-09):
//
//	POST /jp/ja/api/v5/items/search?page=<n>&limit=30
//	{"categories":["c-79"],"order":1,"stock":2,"type":[1],"search":1, …}
//
// c-79 is the 腕時計 category, order 1 = newest first, and a limit above 30 is rejected (422).
// Each item carries name, item_category_name (brand, Japanese), model, item_number (reference),
// display_price_min (JPY, tax included), conditions (dealer rank N/S/SA/A/AB/B/BC/C/J), image.url
// and app_customer (the seller; "ALLU公式" is the house account, others are marketplace sellers).
// The category holds tens of thousands of pieces, so CRAWL_MAX_PAGES × 30 newest items are read
// per run.
package allu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

const (
	baseURL       = "https://allu-official.com"
	pageSize      = 30
	watchCategory = "c-79"
	houseSeller   = "ALLU公式"
)

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "allu" }

// SearchResponse mirrors the items/search payload (only the fields we use).
type SearchResponse struct {
	TotalCount int    `json:"total_count"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	Items      []Item `json:"items"`
}

// Item is one market listing.
type Item struct {
	ID              int64    `json:"id"`
	ManageNo        string   `json:"manage_no"`
	Name            string   `json:"name"`
	StoreName       string   `json:"store_name"`
	IsAlluPartner   bool     `json:"is_allu_partner"`
	DisplayPriceMin float64  `json:"display_price_min"`
	DisplayPriceMax float64  `json:"display_price_max"`
	DiscountPrice   *float64 `json:"discount_price"`
	Image           struct {
		URL string `json:"url"`
	} `json:"image"`
	AppCustomer struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"app_customer"`
	CategoryTop    string  `json:"item_category_top_name"`
	CategoryMiddle string  `json:"item_category_middle_name"`
	CategoryName   string  `json:"item_category_name"` // brand
	Model          string  `json:"model"`
	ItemNumber     string  `json:"item_number"`
	Color          *string `json:"color"`
	Conditions     string  `json:"conditions"`
	Stock          int     `json:"stock"`
	SaleStatus     int     `json:"sale_status"`
	IsNew          bool    `json:"is_new"`
	IsSale         bool    `json:"is_sale"`
}

func searchBody() []byte {
	b, _ := json.Marshal(map[string]any{
		"keyword": "", "categories": []string{watchCategory}, "brands": []string{}, "models": []string{},
		"ranks": []string{}, "gender": []string{}, "color": []string{}, "shop": []string{},
		"order": 1, "stock": 2, "type": []int{1}, "bin": 0, "sale": []string{}, "commitment": []string{},
		"seller": []string{}, "search": 1, "min_price": 0, "max_price": 0,
	})
	return b
}

// Crawl pages through the search API, newest first, until the total is reached.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	seen := map[string]bool{}
	for page := 1; page <= env.MaxPages; page++ {
		res, err := s.fetchPage(ctx, env, page)
		if err != nil {
			return fmt.Errorf("page %d: %w", page, err)
		}
		if len(res.Items) == 0 {
			if page == 1 {
				return fmt.Errorf("empty search result on first page (API changed?)")
			}
			return nil
		}
		for _, it := range res.Items {
			l, ok := ToListing(it)
			if !ok || seen[l.ExternalID] {
				continue
			}
			seen[l.ExternalID] = true
			if err := emit(l); err != nil {
				return err
			}
		}
		if len(res.Items) < pageSize || page*pageSize >= res.TotalCount {
			return nil
		}
	}
	return nil
}

func (s Source) fetchPage(ctx context.Context, env *crawler.Env, page int) (*SearchResponse, error) {
	u := fmt.Sprintf("%s/jp/ja/api/v5/items/search?page=%d&limit=%d", baseURL, page, pageSize)
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(searchBody()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := env.Fetcher.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: http %d", u, resp.StatusCode)
	}
	return ParseSearch(body)
}

// ParseSearch decodes a search response. Exported for fixture tests.
func ParseSearch(body []byte) (*SearchResponse, error) {
	var res SearchResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &res, nil
}

// ToListing converts an item; ok is false for pieces without stock or price.
func ToListing(it Item) (model.Listing, bool) {
	if it.ID == 0 || it.Stock <= 0 || it.DisplayPriceMin <= 0 {
		return model.Listing{}, false
	}
	title := normalize.CleanText(it.Name)
	if title == "" {
		return model.Listing{}, false
	}
	price := it.DisplayPriceMin
	l := model.Listing{
		ExternalID:      strconv.FormatInt(it.ID, 10),
		URL:             fmt.Sprintf("%s/jp/ja/market/items/%d/", baseURL, it.ID),
		Title:           title,
		Model:           normalize.CleanText(it.Model),
		ReferenceNumber: strings.ToUpper(strings.TrimSpace(it.ItemNumber)),
		Price:           &price,
		Currency:        "JPY",
		Condition:       rankToCondition(it.Conditions),
		SellerName:      it.AppCustomer.Name,
		SellerType:      "private",
		LocationCountry: "JP",
	}
	if it.AppCustomer.Name == houseSeller || it.IsAlluPartner {
		l.SellerType = "dealer"
	}
	if b := normalize.DetectBrand(it.CategoryName, it.Name); b != nil {
		l.BrandSlug, l.BrandName = b.Slug, b.Name
	}
	if it.Image.URL != "" {
		l.ImageURLs = []string{it.Image.URL}
	}
	if it.Color != nil && *it.Color != "" {
		l.DialColor = normalize.DialColor(*it.Color)
	}
	attrs := map[string]any{"rank": it.Conditions, "manageNo": it.ManageNo, "store": it.StoreName, "brandCategory": it.CategoryName}
	if it.DiscountPrice != nil && *it.DiscountPrice > 0 {
		attrs["discountPrice"] = *it.DiscountPrice
	}
	if it.DisplayPriceMax > it.DisplayPriceMin {
		attrs["priceMax"] = it.DisplayPriceMax
	}
	l.Attributes, _ = json.Marshal(attrs)
	return l, true
}

// rankToCondition maps ALLU's dealer grades onto the normalized scale (J = junk).
func rankToCondition(rank string) model.Condition {
	switch strings.ToUpper(strings.TrimSpace(rank)) {
	case "N":
		return model.ConditionUnworn
	case "S":
		return model.ConditionNew
	case "SA", "A":
		return model.ConditionVeryGood
	case "AB", "B":
		return model.ConditionGood
	case "BC", "C":
		return model.ConditionFair
	case "J", "D":
		return model.ConditionPoor
	}
	return model.ConditionUnknown
}
