package bellemonde

import (
	"os"
	"strings"
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

func TestParseListFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/list.html")
	if err != nil {
		t.Skip("fixture missing")
	}
	doc, err := crawler.ParseHTML(body, "https://bellemonde.tokyo/view/category/all_items?sort=order&page=1")
	if err != nil {
		t.Fatal(err)
	}
	if Total(doc) < 1000 {
		t.Errorf("total = %d", Total(doc))
	}
	items, total := ParseList(doc)
	// 50 cards: 16 sold out and one 0円 gift-wrapping service, which the price check drops too
	if total != 50 || len(items) != 33 {
		t.Fatalf("expected 33 in-stock watches of 50 cards, got %d of %d", len(items), total)
	}
	withBrand, withRef, withImg, withCond, withPapers := 0, 0, 0, 0, 0
	for i := range items {
		l := &items[i]
		if l.ExternalID == "" || l.URL == "" || l.Title == "" || l.Price == nil || *l.Price <= 0 {
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
		if l.HasPapers != nil && *l.HasPapers {
			withPapers++
		}
	}
	t.Logf("items=%d brand=%d ref=%d img=%d cond=%d papers=%d sample=%+v", len(items), withBrand, withRef, withImg, withCond, withPapers, items[1])
	if withBrand < 30 || withRef < 24 || withImg < 32 || withCond < 32 || withPapers < 10 {
		t.Errorf("too few parsed fields: brand=%d ref=%d img=%d cond=%d papers=%d of %d", withBrand, withRef, withImg, withCond, withPapers, len(items))
	}
	// second card: IWC Pilot's Watch Chronograph 41 IW388101, 【美品】【中古】, papers, ¥ price present
	s := items[1]
	if s.ExternalID != "000000005395" || s.BrandSlug != "iwc" || s.ReferenceNumber != "IW388101" || s.Condition != model.ConditionVeryGood || s.HasPapers == nil || !*s.HasPapers || s.URL != "https://bellemonde.tokyo/view/item/000000005395" {
		t.Errorf("second card parsed wrong: %+v", s)
	}
	for _, l := range items {
		if strings.HasPrefix(l.Title, "【") {
			t.Errorf("markers left in title: %q", l.Title)
		}
	}
}
