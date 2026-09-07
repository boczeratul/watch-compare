package kenwatches

import (
	"os"
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

func TestParsePageFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/list.html")
	if err != nil {
		t.Skip("fixture missing")
	}
	p, err := ParsePage(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.WatchItems) != 20 || p.Count < 100 || p.CurrentPage != 1 {
		t.Fatalf("page: items=%d count=%d current=%d", len(p.WatchItems), p.Count, p.CurrentPage)
	}
	var items []model.Listing
	withPrice, withBrand, withRef, withImg, withMove, withGender, withDia, withYear := 0, 0, 0, 0, 0, 0, 0, 0
	for _, it := range p.WatchItems {
		l, ok := ToListing(it, p)
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
		if l.Movement != "" && l.Movement != model.MovementUnknown {
			withMove++
		}
		if l.Gender != "" && l.Gender != model.GenderUnknown {
			withGender++
		}
		if l.CaseDiameterMM != nil {
			withDia++
		}
		if l.Year != nil {
			withYear++
		}
	}
	t.Logf("items=%d price=%d brand=%d ref=%d img=%d move=%d gender=%d dia=%d year=%d sample=%+v", len(items), withPrice, withBrand, withRef, withImg, withMove, withGender, withDia, withYear, items[0])
	if len(items) != 20 || withPrice < 10 || withBrand < 18 || withRef < 18 || withImg < 18 || withMove < 15 || withGender < 15 || withDia < 15 || withYear < 15 {
		t.Errorf("too few parsed fields: n=%d price=%d brand=%d ref=%d img=%d move=%d gender=%d dia=%d year=%d", len(items), withPrice, withBrand, withRef, withImg, withMove, withGender, withDia, withYear)
	}
	// first item: Audemars Piguet Royal Oak Extra Thin 15202ST.OO.0944ST.02, 39mm, 2005, automatic, men's, price on request
	f := items[0]
	if f.ExternalID != "6a9bb301bdeee6d95ad806c1" || f.BrandSlug != "audemars-piguet" || f.ReferenceNumber != "15202ST.OO.0944ST.02" || f.Price != nil ||
		f.CaseDiameterMM == nil || *f.CaseDiameterMM != 39 || f.Year == nil || *f.Year != 2005 || f.Movement != model.MovementAutomatic || f.Gender != model.GenderMen ||
		f.HasBox == nil || !*f.HasBox || f.HasPapers == nil || !*f.HasPapers || f.Condition != model.ConditionGood {
		t.Errorf("first item parsed wrong: %+v", f)
	}
	if f.URL != "https://kenwatches.com/watch/6a9bb301bdeee6d95ad806c1" || len(f.ImageURLs) != 3 || f.ImageURLs[0] != "https://storage.googleapis.com/kenwatches-storage/watchItem/photos/2026/09/05/e46c5b80-a8f0-11f1-a6a1-4fbd7569a038.jpeg" {
		t.Errorf("url/images: %s %v", f.URL, f.ImageURLs)
	}
	// second item: Cartier WSBB0025 has no cash price but a listed price of HK$30,800
	if s := items[1]; s.ReferenceNumber != "WSBB0025" || s.Price == nil || *s.Price != 30800 {
		t.Errorf("listed-price fallback: %+v", s)
	}
}
