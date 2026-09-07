package chrono24

import (
	"os"
	"strings"
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/config"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
)

// TestParseListFixture checks the selectors against a real rendered page. Unlike the other
// adapters, no fixture ships with the repository: Chrono24 answers plain HTTP with 403, so the
// page has to be captured through Browserless first. Create it with:
//
//	curl -s -X POST "https://production-sfo.browserless.io/smart-scrape?timeout=60000&token=$TOKEN" \
//	  -H 'Content-Type: application/json' \
//	  -d '{"url":"https://www.chrono24.com/rolex/index.htm?pageSize=120&showpage=1&sortorder=5","formats":["html"]}' \
//	  | jq -r .content > internal/crawler/chrono24/testdata/list.html
//
// then run: go test ./internal/crawler/chrono24/
// The file is git-ignored, so nobody has to commit Chrono24 markup.
func TestParseListFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/list.html")
	if err != nil {
		t.Skip("no captured page at testdata/list.html; see the comment above this test")
	}
	if len(body) < 5000 {
		t.Fatalf("captured page is only %d bytes: the render service probably returned an error page, not a listing page", len(body))
	}
	doc, err := crawler.ParseHTML(body, "https://www.chrono24.com/rolex/index.htm")
	if err != nil {
		t.Fatal(err)
	}
	items := ParseList(doc)
	if len(items) == 0 {
		t.Fatal("parsed 0 listings. Either the page is a captcha/blocked page (open it in a browser to check) " +
			"or the selectors in ParseList no longer match Chrono24's markup")
	}
	withPrice, withImg, withTitle := 0, 0, 0
	for _, l := range items {
		if l.ExternalID == "" || l.URL == "" {
			t.Errorf("listing without id or url: %+v", l)
		}
		if l.Title != "" {
			withTitle++
		}
		if l.Price != nil {
			withPrice++
		}
		if len(l.ImageURLs) > 0 {
			withImg++
		}
	}
	// Sellers may hide the price ("Price on request"); the card still prints the shipping cost,
	// which must not be mistaken for the price. The 2026-09 capture holds 8 such cards.
	onRequest, withShipping := 0, 0
	for _, l := range items {
		if l.ShippingPrice != nil {
			withShipping++
		}
		if l.Price == nil {
			onRequest++
			continue
		}
		if *l.Price < 500 {
			t.Errorf("implausibly cheap listing %s: price %v %s (shipping cost parsed as price?)", l.ExternalID, *l.Price, l.Currency)
		}
	}
	t.Logf("priceOnRequest=%d shipping=%d", onRequest, withShipping)
	// Every card names the seller's own country; the source's country (DE) must never be assumed.
	countries := map[string]int{}
	withCity, private := 0, 0
	for _, l := range items {
		if l.LocationCountry == "" {
			t.Errorf("listing %s has no country", l.ExternalID)
		}
		countries[l.LocationCountry]++
		if l.LocationCity != "" {
			withCity++
		}
		if l.SellerType == "private" {
			private++
		}
	}
	t.Logf("countries=%v city=%d private=%d", countries, withCity, private)
	if countries["DE"] == 0 || countries["DE"] == len(items) || countries["HK"] == 0 || countries["GB"] == 0 || countries["UK"] != 0 {
		t.Errorf("country split looks wrong: %v", countries)
	}
	if withCity < len(items)/2 || private == 0 {
		t.Errorf("city=%d private=%d of %d", withCity, private, len(items))
	}
	for _, l := range items {
		if l.LocationCity != "" && strings.ToUpper(l.LocationCity) == l.LocationCountry {
			t.Errorf("listing %s: country code %q stored as city", l.ExternalID, l.LocationCity)
		}
	}
	if withPrice+onRequest != len(items) {
		t.Errorf("price/on-request split does not add up: %d + %d != %d", withPrice, onRequest, len(items))
	}
	first := items[0]
	crawler.EnrichFromTitle(&first)
	t.Logf("items=%d title=%d price=%d img=%d", len(items), withTitle, withPrice, withImg)
	t.Logf("sample: id=%s brand=%s ref=%s price=%v %s title=%q",
		first.ExternalID, first.BrandSlug, first.ReferenceNumber, first.Price, first.Currency, first.Title)
	if withTitle < len(items)/2 || withPrice < len(items)/2 || withImg < len(items)/2 {
		t.Errorf("too many fields missing: title=%d price=%d img=%d of %d listings", withTitle, withPrice, withImg, len(items))
	}
}

func TestBrandsAndBudget(t *testing.T) {
	if got := Brands(&config.Config{}); len(got) != 5 {
		t.Errorf("default brand list = %v, want the five configured brands", got)
	}
	if got := Brands(&config.Config{Chrono24Brands: []string{"patek-philippe"}}); len(got) != 1 || got[0] != "patekphilippe" {
		t.Errorf("brand slug alias not applied: %v", got)
	}
}
