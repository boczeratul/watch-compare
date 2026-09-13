package quark

import (
	"os"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

func parse(t *testing.T, name string) (*goquery.Document, []model.Listing) {
	t.Helper()
	body, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Skip("fixture missing")
	}
	doc, err := crawler.ParseHTML(body, "https://www.909.co.jp/rolex_search/result.html?btn_val=kensaku_s&freeword_s=&p=0&sort=NEW&snum=100")
	if err != nil {
		t.Fatal(err)
	}
	return doc, ParseList(doc)
}

func TestNewPageFixture(t *testing.T) {
	doc, items := parse(t, "list_new.html")
	if Total(doc) < 1000 {
		t.Errorf("total = %d", Total(doc))
	}
	if len(items) != 100 {
		t.Fatalf("expected 100 items, got %d", len(items))
	}
	withPrice, withRef, withImg := 0, 0, 0
	for _, l := range items {
		if l.ExternalID == "" || l.URL == "" || l.Title == "" || l.BrandSlug != "rolex" || l.Condition != model.ConditionNew {
			t.Errorf("bad listing: %+v", l)
		}
		if l.Price != nil {
			withPrice++
		}
		if l.ReferenceNumber != "" {
			withRef++
		}
		if len(l.ImageURLs) > 0 {
			withImg++
		}
	}
	t.Logf("new: items=%d price=%d ref=%d img=%d sample=%+v", len(items), withPrice, withRef, withImg, items[0])
	if withPrice < 95 || withRef < 98 || withImg < 98 {
		t.Errorf("too few fields: price=%d ref=%d img=%d", withPrice, withRef, withImg)
	}
	if f := items[0]; f.ExternalID != "DD-149" || f.ReferenceNumber != "128238" || f.Price == nil || *f.Price != 39999900 || f.Model != "デイデイト36" || f.URL != "https://www.909.co.jp/rolex_catalog/daydate_128238_pzl_00000100_psd.html" {
		t.Errorf("first new item parsed wrong: %+v", f)
	}
}

func TestUsedPageFixture(t *testing.T) {
	_, items := parse(t, "list_used.html")
	// 100 cards on the page, 2 of them vintage pieces already marked SOLD OUT
	if len(items) != 98 {
		t.Fatalf("expected 98 unsold items, got %d", len(items))
	}
	for _, l := range items {
		if l.Price == nil {
			t.Errorf("unsold item without price: %+v", l)
		}
		if strings.Contains(l.ExternalID, "/") {
			t.Errorf("id fell back to a path: %s", l.ExternalID)
		}
	}
	used, withYear, withStore, withPrice := 0, 0, 0, 0
	for _, l := range items {
		if l.Condition == model.ConditionGood || l.Condition == model.ConditionFair {
			used++
		}
		if l.Year != nil {
			withYear++
		}
		if l.LocationCity != "" {
			withStore++
		}
		if l.Price != nil {
			withPrice++
		}
	}
	t.Logf("used: items=%d used=%d year=%d store=%d price=%d sample=%+v", len(items), used, withYear, withStore, withPrice, items[0])
	if used == 0 || withYear < used/2 || withStore < used/2 || withPrice < len(items)*9/10 {
		t.Errorf("too few fields: used=%d year=%d store=%d price=%d of %d", used, withYear, withStore, withPrice, len(items))
	}
}
