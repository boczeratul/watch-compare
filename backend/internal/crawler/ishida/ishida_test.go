package ishida

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
	doc, err := crawler.ParseHTML(body, "https://ishida-watch.com/c/bestvintage/v_brand")
	if err != nil {
		t.Fatal(err)
	}
	if !HasNext(doc) {
		t.Error("first page should link to a next page")
	}
	items := ParseList(doc)
	if len(items) != 20 {
		t.Fatalf("expected 20 cards, got %d", len(items))
	}
	withPrice, withBrand, withRef, withImg, withCond, withMove, withGender := 0, 0, 0, 0, 0, 0, 0
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
		if l.Condition != "" && l.Condition != model.ConditionUnknown {
			withCond++
		}
		if l.Movement != "" && l.Movement != model.MovementUnknown {
			withMove++
		}
		if l.Gender != "" && l.Gender != model.GenderUnknown {
			withGender++
		}
	}
	t.Logf("items=%d price=%d brand=%d ref=%d img=%d cond=%d move=%d gender=%d sample=%+v", len(items), withPrice, withBrand, withRef, withImg, withCond, withMove, withGender, items[0])
	if withPrice < 19 || withBrand < 17 || withRef < 15 || withImg < 19 || withCond < 19 || withMove < 15 || withGender < 15 {
		t.Errorf("too few parsed fields: price=%d brand=%d ref=%d img=%d cond=%d move=%d gender=%d of %d", withPrice, withBrand, withRef, withImg, withCond, withMove, withGender, len(items))
	}
	// first card: pre-owned Cartier Santos de Cartier Skeleton LM, WHSA0015, ¥3,190,000, automatic, men's
	f := items[0]
	if f.ExternalID != "1002900010942-h09" || f.BrandSlug != "cartier" || f.ReferenceNumber != "WHSA0015" || f.Condition != model.ConditionGood ||
		f.Price == nil || *f.Price != 3190000 || f.Movement != model.MovementAutomatic || f.Gender != model.GenderMen {
		t.Errorf("first card parsed wrong: %+v", f)
	}
	if f.URL != "https://ishida-watch.com/c/bestvintage/1002900010942-h09" || len(f.ImageURLs) < 2 {
		t.Errorf("url/images: %s %v", f.URL, f.ImageURLs)
	}
}
