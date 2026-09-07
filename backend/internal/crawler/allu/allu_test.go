package allu

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
	if res.TotalCount < 1000 || len(res.Items) != 30 || res.Limit != 30 {
		t.Fatalf("unexpected envelope: total=%d items=%d limit=%d", res.TotalCount, len(res.Items), res.Limit)
	}
	withBrand, withRef, withImg, withCond, dealers := 0, 0, 0, 0, 0
	var listings []model.Listing
	for _, it := range res.Items {
		l, ok := ToListing(it)
		if !ok {
			continue
		}
		listings = append(listings, l)
		if l.ExternalID == "" || l.URL == "" || l.Title == "" || l.Price == nil {
			t.Errorf("incomplete listing: %+v", l)
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
		if l.SellerType == "dealer" {
			dealers++
		}
	}
	t.Logf("items=%d brand=%d ref=%d img=%d cond=%d dealers=%d sample=%+v", len(listings), withBrand, withRef, withImg, withCond, dealers, listings[0])
	if len(listings) < 28 || withBrand < 28 || withRef < 28 || withImg < 28 || withCond < 28 || dealers < 28 {
		t.Errorf("too few parsed fields: n=%d brand=%d ref=%d img=%d cond=%d dealers=%d", len(listings), withBrand, withRef, withImg, withCond, dealers)
	}
	// first capture item: Hermès Cape Cod CC1.312, rank A, ¥359,800
	first := listings[0]
	if first.ExternalID != "2099173" || first.ReferenceNumber != "CC1.312" || first.Condition != model.ConditionVeryGood || *first.Price != 359800 || first.BrandSlug != "hermes" {
		t.Errorf("first item parsed wrong: %+v", first)
	}
	if first.URL != "https://allu-official.com/jp/ja/market/items/2099173/" {
		t.Errorf("bad url %s", first.URL)
	}
}

func TestRankToCondition(t *testing.T) {
	cases := map[string]model.Condition{"N": model.ConditionUnworn, "S": model.ConditionNew, "SA": model.ConditionVeryGood, "A": model.ConditionVeryGood, "AB": model.ConditionGood, "B": model.ConditionGood, "BC": model.ConditionFair, "C": model.ConditionFair, "J": model.ConditionPoor, "": model.ConditionUnknown}
	for in, want := range cases {
		if got := rankToCondition(in); got != want {
			t.Errorf("rank %q = %s, want %s", in, got, want)
		}
	}
}
