package jdpawn

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
	doc, err := crawler.ParseHTML(body, "https://www.jdpawn.com.tw/luxury/Stuffs/Watch")
	if err != nil {
		t.Fatal(err)
	}
	if Pages(doc) != 13 {
		t.Errorf("pages = %d, want 13", Pages(doc))
	}
	items := ParseList(doc)
	if len(items) != 16 {
		t.Fatalf("expected 16 pieces, got %d", len(items))
	}
	withPrice, withBrand, withRef, withDia, withMove, withBox, withCity := 0, 0, 0, 0, 0, 0, 0
	for i := range items {
		l := &items[i]
		if l.ExternalID == "" || l.URL == "" || l.Title == "" || len(l.ImageURLs) != 1 {
			t.Errorf("incomplete listing: %+v", l)
		}
		crawler.EnrichFromTitle(l)
		if l.Price != nil {
			withPrice++
		}
		if l.BrandSlug != "" {
			withBrand++
		}
		if l.ReferenceNumber != "" {
			withRef++
		}
		if l.CaseDiameterMM != nil {
			withDia++
		}
		if l.Movement != "" && l.Movement != model.MovementUnknown {
			withMove++
		}
		if l.HasBox != nil {
			withBox++
		}
		if l.LocationCity != "" {
			withCity++
		}
	}
	t.Logf("items=%d price=%d brand=%d ref=%d dia=%d move=%d box=%d city=%d sample=%+v", len(items), withPrice, withBrand, withRef, withDia, withMove, withBox, withCity, items[1])
	if withPrice < 16 || withBrand < 16 || withRef < 12 || withDia < 15 || withMove < 15 || withBox < 10 || withCity < 15 {
		t.Errorf("too few parsed fields: price=%d brand=%d ref=%d dia=%d move=%d box=%d city=%d of %d", withPrice, withBrand, withRef, withDia, withMove, withBox, withCity, len(items))
	}
	// T848: Cartier Ballon Bleu W6920042, NT$126,000, automatic, 42 mm, box and papers, Taichung
	var f *model.Listing
	for i := range items {
		if items[i].ExternalID == "T848" {
			f = &items[i]
		}
	}
	if f == nil {
		t.Fatal("T848 not parsed")
	}
	if f.BrandSlug != "cartier" || f.ReferenceNumber != "W6920042" || f.Price == nil || *f.Price != 126000 || f.Movement != model.MovementAutomatic ||
		f.CaseDiameterMM == nil || *f.CaseDiameterMM != 42 || f.HasBox == nil || !*f.HasBox || f.HasPapers == nil || !*f.HasPapers || f.LocationCity != "Taichung" ||
		f.URL != "https://www.jdpawn.com.tw/luxury/Stuffs/SingleProduct/T848" || f.ImageURLs[0] != "https://www.jdpawn.com.tw/luxury/StuffPictures/T848.jpg" {
		t.Errorf("T848 parsed wrong: %+v", f)
	}
}
