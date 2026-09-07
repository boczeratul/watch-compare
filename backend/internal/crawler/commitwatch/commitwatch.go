// Package commitwatch indexes https://commit-watch.co.jp (Commit Ginza, Tokyo; JPY).
//
// The shop runs on Shopify, so instead of scraping HTML we read the public storefront feed
// GET /products.json?limit=250&page=N (verified 2026-09; 250 products per page). Each product has
// vendor (the brand, in English), title (Japanese, including "Ref.<number>" and the condition
// grade), variants[0].price/available, images[].src and tags.
package commitwatch

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

const (
	baseURL  = "https://commit-watch.co.jp"
	pageSize = 250
)

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "commitwatch" }

// Feed mirrors the Shopify products.json payload (only the fields we use).
type Feed struct {
	Products []Product `json:"products"`
}

// Product is one Shopify product.
type Product struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Handle      string    `json:"handle"`
	BodyHTML    string    `json:"body_html"`
	Vendor      string    `json:"vendor"`
	ProductType string    `json:"product_type"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at"`
	Variants    []struct {
		Price          string `json:"price"`
		CompareAtPrice string `json:"compare_at_price"`
		Available      bool   `json:"available"`
		SKU            string `json:"sku"`
	} `json:"variants"`
	Images []struct {
		Src string `json:"src"`
	} `json:"images"`
}

var (
	refRe  = regexp.MustCompile(`(?i)Ref\.?\s*([A-Z0-9][A-Z0-9./-]{2,})`)
	tagRe  = regexp.MustCompile(`<[^>]+>`)
	yearRe = regexp.MustCompile(`(\d{4})年`)
)

// Crawl pages through the JSON feed until a short page.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	seen := map[string]bool{}
	for page := 1; page <= env.MaxPages; page++ {
		body, err := env.Fetcher.Get(ctx, fmt.Sprintf("%s/products.json?limit=%d&page=%d", baseURL, pageSize, page))
		if err != nil {
			return fmt.Errorf("page %d: %w", page, err)
		}
		var feed Feed
		if err := json.Unmarshal(body, &feed); err != nil {
			return fmt.Errorf("page %d: decode: %w", page, err)
		}
		if len(feed.Products) == 0 {
			if page == 1 {
				return fmt.Errorf("empty feed on first page (endpoint changed?)")
			}
			return nil
		}
		for _, p := range feed.Products {
			l, ok := ToListing(p)
			if !ok || seen[l.ExternalID] {
				continue
			}
			seen[l.ExternalID] = true
			if err := emit(l); err != nil {
				return err
			}
		}
		if len(feed.Products) < pageSize {
			return nil
		}
	}
	return nil
}

// ToListing converts a Shopify product; ok is false for non-watches and sold-out items.
func ToListing(p Product) (model.Listing, bool) {
	if p.ProductType != "" && p.ProductType != "腕時計" && !strings.Contains(p.ProductType, "時計") {
		return model.Listing{}, false
	}
	if len(p.Variants) == 0 || !p.Variants[0].Available {
		return model.Listing{}, false
	}
	l := model.Listing{
		ExternalID:      strconv.FormatInt(p.ID, 10),
		URL:             baseURL + "/products/" + p.Handle,
		Title:           normalize.CleanText(strings.TrimSuffix(strings.TrimSpace(p.Title), " "+p.Handle)),
		Currency:        "JPY",
		SellerType:      "dealer",
		SellerName:      "Commit Ginza",
		LocationCountry: "JP",
		LocationCity:    "Tokyo",
		Condition:       model.ConditionUnknown,
	}
	if b := normalize.DetectBrand(p.Vendor, p.Title); b != nil {
		l.BrandSlug, l.BrandName = b.Slug, b.Name
	}
	if v, err := strconv.ParseFloat(p.Variants[0].Price, 64); err == nil && v > 0 {
		l.Price = &v
	}
	if m := refRe.FindStringSubmatch(p.Title); m != nil {
		l.ReferenceNumber = strings.ToUpper(m[1])
	}
	for _, im := range p.Images {
		if im.Src != "" && len(l.ImageURLs) < 8 {
			l.ImageURLs = append(l.ImageURLs, im.Src)
		}
	}
	// Condition grade is embedded in the title: 新品 / 未使用 / 極美中古 / 美中古 / 中古
	switch {
	case strings.Contains(p.Title, "未使用"):
		l.Condition = model.ConditionUnworn
	case strings.Contains(p.Title, "新品"):
		l.Condition = model.ConditionNew
	case strings.Contains(p.Title, "極美中古") || strings.Contains(p.Title, "美中古"):
		l.Condition = model.ConditionVeryGood
	case strings.Contains(p.Title, "中古"):
		l.Condition = model.ConditionGood
	}
	desc := normalize.CleanText(tagRe.ReplaceAllString(p.BodyHTML, " "))
	if desc != "" {
		l.Description = desc
		if m := yearRe.FindStringSubmatch(desc); m != nil {
			if y, err := strconv.Atoi(m[1]); err == nil && y >= 1900 && y <= 2100 {
				l.Year = &y
			}
		}
		l.HasBox, l.HasPapers = normalize.DetectBoxPapers(desc)
	}
	attrs := map[string]any{"tags": p.Tags, "vendor": p.Vendor, "listedAt": p.CreatedAt}
	if p.Variants[0].CompareAtPrice != "" {
		attrs["compareAtPrice"] = p.Variants[0].CompareAtPrice
	}
	l.Attributes, _ = json.Marshal(attrs)
	return l, true
}
