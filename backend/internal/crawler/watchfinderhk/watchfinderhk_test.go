package watchfinderhk

import (
	"os"
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
)

func TestParseListFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/list.html")
	if err != nil {
		t.Skip("fixture missing")
	}
	doc, err := crawler.ParseHTML(body, "https://www.watchfinder.hk/watches")
	if err != nil {
		t.Fatal(err)
	}
	if !HasNext(doc) {
		t.Error("first page should link to a next page")
	}
	items := ParseList(doc)
	if len(items) != 42 {
		t.Fatalf("expected 42 cards, got %d", len(items))
	}
	withPrice, withBrand, withRef, withImg, withYear, withBox := 0, 0, 0, 0, 0, 0
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
		if l.Year != nil {
			withYear++
		}
		if l.HasBox != nil {
			withBox++
		}
	}
	t.Logf("items=%d price=%d brand=%d ref=%d img=%d year=%d box=%d sample=%+v", len(items), withPrice, withBrand, withRef, withImg, withYear, withBox, items[0])
	if withPrice < 40 || withBrand < 38 || withRef < 40 || withImg < 40 || withYear < 35 || withBox < 40 {
		t.Errorf("too few parsed fields: price=%d brand=%d ref=%d img=%d year=%d box=%d of %d", withPrice, withBrand, withRef, withImg, withYear, withBox, len(items))
	}
	// first card: Jaeger-LeCoultre Master Ultra Thin 114842J, HK$180,540, 2025, box and papers
	f := items[0]
	if f.ExternalID != "437005" || f.BrandSlug != "jaeger-lecoultre" || f.ReferenceNumber != "114842J" || f.Price == nil || *f.Price != 180540 ||
		f.Currency != "HKD" || f.Year == nil || *f.Year != 2025 || f.HasBox == nil || !*f.HasBox || f.HasPapers == nil || !*f.HasPapers {
		t.Errorf("first card parsed wrong: %+v", f)
	}
	if f.URL != "https://www.watchfinder.hk/watches/jaeger-lecoultre/master-ultra-thin/114842j/437005" || len(f.ImageURLs) != 1 || f.ImageURLs[0] != "https://www.watchfinder.hk/media/catalog/product/W/a/Watch-1-Jaeger-LeCoultre-MasterUltraThin-114842J-437005-260817-140401858.jpg" {
		t.Errorf("url/image: %s %v", f.URL, f.ImageURLs)
	}
}
