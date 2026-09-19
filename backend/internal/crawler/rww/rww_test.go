package rww

import (
	"encoding/json"
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

// 配件 is an exhaustive accessories field: "淨錶" / "凈錶" (watch only) means no box and no papers,
// and the value must stop where the next label (狀態, 年份, 銷售地點) starts.
func TestAccessoriesField(t *testing.T) {
	cases := []struct {
		desc        string
		box, papers bool
		accessories string
	}{
		{"<p>型號: Rolex 116610LN</p><p>配件: 淨錶</p><p>狀態: 40mm, 二手95%新</p><p>銷售地點: 尖沙咀 店</p>", false, false, "淨錶"},
		{"<p>型號: Rolex 16610</p><p>配件:&nbsp;<font>凈錶</font></p><p>狀態: 40mm, 新卡!!, 二手95%新</p>", false, false, "凈錶"},
		{"<p>型號: Rolex 1601</p><p>配件: 淨錶, 跟20格, 跟18K代用帶</p><p>狀態: 36mm</p>", false, false, "淨錶, 跟20格, 跟18K代用帶"},
		{"<p>型號: Patek Philippe 5146J</p><p>配件: 一錶一紙</p><p>狀態: 39mm</p>", false, true, "一錶一紙"},
		{"<p>型號: Rolex 126610LN</p><p>配件: Full Set, 全齊, 跟AD單</p><p>狀態: 41mm, 100%全新 年份: 2026年 銷售地點: 尖沙咀店</p>", true, true, "Full Set, 全齊, 跟AD單"},
	}
	for _, c := range cases {
		l, ok := ToListing(Record{LinkID: "z", Name: "Submariner系列 40mm 二手95%新 Rolex 116610LN(尖沙咀店)", MinFinPrice: "66800", Desc: c.desc})
		if !ok {
			t.Fatalf("listing skipped: %s", c.desc)
		}
		if l.HasBox == nil || *l.HasBox != c.box || l.HasPapers == nil || *l.HasPapers != c.papers {
			t.Errorf("%q: box=%v papers=%v, want %v/%v", c.accessories, l.HasBox, l.HasPapers, c.box, c.papers)
		}
		var attrs map[string]any
		if err := json.Unmarshal(l.Attributes, &attrs); err != nil || attrs["accessories"] != c.accessories {
			t.Errorf("accessories attribute = %q, want %q", attrs["accessories"], c.accessories)
		}
	}
	// no 配件 line at all: unknown, not "no"
	l, _ := ToListing(Record{LinkID: "n", Name: "Datejust系列 36mm 二手 Rolex 1601(尖沙咀店)", MinFinPrice: "30000", Desc: "<p>型號: Rolex 1601</p><p>狀態: 36mm</p>"})
	if l.HasBox != nil || l.HasPapers != nil {
		t.Errorf("missing 配件 must stay unknown, got %v/%v", l.HasBox, l.HasPapers)
	}
}
