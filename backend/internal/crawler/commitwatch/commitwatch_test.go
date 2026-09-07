package commitwatch

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
)

func TestToListingFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/products.json")
	if err != nil {
		t.Skip("fixture missing")
	}
	var feed Feed
	if err := json.Unmarshal(body, &feed); err != nil {
		t.Fatal(err)
	}
	if len(feed.Products) < 20 {
		t.Fatalf("expected products, got %d", len(feed.Products))
	}
	n, withPrice, withBrand, withRef, withImg, withCond := 0, 0, 0, 0, 0, 0
	for _, p := range feed.Products {
		l, ok := ToListing(p)
		if !ok {
			continue
		}
		n++
		crawler.EnrichFromTitle(&l)
		if l.ExternalID == "" || l.URL == "" || l.Title == "" {
			t.Errorf("incomplete listing: %+v", l)
		}
		if l.Price != nil {
			withPrice++
		}
		if l.BrandSlug != "" {
			withBrand++
		}
		if l.ReferenceNumber != "" {
			withRef++
		}
		if len(l.ImageURLs) > 0 {
			withImg++
		}
		if l.Condition != "unknown" {
			withCond++
		}
		if n == 1 {
			t.Logf("sample: %s | %s | %s | ref=%s | %v JPY | cond=%s | model=%s", l.ExternalID, l.BrandName, l.Title, l.ReferenceNumber, *l.Price, l.Condition, l.Model)
		}
	}
	t.Logf("listings=%d price=%d brand=%d ref=%d img=%d cond=%d", n, withPrice, withBrand, withRef, withImg, withCond)
	if n < 10 || withPrice < n || withBrand < n*8/10 || withRef < n/2 || withImg < n*8/10 {
		t.Errorf("too few parsed fields")
	}
}
