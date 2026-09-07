package wristcheck

import (
	"os"
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

func TestParseSearchFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/search.json")
	if err != nil {
		t.Skip("fixture missing")
	}
	res, err := ParseSearch(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) != 100 || res.NbPages < 10 || res.NbHits < 1000 {
		t.Fatalf("envelope: hits=%d pages=%d total=%d", len(res.Hits), res.NbPages, res.NbHits)
	}
	var items []model.Listing
	withPrice, withBrand, withRef, withImg, withCond, hk, ny := 0, 0, 0, 0, 0, 0, 0
	for _, h := range res.Hits {
		l, ok := ToListing(h)
		if !ok {
			continue
		}
		items = append(items, l)
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
		if l.Condition != model.ConditionUnknown {
			withCond++
		}
		switch l.LocationCountry {
		case "HK":
			hk++
		case "US":
			ny++
		default:
			t.Errorf("listing %s has no location", l.ExternalID)
		}
	}
	t.Logf("items=%d price=%d brand=%d ref=%d img=%d cond=%d hk=%d ny=%d sample=%+v", len(items), withPrice, withBrand, withRef, withImg, withCond, hk, ny, items[0])
	if len(items) != 100 || withPrice < 95 || withBrand < 90 || withRef < 90 || withImg < 95 || withCond < 95 || hk == 0 || ny == 0 {
		t.Errorf("too few parsed fields: n=%d price=%d brand=%d ref=%d img=%d cond=%d hk=%d ny=%d", len(items), withPrice, withBrand, withRef, withImg, withCond, hk, ny)
	}
	for _, l := range items {
		if l.Currency != "HKD" || l.Price != nil && *l.Price < 1000 {
			t.Errorf("odd price on %s: %v %s", l.ExternalID, l.Price, l.Currency)
		}
	}
}

func TestSoldAndCondition(t *testing.T) {
	if _, ok := ToListing(Hit{ObjectID: "x", Slug: "s", Status: "sold"}); ok {
		t.Error("sold hit must be skipped")
	}
	g := func(s string) *string { return &s }
	cases := []struct {
		h    Hit
		want model.Condition
	}{
		{Hit{Condition: "WATCH_UNWORN"}, model.ConditionUnworn},
		{Hit{Condition: "WATCH_WORN", WCGrade: g("BRAND_NEW")}, model.ConditionNew},
		{Hit{Condition: "WATCH_WORN", WCGrade: g("9.0")}, model.ConditionVeryGood},
		{Hit{Condition: "WATCH_WORN", WCGrade: g("8.0")}, model.ConditionGood},
		{Hit{Condition: "WATCH_WORN", WCGrade: g("6.0")}, model.ConditionFair},
		{Hit{Condition: "WATCH_WORN"}, model.ConditionGood},
		{Hit{}, model.ConditionUnknown},
	}
	for _, c := range cases {
		if got := condition(c.h); got != c.want {
			t.Errorf("condition(%+v) = %s, want %s", c.h, got, c.want)
		}
	}
}
