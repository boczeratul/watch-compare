// Package wristcheck indexes https://wristcheck.com (Wristcheck, Hong Kong with a New York
// showroom; priced in HKD).
//
// The Next.js storefront searches an Algolia index with a public search-only key embedded in the
// page (verified 2026-09):
//
//	POST https://RJEIADM991-dsn.algolia.net/1/indexes/prod_product_listings/query
//	{"params": "query=&hitsPerPage=100&page=<n>&filters=NOT status:sold"}
//
// The index keeps sold pieces (status "sold"), so the crawl filters them out (~1.8k available of
// ~7.7k). Hits carry title, brand{label,slug}, slug, referenceNumber, family.label (model),
// price{hkd,usd,…}, condition (WATCH_UNWORN | WATCH_WORN), wcGrade ("8.0" … "10.0", BRAND_NEW),
// certificates (original_box, has_papers), yearPurchased, caseDiameter, gender, location
// (hk | us_ny), heroImage / imageList. The product page is /buy/<brand.slug>/<slug>.
package wristcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

const (
	baseURL   = "https://wristcheck.com"
	appID     = "RJEIADM991"
	searchKey = "fd07f2301403de71c54f59b331d5c635" // public search-only key served to every visitor
	indexName = "prod_product_listings"
	pageSize  = 100
)

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "wristcheck" }

// Marketplace marks Wristcheck as multi-location: each hit says whether it sits in Hong Kong or
// New York, so an unknown location is not assumed to be the source's country.
func (Source) Marketplace() bool { return true }

// SearchResponse mirrors the Algolia query envelope.
type SearchResponse struct {
	Hits    []Hit `json:"hits"`
	NbHits  int   `json:"nbHits"`
	Page    int   `json:"page"`
	NbPages int   `json:"nbPages"`
}

// Hit is one indexed variant.
type Hit struct {
	ObjectID        string `json:"objectID"`
	Title           string `json:"title"`
	Model           string `json:"model"`
	SKU             string `json:"sku"`
	Gender          string `json:"gender"`
	ReferenceNumber string `json:"referenceNumber"`
	Slug            string `json:"slug"`
	Brand           struct {
		Label string `json:"label"`
		Slug  string `json:"slug"`
	} `json:"brand"`
	Family struct {
		Label string `json:"label"`
	} `json:"family"`
	Certificates  []string           `json:"certificates"`
	WCGrade       *string            `json:"wcGrade"`
	Location      string             `json:"location"`
	Condition     string             `json:"condition"`
	Price         map[string]float64 `json:"price"`
	LowestPrice   map[string]float64 `json:"lowestPrice"`
	MarketPrice   *float64           `json:"marketPrice"`
	HeroImage     string             `json:"heroImage"`
	ImageList     []string           `json:"imageList"`
	Status        string             `json:"status"`
	OriginBox     bool               `json:"originBox"`
	OriginalPaper bool               `json:"originalPaper"`
	ServicePapers bool               `json:"servicePapers"`
	YearPurchased *int               `json:"yearPurchased"`
	CaseDiameter  *float64           `json:"caseDiameter"`
	VariantCount  int                `json:"variantCount"`
	LimitedEd     bool               `json:"wristcheckLimitedEdition"`
	UpdatedAt     string             `json:"updatedAt"`
}

// Crawl pages through the unsold hits.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	seen := map[string]bool{}
	for page := 0; page < env.MaxPages; page++ {
		res, err := s.fetchPage(ctx, env, page)
		if err != nil {
			return fmt.Errorf("page %d: %w", page, err)
		}
		if len(res.Hits) == 0 {
			if page == 0 {
				return fmt.Errorf("empty index response on first page (index or key changed?)")
			}
			return nil
		}
		for _, h := range res.Hits {
			l, ok := ToListing(h)
			if !ok || seen[l.ExternalID] {
				continue
			}
			seen[l.ExternalID] = true
			if err := emit(l); err != nil {
				return err
			}
		}
		if page+1 >= res.NbPages {
			return nil
		}
	}
	return nil
}

func (s Source) fetchPage(ctx context.Context, env *crawler.Env, page int) (*SearchResponse, error) {
	params := url.Values{}
	params.Set("query", "")
	params.Set("hitsPerPage", strconv.Itoa(pageSize))
	params.Set("page", strconv.Itoa(page))
	params.Set("filters", "NOT status:sold")
	body, _ := json.Marshal(map[string]string{"params": params.Encode()})
	u := fmt.Sprintf("https://%s-dsn.algolia.net/1/indexes/%s/query", appID, indexName)
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Algolia-Application-Id", appID)
	req.Header.Set("X-Algolia-API-Key", searchKey)
	resp, err := env.Fetcher.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("algolia: http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw[:min(len(raw), 200)])))
	}
	return ParseSearch(raw)
}

