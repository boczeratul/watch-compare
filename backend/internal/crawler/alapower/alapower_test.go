package alapower

import (
	"os"
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
)

func TestParseListFixtures(t *testing.T) {
	cases := []struct {
		fixture, url, category string
		min                    int
	}{
		{"testdata/hourstack_list.html", "https://www.hourstack.com.tw/product.asp?cat=5", "Breitling‧百年靈", 3},
		{"testdata/rdwatch_list.html", "https://www.rdwatch.com.tw/product.asp?cat=47", "Rolex‧勞力士", 20},
	}
	for _, c := range cases {
		body, err := os.ReadFile(c.fixture)
		if err != nil {
			t.Skip("fixture missing")
		}
		doc, err := crawler.ParseHTML(body, c.url)
		if err != nil {
			t.Fatal(err)
		}
		items := Site{SellerName: "x", Boilerplate: []string{"誠摯邀請"}}.ParseList(doc, c.category)
		if len(items) < c.min {
			t.Fatalf("%s: expected >= %d items, got %d", c.fixture, c.min, len(items))
		}
		withPrice, withBrand, withImg, withRef := 0, 0, 0, 0
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
			crawler.EnrichFromTitle(&l)
			if l.ReferenceNumber != "" {
				withRef++
			}
		}
		t.Logf("%s: items=%d price=%d brand=%d img=%d ref=%d first=%q", c.fixture, len(items), withPrice, withBrand, withImg, withRef, items[0].Title)
		if withPrice < len(items)*9/10 || withBrand < len(items)/2 || withImg < len(items)/2 {
			t.Errorf("%s: too few parsed fields: price=%d brand=%d img=%d of %d", c.fixture, withPrice, withBrand, withImg, len(items))
		}
	}
}
