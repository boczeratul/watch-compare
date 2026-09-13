package watchandtime

import (
	"os"
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
	cats := Categories(load(t, "home.html", "https://www.watchandtime.com/index.asp"))
	if len(cats) != 30 {
		t.Errorf("expected 30 watch categories (31 minus jewellery), got %d: %v", len(cats), cats)
	}
	for _, c := range cats {
		if c.ID == "82" {
			t.Errorf("jewellery category not skipped: %+v", c)
		}
	}
}

func TestListAndDetailFixture(t *testing.T) {
	doc := load(t, "list.html", "https://www.watchandtime.com/products_list.asp?id=77")
	total, cur, last := Pagination(doc)
	if total != 74 || cur != 1 || last != 4 {
		t.Errorf("pagination = %d %d %d", total, cur, last)
	}
	items := ParseList(doc, "Rolex")
	if len(items) != 24 {
		t.Fatalf("expected 24 cards, got %d", len(items))
	}
	priced := 0
	for _, l := range items {
		if l.ExternalID == "" || l.URL == "" || l.BrandSlug != "rolex" || len(l.ImageURLs) != 1 || l.Currency != "TWD" {
			t.Errorf("incomplete card: %+v", l)
		}
		if l.Price != nil {
			priced++
		}
	}
	if priced < 20 { // a few cards say 電洽 (call for price)
		t.Errorf("only %d of %d cards priced", priced, len(items))
	}
	f := items[0]
	if f.ExternalID != "18597" || *f.Price != 355000 || f.URL != "https://www.watchandtime.com/products_open.asp?id=18597" || f.ImageURLs[0] != "https://www.watchandtime.com/product/product_big/m-16799.JPG" {
		t.Errorf("first card parsed wrong: %+v", f)
	}
	ApplyDetail(&f, load(t, "item.html", f.URL))
	crawler.EnrichFromTitle(&f)
	t.Logf("detail: %+v", f)
	if f.Title != `Rolex Oyster Perpetual Date GMT-Master 16750 "Pepsi". 40mm` || f.ReferenceNumber != "16750" || f.CaseMaterial != "不鏽鋼" ||
		f.Movement != model.MovementAutomatic || f.Gender != model.GenderMen || f.Year == nil || *f.Year != 1985 || f.Description == "" || len(f.ImageURLs) < 3 {
		t.Errorf("detail parsed wrong: %+v", f)
	}
	if f.CaseDiameterMM == nil || *f.CaseDiameterMM != 40 {
		t.Errorf("diameter: %v", f.CaseDiameterMM)
	}
}

func TestMaterialPrefix(t *testing.T) {
	doc, _ := crawler.ParseHTML([]byte(`<html><body>編號：M1 售價：NT$ 1元 CASE：18K Rose Gold. Saxonia 842.032 Factory Diamond Set Bezel. 37mm. 功能：男錶，自動機芯 附件狀況：附原裝錶盒，2015 年製造。│ 回首頁</body></html>`), "https://www.watchandtime.com/products_open.asp?id=1")
	l := model.Listing{}
	ApplyDetail(&l, doc)
	if l.CaseMaterial != "18K Rose Gold" || l.Title != "Saxonia 842.032 Factory Diamond Set Bezel. 37mm" || l.Year == nil || *l.Year != 2015 || l.HasBox == nil || !*l.HasBox {
		t.Errorf("parsed: %+v", l)
	}
}
