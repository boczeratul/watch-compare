package kitamura

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
	doc, err := crawler.ParseHTML(body, "https://shop.kitamura.jp/watch/buy/search/?page=1&sort=used_update_dt&pageSize=90")
	if err != nil {
		t.Fatal(err)
	}
	if LastPage(doc) < 2 {
		t.Errorf("last page = %d", LastPage(doc))
	}
	items := ParseList(doc)
	if len(items) != 90 {
		t.Fatalf("expected 90 cards, got %d", len(items))
	}
	withPrice, withBrand, withRef, withImg, withCity := 0, 0, 0, 0, 0
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
		if l.LocationCity != "" {
			withCity++
		}
		if l.Condition == model.ConditionUnknown {
			t.Errorf("no condition on %s", l.ExternalID)
		}
	}
	t.Logf("items=%d price=%d brand=%d ref=%d img=%d city=%d sample=%+v", len(items), withPrice, withBrand, withRef, withImg, withCity, items[2])
	if withPrice < 88 || withBrand < 80 || withRef < 85 || withImg < 88 || withCity < 50 { // the online shop carries no city
		t.Errorf("too few parsed fields: price=%d brand=%d ref=%d img=%d city=%d of %d", withPrice, withBrand, withRef, withImg, withCity, len(items))
	}
	// third card: Rolex Explorer II 16570, grade A, ¥1,348,000, Toyama shop
	f := items[2]
	if f.ExternalID != "2446010040144" || f.BrandSlug != "rolex" || f.ReferenceNumber != "16570" || f.Price == nil || *f.Price != 1348000 || f.Condition != model.ConditionGood || f.LocationCity != "富山" || f.URL != "https://shop.kitamura.jp/watch/buy/item/2446010040144/" {
		t.Errorf("third card parsed wrong: %+v", f)
	}
}

func TestGrades(t *testing.T) {
	for g, want := range map[string]model.Condition{"AA": model.ConditionVeryGood, "A": model.ConditionGood, "AB": model.ConditionGood, "B": model.ConditionFair, "C": model.ConditionPoor, "": model.ConditionGood} {
		if got := gradeToCondition(g); got != want {
			t.Errorf("grade %q = %s, want %s", g, got, want)
		}
	}
}
