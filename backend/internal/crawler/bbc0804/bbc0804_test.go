package bbc0804

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

func load(t *testing.T, name, url string) *goquery.Document {
	t.Helper()
	body, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Skip("fixture missing")
	}
	doc, err := crawler.ParseHTML(body, url)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestCategoriesFixture(t *testing.T) {
	cats := Categories(load(t, "home.html", "https://www.bbc0804.com.tw/goods.html"))
	slugs := map[string]bool{}
	for _, c := range cats {
		slugs[c.Slug] = true
	}
	for _, want := range []string{"rolex", "djdd", "rolexman2", "rolexwoman", "tudor", "omega", "3p", "longines", "oris", "chopard", "iwc", "diamongold", "g2"} {
		if !slugs[want] {
			t.Errorf("category %s missing from %v", want, cats)
		}
	}
	for _, bad := range []string{"diamongia", "diamon100", "moto", "g6", "g4", "sss3", "g3", "pearl", "-k-", "pienpai", "goods"} {
		if slugs[bad] {
			t.Errorf("non-watch category %s included", bad)
		}
	}
}

func TestListAndDetailFixture(t *testing.T) {
	items := ParseList(load(t, "list.html", "https://www.bbc0804.com.tw/rolex.html"), "勞力士男錶 運動款")
	if len(items) != 27 {
		t.Fatalf("expected 27 cards, got %d", len(items))
	}
	withPrice, withRef, withYear, withDia := 0, 0, 0, 0
	for _, l := range items {
		if l.ExternalID == "" || l.URL == "" || l.Title == "" || l.BrandSlug != "rolex" || len(l.ImageURLs) != 1 || l.Currency != "TWD" {
			t.Errorf("incomplete card: %+v", l)
		}
		if l.Price != nil {
			withPrice++
		}
		if l.ReferenceNumber != "" {
			withRef++
		}
		if l.Year != nil {
			withYear++
		}
		if l.CaseDiameterMM != nil {
			withDia++
		}
	}
	t.Logf("items=%d price=%d ref=%d year=%d dia=%d sample=%+v", len(items), withPrice, withRef, withYear, withDia, items[0])
	if withPrice < 25 || withRef < 22 || withYear < 18 || withDia < 22 {
		t.Errorf("too few parsed fields: price=%d ref=%d year=%d dia=%d of %d", withPrice, withRef, withYear, withDia, len(items))
	}
	f := items[0] // Rolex Daytona 116503, $64.8萬, 40 mm, papers dated 2021
	if f.ExternalID != "4123627" || f.ReferenceNumber != "116503" || f.Price == nil || *f.Price != 648000 || f.Year == nil || *f.Year != 2021 ||
		f.CaseDiameterMM == nil || *f.CaseDiameterMM != 40 || f.URL != "https://www.bbc0804.com.tw/product-detail-4123627.html" ||
		f.ImageURLs[0] != "https://static.iyp.tw/18440/products/photooriginal-3183065-V5EGe.jpg" {
		t.Errorf("first card parsed wrong: %+v", f)
	}
	if strings.HasSuffix(f.Title, "萬") {
		t.Errorf("price left in title: %q", f.Title)
	}
	ApplyDetail(&f, load(t, "item.html", f.URL))
	t.Logf("detail: %+v", f)
	// this piece's line reads "動力來源：4130機芯" with no winding word, so the movement stays unknown
	if f.HasBox == nil || !*f.HasBox || f.HasPapers == nil || !*f.HasPapers || f.Movement == model.MovementQuartz || f.Description == "" || len(f.ImageURLs) < 2 {
		t.Errorf("detail parsed wrong: %+v", f)
	}
	var attrs map[string]any
	if err := json.Unmarshal(f.Attributes, &attrs); err != nil || attrs["accessories"] != "原盒1 保卡1 說明書2 吊牌2" || attrs["boughtFrom"] != "國外AD" {
		t.Errorf("detail attributes: %s", f.Attributes)
	}
}

func TestParsePrice(t *testing.T) {
	for in, want := range map[string]float64{"售價:$64.8萬": 648000, "$7.2萬": 72000, "$58,000": 58000, "售價:$102.8萬 ": 1028000} {
		if got, ok := ParsePrice(in); !ok || got != want {
			t.Errorf("ParsePrice(%q) = %v %v, want %v", in, got, ok, want)
		}
	}
	if _, ok := ParsePrice("售價:電洽"); ok {
		t.Error("no number must not parse")
	}
}
