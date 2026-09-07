package lips

import (
	"os"
	"strings"
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

func TestParseListFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/list.html")
	if err != nil {
		t.Skip("fixture missing")
	}
	doc, err := crawler.ParseHTML(body, "https://lips-online.jp/ec/products/list?category_id=1")
	if err != nil {
		t.Fatal(err)
	}
	items := ParseList(doc)
	if len(items) != 40 {
		t.Fatalf("expected 40 cards, got %d", len(items))
	}
	withPrice, withBrand, withRef, withImg, withCond, withBox := 0, 0, 0, 0, 0, 0
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
		if l.Condition != "" && l.Condition != model.ConditionUnknown {
			withCond++
		}
		if l.HasBox != nil && *l.HasBox {
			withBox++
		}
	}
	t.Logf("items=%d price=%d brand=%d ref=%d img=%d cond=%d box=%d sample=%+v", len(items), withPrice, withBrand, withRef, withImg, withCond, withBox, items[1])
	if withPrice < 38 || withBrand < 38 || withRef < 38 || withImg < 38 || withCond < 38 || withBox < 20 {
		t.Errorf("too few parsed fields: price=%d brand=%d ref=%d img=%d cond=%d box=%d of %d", withPrice, withBrand, withRef, withImg, withCond, withBox, len(items))
	}
	// second card in the capture: Rolex Daytona 126500LN, 新品, ¥1,568,000 → the grade must map to "new"
	// and an "SAランク" card to very_good
	if items[1].ReferenceNumber != "126500LN" || items[1].Condition != model.ConditionNew || items[1].BrandSlug != "rolex" {
		t.Errorf("second card parsed wrong: %+v", items[1])
	}
	sa := 0
	for _, l := range items {
		if l.Condition == model.ConditionVeryGood {
			sa++
		}
		if strings.ContainsAny(l.ReferenceNumber, "()（）") {
			t.Errorf("serial suffix left in reference %q", l.ReferenceNumber)
		}
	}
	if sa == 0 {
		t.Error("no SA/A rank cards mapped to very_good")
	}
}
