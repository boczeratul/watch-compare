package sevenhours

import (
	"os"
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
)

func TestParseListFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/list.html")
	if err != nil {
		t.Skip("fixture missing")
	}
	doc, err := crawler.ParseHTML(body, "https://7hours.jp/SHOP/list.php?Search=&Type=01&PAGE=1")
	if err != nil {
		t.Fatal(err)
	}
	if !HasNext(doc) {
		t.Error("first page should link to a next page")
	}
	items := ParseList(doc)
	if len(items) != 40 {
		t.Fatalf("expected 40 cards, got %d", len(items))
	}
	withPrice, withExcl, withBrand, withRef, withImg := 0, 0, 0, 0, 0
	for i := range items {
		l := &items[i]
		if l.ExternalID == "" || l.URL == "" || l.Title == "" {
			t.Errorf("incomplete listing: %+v", l)
		}
		crawler.EnrichFromTitle(l)
		if l.Price != nil {
			withPrice++
		}
		if l.PriceExclTax != nil {
			if *l.PriceExclTax >= *l.Price {
				t.Errorf("tax-free price %v not below listed %v for %s", *l.PriceExclTax, *l.Price, l.ExternalID)
			}
			withExcl++
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
	}
	t.Logf("items=%d price=%d excl=%d brand=%d ref=%d img=%d sample=%+v", len(items), withPrice, withExcl, withBrand, withRef, withImg, items[0])
	if withPrice < 38 || withExcl < 38 || withBrand < 30 || withRef < 30 || withImg < 38 {
		t.Errorf("too few parsed fields: price=%d excl=%d brand=%d ref=%d img=%d of %d", withPrice, withExcl, withBrand, withRef, withImg, len(items))
	}
	if items[0].ExternalID != "149117" || items[0].Title[:5] != "ROLEX" {
		t.Errorf("first card parsed wrong: %+v", items[0])
	}
}
