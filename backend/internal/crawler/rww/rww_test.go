package rww

import (
	"os"
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

func TestParseCategoriesFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/categories.json")
	if err != nil {
		t.Skip("fixture missing")
	}
	cats, err := ParseCategories(body)
	if err != nil {
		t.Fatal(err)
	}
	// new/used Rolex + new/used other brands; bags, shops and 其他 are left out
	if len(cats) != 4 {
		t.Errorf("watch categories = %v, want 4", cats)
	}
}

func TestParseListFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/list.json")
	if err != nil {
		t.Skip("fixture missing")
	}
	res, err := ParseList(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Records) != 100 || res.TotalPages < 10 {
		t.Fatalf("envelope: records=%d ttlpage=%d", len(res.Records), res.TotalPages)
	}
	var items []model.Listing
	withPrice, withBrand, withRef, withImg, withCond, withYear, withDia := 0, 0, 0, 0, 0, 0, 0
	for _, r := range res.Records {
		l, ok := ToListing(r)
		if !ok {
			continue
		}
		crawler.EnrichFromTitle(&l)
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
		if l.Year != nil {
			withYear++
		}
		if l.CaseDiameterMM != nil {
			withDia++
		}
	}
	t.Logf("items=%d price=%d brand=%d ref=%d img=%d cond=%d year=%d dia=%d sample=%+v", len(items), withPrice, withBrand, withRef, withImg, withCond, withYear, withDia, items[0])
	if len(items) < 90 || withPrice < 85 || withBrand < 90 || withRef < 90 || withImg < 90 || withCond < 90 || withDia < 80 {
		t.Errorf("too few parsed fields: n=%d price=%d brand=%d ref=%d img=%d cond=%d dia=%d", len(items), withPrice, withBrand, withRef, withImg, withCond, withDia)
	}
	f := items[0] // Daytona 40mm 全新 Rolex 126519LN 黑面 (2026年)(尖沙咀店)
	if f.BrandSlug != "rolex" || f.ReferenceNumber != "126519LN" || f.Condition != model.ConditionNew || f.Model != "Daytona" || f.Price == nil || f.Currency != "HKD" || f.CaseDiameterMM == nil || *f.CaseDiameterMM != 40 {
		t.Errorf("first record parsed wrong: %+v", f)
	}
	if f.URL != "https://rwwwatch.com/app/web/product_detail.php?linkid="+f.ExternalID {
		t.Errorf("bad url %s", f.URL)
	}
}

func TestSoldAndCondition(t *testing.T) {
	if _, ok := ToListing(Record{LinkID: "x", Name: "(已售)Daytona系列 40mm 全新未用品 Rolex 116508 綠面(2022年)(尖沙咀店)"}); ok {
		t.Error("sold piece must be skipped")
	}
	cases := map[string]model.Condition{
		"Daytona系列 40mm 全新 Rolex 126500LN":                   model.ConditionNew,
		"Daytona系列 40mm 全新未用品 Rolex 116508":                  model.ConditionUnworn,
		"Cellini系列 39mm 二手99%新 Rolex 50529黑":                 model.ConditionVeryGood,
		"Aquanaut系列 40.8mm 二手 95%新 Patek Philippe 5164R-001": model.ConditionGood,
		"GMT系列 40mm 二手85%新 Rolex 16710":                      model.ConditionFair,
		"Something 二手 Rolex 1601":                            model.ConditionGood,
	}
	for in, want := range cases {
		if got := condition(in); got != want {
			t.Errorf("condition(%q) = %s, want %s", in, got, want)
		}
	}
	l, _ := ToListing(Record{LinkID: "y", Name: "Cellini系列 39mm 二手99%新 Rolex 50529黑(2021年)(尖沙咀店)", MinFinPrice: "124800",
		Desc: "<p>型號: Rolex&nbsp;<span>50529</span></p><p>配件: Full Set, 全齊</p><p>狀態: 新卡!!, 二手99%新, 2021年</p><p>***部分產品價格***</p>"})
	if l.ReferenceNumber != "50529" || l.HasBox == nil || !*l.HasBox || l.HasPapers == nil || !*l.HasPapers || *l.Price != 124800 || l.Model != "Cellini" {
		t.Errorf("description parsing: %+v", l)
	}
}
