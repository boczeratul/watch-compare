// Package ebay indexes eBay wristwatch listings through the official Browse API
// (https://developer.ebay.com/api-docs/buy/browse/overview.html). Scraping eBay HTML violates
// its terms and is unreliable; the Browse API is free with an eBay developer account.
//
// Requires EBAY_CLIENT_ID / EBAY_CLIENT_SECRET (client-credentials OAuth). Marketplaces come from
// EBAY_MARKETPLACES (e.g. EBAY_US,EBAY_GB,EBAY_DE). Search is done per brand within the
// "Wristwatches" category (31387) because the API caps any single query at 10,000 results.
package ebay

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

const (
	tokenURL      = "https://api.ebay.com/identity/v1/oauth2/token"
	searchURL     = "https://api.ebay.com/buy/browse/v1/item_summary/search"
	wristwatchCat = "31387"
	pageSize      = 200
	maxOffset     = 10000
)

// Brands searched on eBay. Keep this list focused: each brand × marketplace is up to 50 API calls.
var Brands = []string{
	"rolex", "tudor", "omega", "patek-philippe", "audemars-piguet", "vacheron-constantin", "a-lange-sohne",
	"cartier", "breitling", "iwc", "jaeger-lecoultre", "panerai", "hublot", "tag-heuer", "zenith", "breguet",
	"blancpain", "grand-seiko", "seiko", "longines", "oris", "nomos", "sinn", "tissot", "hamilton",
}

// Source implements crawler.Source.
type Source struct {
	mu     sync.Mutex
	token  string
	expiry time.Time
}

// Key returns the source key.
func (*Source) Key() string { return "ebay" }

func (s *Source) accessToken(ctx context.Context, env *crawler.Env) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token != "" && time.Now().Before(s.expiry.Add(-2*time.Minute)) {
		return s.token, nil
	}
	form := url.Values{"grant_type": {"client_credentials"}, "scope": {"https://api.ebay.com/oauth/api_scope"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(env.Cfg.EbayClientID+":"+env.Cfg.EbayClientSecret)))
	resp, err := env.Fetcher.Client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ebay token: http %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", err
	}
	s.token = tok.AccessToken
	s.expiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	return s.token, nil
}

type searchResponse struct {
	Total         int           `json:"total"`
	Next          string        `json:"next"`
	ItemSummaries []itemSummary `json:"itemSummaries"`
}

type itemSummary struct {
	ItemID           string `json:"itemId"`
	Title            string `json:"title"`
	ItemWebURL       string `json:"itemWebUrl"`
	Condition        string `json:"condition"`
	ItemCreationDate string `json:"itemCreationDate"`
	Price            struct {
		Value    string `json:"value"`
		Currency string `json:"currency"`
	} `json:"price"`
	Image struct {
		ImageURL string `json:"imageUrl"`
	} `json:"image"`
	AdditionalImages []struct {
		ImageURL string `json:"imageUrl"`
	} `json:"additionalImages"`
	ItemLocation struct {
		City    string `json:"city"`
		Country string `json:"country"`
	} `json:"itemLocation"`
	Seller struct {
		Username           string `json:"username"`
		FeedbackPercentage string `json:"feedbackPercentage"`
		FeedbackScore      int    `json:"feedbackScore"`
	} `json:"seller"`
	ShippingOptions []struct {
		ShippingCost struct {
			Value    string `json:"value"`
			Currency string `json:"currency"`
		} `json:"shippingCost"`
		ShippingCostType string `json:"shippingCostType"`
	} `json:"shippingOptions"`
	BuyingOptions []string `json:"buyingOptions"`
	Categories    []struct {
		CategoryID string `json:"categoryId"`
	} `json:"categories"`
}

// Crawl searches each brand on each configured marketplace.
func (s *Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	if env.Cfg.EbayClientID == "" || env.Cfg.EbayClientSecret == "" {
		return fmt.Errorf("EBAY_CLIENT_ID / EBAY_CLIENT_SECRET not set; skipping eBay")
	}
	seen := map[string]bool{}
	for _, market := range env.Cfg.EbayMarketplaces {
		for _, slug := range Brands {
			b, ok := normalize.BrandBySlug(slug)
			if !ok {
				continue
			}
			if err := s.searchBrand(ctx, env, market, b.Name, seen, emit); err != nil {
				if strings.Contains(err.Error(), "http 429") {
					return err // quota: stop everything
				}
				env.Log.Warn().Err(err).Str("market", market).Str("brand", b.Name).Msg("ebay brand search failed")
			}
		}
	}
	return nil
}

