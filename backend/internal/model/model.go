// Package model defines the domain types shared by the crawler, repository and API.
package model

import (
	"encoding/json"
	"time"
)

// Condition is the normalized condition scale (mirrors Chrono24's).
type Condition string

const (
	ConditionNew      Condition = "new"
	ConditionUnworn   Condition = "unworn"
	ConditionVeryGood Condition = "very_good"
	ConditionGood     Condition = "good"
	ConditionFair     Condition = "fair"
	ConditionPoor     Condition = "poor"
	ConditionUnknown  Condition = "unknown"
)

// Movement is the normalized movement type.
type Movement string

const (
	MovementAutomatic Movement = "automatic"
	MovementManual    Movement = "manual"
	MovementQuartz    Movement = "quartz"
	MovementUnknown   Movement = "unknown"
)

// Gender is the normalized target gender.
type Gender string

const (
	GenderMen     Gender = "men"
	GenderWomen   Gender = "women"
	GenderUnisex  Gender = "unisex"
	GenderUnknown Gender = "unknown"
)

// Source describes a crawled marketplace.
type Source struct {
	ID       int16  `json:"id"`
	Key      string `json:"key"`
	Name     string `json:"name"`
	BaseURL  string `json:"baseUrl"`
	Country  string `json:"country"`
	Currency string `json:"currency"`
	Enabled  bool   `json:"enabled"`
}

// Brand is a canonical watch brand.
type Brand struct {
	ID           int    `json:"id"`
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	ListingCount int    `json:"listingCount"`
}

// Listing is a single watch offer as stored in PostgreSQL.
type Listing struct {
	ID              int64           `json:"id"`
	SourceID        int16           `json:"-"`
	SourceKey       string          `json:"source"`
	SourceName      string          `json:"sourceName"`
	ExternalID      string          `json:"externalId"`
	URL             string          `json:"url"`
	Title           string          `json:"title"`
	BrandID         *int            `json:"brandId,omitempty"`
	BrandSlug       string          `json:"brand,omitempty"`
	BrandName       string          `json:"brandName,omitempty"`
	Model           string          `json:"model,omitempty"`
	ReferenceNumber string          `json:"referenceNumber,omitempty"`
	Condition       Condition       `json:"condition"`
	Year            *int            `json:"year,omitempty"`
	CaseDiameterMM  *float64        `json:"caseDiameterMm,omitempty"`
	CaseMaterial    string          `json:"caseMaterial,omitempty"`
	DialColor       string          `json:"dialColor,omitempty"`
	Movement        Movement        `json:"movement"`
	Gender          Gender          `json:"gender"`
	HasBox          *bool           `json:"hasBox,omitempty"`
	HasPapers       *bool           `json:"hasPapers,omitempty"`
	Price           *float64        `json:"price,omitempty"`
	Currency        string          `json:"currency,omitempty"`
	PriceUSD        *float64        `json:"priceUsd,omitempty"`
	ShippingPrice   *float64        `json:"shippingPrice,omitempty"`
	LocationCountry string          `json:"locationCountry,omitempty"`
	LocationCity    string          `json:"locationCity,omitempty"`
	SellerName      string          `json:"sellerName,omitempty"`
	SellerType      string          `json:"sellerType,omitempty"`
	ImageURLs       []string        `json:"imageUrls"`
	Description     string          `json:"description,omitempty"`
	Attributes      json.RawMessage `json:"attributes,omitempty"`
	IsActive        bool            `json:"isActive"`
	FirstSeenAt     time.Time       `json:"firstSeenAt"`
	LastSeenAt      time.Time       `json:"lastSeenAt"`
}

// PricePoint is one observation in a listing's price history.
type PricePoint struct {
	Price      *float64  `json:"price"`
	Currency   string    `json:"currency"`
	PriceUSD   *float64  `json:"priceUsd"`
	ObservedAt time.Time `json:"observedAt"`
}

// CrawlRun is an audit row for one crawler execution against one source.
type CrawlRun struct {
	ID                  int64      `json:"id"`
	SourceKey           string     `json:"source"`
	StartedAt           time.Time  `json:"startedAt"`
	FinishedAt          *time.Time `json:"finishedAt,omitempty"`
	Status              string     `json:"status"`
	ListingsSeen        int        `json:"listingsSeen"`
	ListingsNew         int        `json:"listingsNew"`
	ListingsUpdated     int        `json:"listingsUpdated"`
	ListingsDeactivated int        `json:"listingsDeactivated"`
	Error               string     `json:"error,omitempty"`
}

// SortKey enumerates the supported sort orders (Chrono24 parity).
type SortKey string

const (
	SortRelevance SortKey = "relevance"
	SortPriceAsc  SortKey = "price_asc"
	SortPriceDesc SortKey = "price_desc"
	SortNewest    SortKey = "newest"
	SortOldest    SortKey = "oldest"
	SortYearDesc  SortKey = "year_desc"
	SortYearAsc   SortKey = "year_asc"
	SortSizeAsc   SortKey = "size_asc"
	SortSizeDesc  SortKey = "size_desc"
)

// ListingQuery is the parsed, validated search request.
type ListingQuery struct {
	Text            string
	TextAlt         string   // canonicalized (English) form of Text, OR-ed in when different
	Brands          []string // slugs
	Model           string
	Reference       string
	Sources         []string // keys
	Conditions      []Condition
	Movements       []Movement
	Genders         []Gender
	Countries       []string
	DialColors      []string
	PriceMinUSD     *float64
	PriceMaxUSD     *float64
	YearMin         *int
	YearMax         *int
	DiameterMin     *float64
	DiameterMax     *float64
	HasBox          *bool
	HasPapers       *bool
	Sort            SortKey
	Page            int
	PerPage         int
	IncludeInactive bool
}

// FacetValue is one bucket in a facet aggregation.
type FacetValue struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

// Facets accompany a search result so the UI can render filter counts.
type Facets struct {
	Brands     []FacetValue `json:"brands"`
	Sources    []FacetValue `json:"sources"`
	Conditions []FacetValue `json:"conditions"`
	Movements  []FacetValue `json:"movements"`
	Countries  []FacetValue `json:"countries"`
	DialColors []FacetValue `json:"dialColors"`
	Years      []FacetValue `json:"years"`
	PriceMin   *float64     `json:"priceMinUsd,omitempty"`
	PriceMax   *float64     `json:"priceMaxUsd,omitempty"`
}

// NewFacets returns a Facets with every bucket initialized. Never build a Facets literal for a
// response: a nil slice marshals to JSON null, and the frontend spreads these arrays.
func NewFacets() Facets {
	return Facets{
		Brands:     []FacetValue{},
		Sources:    []FacetValue{},
		Conditions: []FacetValue{},
		Movements:  []FacetValue{},
		Countries:  []FacetValue{},
		DialColors: []FacetValue{},
		Years:      []FacetValue{},
	}
}

// SearchResult is the paginated response payload.
type SearchResult struct {
	Items      []Listing `json:"items"`
	Total      int       `json:"total"`
	Page       int       `json:"page"`
	PerPage    int       `json:"perPage"`
	TotalPages int       `json:"totalPages"`
	Facets     Facets    `json:"facets"`
}

// ExchangeRate is 1 USD expressed in Quote.
type ExchangeRate struct {
	Quote     string    `json:"quote"`
	Rate      float64   `json:"rate"`
	FetchedAt time.Time `json:"fetchedAt"`
}
