package watchnian

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
	doc, err := crawler.ParseHTML(body, "https://watchnian.com/shop/c/crl/")
	if err != nil {
		t.Fatal(err)
	}
	items := ParseList(doc)
	if len(items) < 20 {
		t.Fatalf("expected many items, got %d", len(items))
	}
	withPrice, withBrand, withRef, withImg, withCond := 0, 0, 0, 0, 0
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
		if l.ReferenceNumber != "" {
			withRef++
		}
		if len(l.ImageURLs) > 0 {
			withImg++
		}
		if l.Condition != "" && l.Condition != "unknown" {
			withCond++
		}
	}
	t.Logf("items=%d price=%d brand=%d ref=%d img=%d cond=%d sample=%+v", len(items), withPrice, withBrand, withRef, withImg, withCond, items[0])
	if withPrice < len(items)/2 || withBrand < len(items)/2 || withImg < len(items)/2 {
		t.Errorf("too few parsed fields: price=%d brand=%d img=%d of %d", withPrice, withBrand, withImg, len(items))
	}
}