func (s *Source) searchBrand(ctx context.Context, env *crawler.Env, market, brand string, seen map[string]bool, emit crawler.Emit) error {
	for offset := 0; offset < maxOffset; offset += pageSize {
		token, err := s.accessToken(ctx, env)
		if err != nil {
			return err
		}
		q := url.Values{
			"category_ids":  {wristwatchCat},
			"aspect_filter": {fmt.Sprintf("categoryId:%s,Brand:{%s}", wristwatchCat, brand)},
			"filter":        {"buyingOptions:{FIXED_PRICE|BEST_OFFER},priceCurrency:" + marketCurrency(market) + ",price:[500..]"},
			"sort":          {"newlyListed"},
			"limit":         {strconv.Itoa(pageSize)},
			"offset":        {strconv.Itoa(offset)},
		}
		req, err := http.NewRequest(http.MethodGet, searchURL+"?"+q.Encode(), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-EBAY-C-MARKETPLACE-ID", market)
		req.Header.Set("Accept", "application/json")
		resp, err := env.Fetcher.Do(ctx, req)
		if err != nil {
			return err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("ebay search: http %d: %s", resp.StatusCode, truncate(string(body), 200))
		}
		var sr searchResponse
		if err := json.Unmarshal(body, &sr); err != nil {
			return err
		}
		for _, it := range sr.ItemSummaries {
			if seen[it.ItemID] {
				continue
			}
			seen[it.ItemID] = true
			l := toListing(it, market)
			if err := emit(l); err != nil {
				return err
			}
		}
		if sr.Next == "" || len(sr.ItemSummaries) < pageSize || (offset/pageSize)+1 >= env.MaxPages {
			return nil
		}
	}
	return nil
}

func toListing(it itemSummary, market string) model.Listing {
	l := model.Listing{
		ExternalID:      it.ItemID,
		URL:             it.ItemWebURL,
		Title:           normalize.CleanText(it.Title),
		Condition:       normalize.Condition(it.Condition),
		Currency:        it.Price.Currency,
		LocationCountry: it.ItemLocation.Country,
		LocationCity:    it.ItemLocation.City,
		SellerName:      it.Seller.Username,
		SellerType:      "private",
	}
	if p, err := strconv.ParseFloat(it.Price.Value, 64); err == nil && p > 0 {
		l.Price = &p
	}
	if it.Image.ImageURL != "" {
		l.ImageURLs = append(l.ImageURLs, hiRes(it.Image.ImageURL))
	}
	for _, im := range it.AdditionalImages {
		if im.ImageURL != "" && len(l.ImageURLs) < 12 {
			l.ImageURLs = append(l.ImageURLs, hiRes(im.ImageURL))
		}
	}
	for _, so := range it.ShippingOptions {
		if v, err := strconv.ParseFloat(so.ShippingCost.Value, 64); err == nil {
			l.ShippingPrice = &v
			break
		}
	}
	if it.Seller.FeedbackScore > 1000 {
		l.SellerType = "dealer"
	}
	attrs := map[string]any{
		"marketplace":         market,
		"ebayCondition":       it.Condition,
		"buyingOptions":       it.BuyingOptions,
		"sellerFeedbackPct":   it.Seller.FeedbackPercentage,
		"sellerFeedbackScore": it.Seller.FeedbackScore,
		"listedAt":            it.ItemCreationDate,
	}
	l.Attributes, _ = json.Marshal(attrs)
	return l
}

// hiRes asks eBay's image CDN for the 1600px rendition.
func hiRes(u string) string {
	if i := strings.LastIndex(u, "/s-l"); i > 0 {
		if j := strings.Index(u[i:], "."); j > 0 {
			return u[:i] + "/s-l1600" + u[i+j:]
		}
	}
	return u
}

func marketCurrency(market string) string {
	switch market {
	case "EBAY_GB":
		return "GBP"
	case "EBAY_DE", "EBAY_FR", "EBAY_IT", "EBAY_ES", "EBAY_NL", "EBAY_AT", "EBAY_BE", "EBAY_IE":
		return "EUR"
	case "EBAY_CH":
		return "CHF"
	case "EBAY_AU":
		return "AUD"
	case "EBAY_CA":
		return "CAD"
	case "EBAY_HK":
		return "HKD"
	case "EBAY_SG":
		return "SGD"
	}
	return "USD"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