// ParseSearch decodes an Algolia response. Exported for fixture tests.
func ParseSearch(body []byte) (*SearchResponse, error) {
	var res SearchResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &res, nil
}

// ToListing converts a hit; ok is false for sold pieces.
func ToListing(h Hit) (model.Listing, bool) {
	if h.ObjectID == "" || h.Status == "sold" || h.Slug == "" {
		return model.Listing{}, false
	}
	title := normalize.CleanText(strings.TrimSpace(h.Brand.Label + " " + h.Title))
	if h.ReferenceNumber != "" && !strings.Contains(strings.ToUpper(title), strings.ToUpper(h.ReferenceNumber)) {
		title += " " + h.ReferenceNumber
	}
	l := model.Listing{
		ExternalID:      h.ObjectID,
		URL:             fmt.Sprintf("%s/buy/%s/%s", baseURL, h.Brand.Slug, h.Slug),
		Title:           title,
		Model:           normalize.CleanText(h.Family.Label),
		ReferenceNumber: strings.ToUpper(strings.TrimSpace(h.ReferenceNumber)),
		Currency:        "HKD",
		Condition:       condition(h),
		SellerType:      "dealer",
		SellerName:      "Wristcheck",
		Gender:          normalize.GenderType(h.Gender),
	}
	switch h.Location {
	case "hk":
		l.LocationCountry, l.LocationCity = "HK", "Hong Kong"
	case "us_ny":
		l.LocationCountry, l.LocationCity = "US", "New York"
	}
	if b := normalize.DetectBrand(h.Brand.Label, title); b != nil {
		l.BrandSlug, l.BrandName = b.Slug, b.Name
	}
	if v := h.Price["hkd"]; v > 0 {
		l.Price = &v
	} else if v := h.LowestPrice["hkd"]; v > 0 {
		l.Price = &v
	}
	box, papers := h.OriginBox, h.OriginalPaper
	for _, c := range h.Certificates {
		switch c {
		case "original_box":
			box = true
		case "has_papers":
			papers = true
		}
	}
	l.HasBox, l.HasPapers = &box, &papers
	l.Year = h.YearPurchased
	if h.CaseDiameter != nil && *h.CaseDiameter >= 15 && *h.CaseDiameter <= 70 {
		l.CaseDiameterMM = h.CaseDiameter
	}
	if h.HeroImage != "" {
		l.ImageURLs = append(l.ImageURLs, h.HeroImage)
	}
	for _, im := range h.ImageList {
		if im != "" && im != h.HeroImage && len(l.ImageURLs) < 8 {
			l.ImageURLs = append(l.ImageURLs, im)
		}
	}
	attrs := map[string]any{"condition": h.Condition, "location": h.Location, "sku": h.SKU, "variants": h.VariantCount, "updatedAt": h.UpdatedAt}
	if h.WCGrade != nil {
		attrs["wcGrade"] = *h.WCGrade
	}
	if h.MarketPrice != nil && *h.MarketPrice > 0 {
		attrs["marketPrice"] = *h.MarketPrice
	}
	if usd := h.Price["usd"]; usd > 0 {
		attrs["priceUsdListed"] = usd
	}
	if h.LimitedEd {
		attrs["wristcheckLimitedEdition"] = true
	}
	if h.ServicePapers {
		attrs["servicePapers"] = true
	}
	l.Attributes, _ = json.Marshal(attrs)
	return l, true
}

// condition maps WATCH_UNWORN / WATCH_WORN plus the Wristcheck grade (out of 10) onto the scale.
func condition(h Hit) model.Condition {
	grade := ""
	if h.WCGrade != nil {
		grade = strings.ToUpper(*h.WCGrade)
	}
	if grade == "BRAND_NEW" {
		return model.ConditionNew
	}
	if h.Condition == "WATCH_UNWORN" {
		return model.ConditionUnworn
	}
	if g, err := strconv.ParseFloat(grade, 64); err == nil {
		switch {
		case g >= 9:
			return model.ConditionVeryGood
		case g >= 7:
			return model.ConditionGood
		default:
			return model.ConditionFair
		}
	}
	if h.Condition == "WATCH_WORN" {
		return model.ConditionGood
	}
	return model.ConditionUnknown
}
