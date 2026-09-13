package rasin

import (
	"os"
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

func TestParseListFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/list.html")
	if err != nil {
		t.Skip("fixture missing")
	}
	doc, err := crawler.ParseHTML(body, "https://www.rasin.co.jp/SHOP/list.php?Search=&Type=01&PAGE=1")
	if err != nil {
		t.Fatal(err)
	}
	if !HasNext(doc) {
		t.Error("first page should link to a next page")
	}
	items, total := ParseList(doc)
	if total != 40 || len(items) != 40 {
		t.Fatalf("expected 40 priced cards of 40, got %d of %d", len(items), total)
	}
	withBrand, withRef, withImg, withCond, withGender := 0, 0, 0, 0, 0
	for i := range items {
		l := &items[i]
		if l.ExternalID == "" || l.URL == "" || l.Title == "" || l.Price == nil {
			t.Errorf("incomplete listing: %+v", l)
		}
		crawler.EnrichFromTitle(l)
		if l.BrandSlug != "" {
			withBrand++
		}
		if l.ReferenceNumber != "" {
			withRef++
		}
		if len(l.ImageURLs) > 0 {
			withImg++
		}
		if l.Condition != model.ConditionUnknown {
			withCond++
		}
		if l.Gender != "" && l.Gender != model.GenderUnknown {
			withGender++
		}
	}
	t.Logf("items=%d brand=%d ref=%d img=%d cond=%d gender=%d sample=%+v", len(items), withBrand, withRef, withImg, withCond, withGender, items[0])
	if withBrand < 36 || withRef < 34 || withImg < 38 || withCond < 38 || withGender < 30 {
		t.Errorf("too few parsed fields: brand=%d ref=%d img=%d cond=%d gender=%d of %d", withBrand, withRef, withImg, withCond, withGender, len(items))
	}
	// first card: Rolex Oyster Perpetual 116000, pre-owned, ¥1,480,000
	f := items[0]
	if f.ExternalID != "U-116000STBDP" || f.BrandSlug != "rolex" || f.ReferenceNumber != "116000" || *f.Price != 1480000 || f.Condition != model.ConditionGood || f.URL != "https://www.rasin.co.jp/SHOP/U-116000STBDP.html" {
		t.Errorf("first card parsed wrong: %+v", f)
	}
}

func TestSoldOutPageFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/list_soldout.html")
	if err != nil {
		t.Skip("fixture missing")
	}
	doc, err := crawler.ParseHTML(body, "https://www.rasin.co.jp/SHOP/list.php?Search=&Type=01&PAGE=100")
	if err != nil {
		t.Fatal(err)
	}
	items, total := ParseList(doc)
	if total != 40 || len(items) != 0 {
		t.Errorf("a page of ¥0 cards must yield no listings: got %d of %d", len(items), total)
	}
}
