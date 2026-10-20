package commitwatch

import (
	"os"
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

// testdata/list.html is page 1 of /collections/all?sort_by=created-descending captured 2026-10-20:
// 48 cards, 22 for sale, the rest SOLD OUT or RESERVED.
func TestParseListFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/list.html")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := crawler.ParseHTML(body, baseURL+"/collections/all?sort_by=created-descending")
	if err != nil {
		t.Fatal(err)
	}
	cards, items := ParseList(doc)
	if cards != 48 || len(items) != 22 {
		t.Fatalf("cards=%d listings=%d, want 48/22", cards, len(items))
	}
	var d model.Listing
	for _, l := range items {
		if l.ExternalID == "" || l.Price == nil || l.PriceExclTax == nil || l.BrandSlug == "" || len(l.ImageURLs) == 0 {
			t.Errorf("incomplete listing: %+v", l)
		}
		if l.ExternalID == "10348078432418" {
			d = l
		}
	}
	if d.URL != baseURL+"/products/d6y8dkkt" || d.BrandSlug != "rolex" || d.ReferenceNumber != "126519LN" ||
		*d.Price != 15510000 || *d.PriceExclTax != 14100000 || d.Condition != model.ConditionUnworn ||
		d.Title != "ロレックス デイトナ Ref.126519LN ホワイトゴールド メテオライト文字盤 未使用" ||
		d.ImageURLs[0] != "https://commit-watch.co.jp/cdn/shop/files/d6y8dkkt-010.jpg?v=1792468302&width=1920" {
		t.Errorf("d6y8dkkt: %+v %v %v", d, *d.Price, d.ImageURLs)
	}
}
