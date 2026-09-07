package housekihiroba

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

func TestBrandCategoriesFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/landing.html")
	if err != nil {
		t.Skip("fixture missing")
	}
	doc, err := crawler.ParseHTML(body, "https://housekihiroba.jp/shop/c/cwatch/")
	if err != nil {
		t.Fatal(err)
	}
	cats := BrandCategories(doc)
	if len(cats) < 15 || cats[0] != "c01rx" {
		t.Fatalf("brand categories = %v", cats)
	}
}

func TestParseListFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/list.html")
	if err != nil {
		t.Skip("fixture missing")
	}
	doc, err := crawler.ParseHTML(body, "https://housekihiroba.jp/shop/c/c01rx/")
	if err != nil {
		t.Fatal(err)
	}
	if !HasNext(doc) {
		t.Error("first Rolex page should link to a next page")
	}
	items := ParseList(doc)
	if len(items) != 32 {
		t.Fatalf("expected 32 cards in the goods list (ranking tiles excluded), got %d", len(items))
	}
	withPrice, withBrand, withRef, withImg, withCond, withDia, withMove := 0, 0, 0, 0, 0, 0, 0
	for _, l := range items {
		if l.ExternalID == "" || l.URL == "" || l.Title == "" {
			t.Errorf("incomplete listing: %+v", l)
		}
		if l.Price != nil {
			withPrice++
		}
		if l.BrandSlug == "rolex" {
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
		if l.CaseDiameterMM != nil {
			withDia++
		}
		if l.Movement != "" && l.Movement != model.MovementUnknown {
			withMove++
		}
	}
	t.Logf("items=%d price=%d brand=%d ref=%d img=%d cond=%d dia=%d move=%d sample=%+v", len(items), withPrice, withBrand, withRef, withImg, withCond, withDia, withMove, items[0])
	if withPrice < 30 || withBrand < 30 || withRef < 30 || withImg < 30 || withCond < 30 || withDia < 25 || withMove < 25 {
		t.Errorf("too few parsed fields: price=%d brand=%d ref=%d img=%d cond=%d dia=%d move=%d of %d", withPrice, withBrand, withRef, withImg, withCond, withDia, withMove, len(items))
	}
	// first card: GMT-Master II 126710GRNR, unused, ¥3,650,000
	f := items[0]
	if f.ExternalID != "663890001" || f.ReferenceNumber != "126710GRNR" || f.Condition != model.ConditionUnworn || f.Price == nil || *f.Price != 3650000 || f.Gender != model.GenderMen {
		t.Errorf("first card parsed wrong: %+v", f)
	}
	// a struck-through previous price must never be taken as the current price
	marked := 0
	for _, l := range items {
		var attrs struct {
			Previous string `json:"previousPrice"`
		}
		_ = json.Unmarshal(l.Attributes, &attrs)
		if attrs.Previous == "" {
			continue
		}
		marked++
		prev, _ := normalize.ParsePrice(attrs.Previous)
		if l.Price == nil || *l.Price >= prev {
			t.Errorf("reduced card %s: price %v not below previous %v", l.ExternalID, l.Price, prev)
		}
	}
	t.Logf("cards with a previous price: %d", marked)
}
