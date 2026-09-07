package hourstack

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
	doc, err := crawler.ParseHTML(body, "https://www.hourstack.com.tw/product.asp?cat=5")
	if err != nil {
		t.Fatal(err)
	}
	items := Source{}.parseList(doc, "Breitling‧百年靈")
	if len(items) < 3 {
		t.Fatalf("expected items, got %d", len(items))
	}
	withPrice, withBrand, withImg := 0, 0, 0
	for _, l := range items {
		if l.ExternalID == "" || l.URL == "" || l.Title == "" {
			t.Errorf("incomplete listing: %+v", l)
		}
		if l.Price != nil {
			withPrice++
		}
		if l.BrandSlug != "" {
			withBrand++
		}
		if len(l.ImageURLs) > 0 {
			withImg++
		}
	}
	for _, l := range items {
		crawler.EnrichFromTitle(&l)
		t.Logf("%s | %s | ref=%s | %.0f %s | dia=%v | img=%d", l.ExternalID, l.Title, l.ReferenceNumber, deref(l.Price), l.Currency, l.CaseDiameterMM, len(l.ImageURLs))
		break
	}
	t.Logf("items=%d price=%d brand=%d img=%d", len(items), withPrice, withBrand, withImg)
	if withPrice < len(items)/2 || withBrand < len(items)/2 || withImg < len(items)/2 {
		t.Errorf("too few parsed fields: price=%d brand=%d img=%d of %d", withPrice, withBrand, withImg, len(items))
	}
}

func deref(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}
