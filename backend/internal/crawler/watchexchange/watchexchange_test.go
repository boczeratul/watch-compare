package watchexchange

import (
	"os"
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

// testdata/new.json follows the shape of /r/Watchexchange/new.json?raw_json=1 (Reddit listing of
// t3 links); the posts are representative, not captured, since reddit.com is unreachable from CI.
func TestParseListingFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/new.json")
	if err != nil {
		t.Fatal(err)
	}
	posts, after, err := ParseListing(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 9 || after != "t3_1g5x0h9" {
		t.Fatalf("posts=%d after=%q", len(posts), after)
	}
	got := map[string]model.Listing{}
	for _, p := range posts {
		if l, ok := ToListing(p); ok {
			crawler.EnrichFromTitle(&l)
			l.ImageURLs = crawler.PhotoURLs(l.ImageURLs)
			got[l.ExternalID] = l
		}
	}
	// stickied META, WTB, Sold and removed posts are skipped
	for _, id := range []string{"1g5zz01", "1g5wk11", "1g5vb88", "1g5ua31"} {
		if _, ok := got[id]; ok {
			t.Errorf("%s should be skipped", id)
		}
	}
	if len(got) != 5 {
		t.Fatalf("expected 5 listings, got %d: %v", len(got), got)
	}

	bb := got["1g5yq7k"]
	if bb.Title != "Tudor Black Bay 58 79030N Blue, full set 2022 - $3,100 shipped CONUS" || bb.BrandSlug != "tudor" || bb.ReferenceNumber != "79030N" ||
		bb.Price == nil || *bb.Price != 3100 || bb.Currency != "USD" || bb.LocationCountry != "US" || bb.SellerName != "u/bb58_seller" ||
		bb.URL != "https://www.reddit.com/r/Watchexchange/comments/1g5yq7k/wts_tudor_black_bay_58_79030n_blue_full_set_2022/" ||
		bb.HasBox == nil || !*bb.HasBox || bb.HasPapers == nil || !*bb.HasPapers || bb.Year == nil || *bb.Year != 2022 || bb.Condition != model.ConditionGood {
		t.Errorf("BB58 parsed wrong: %+v", bb)
	}
	if len(bb.ImageURLs) != 2 || bb.ImageURLs[0] != "https://preview.redd.it/a1b2c3.jpg?width=4032&format=pjpg&auto=webp&s=1" {
		t.Errorf("BB58 gallery order: %v", bb.ImageURLs)
	}

	sp := got["1g5xr2m"]
	if sp.BrandSlug != "omega" || sp.ReferenceNumber != "310.30.42.50.01.001" || sp.Price == nil || *sp.Price != 7400 || sp.Currency != "CAD" || sp.LocationCountry != "" {
		t.Errorf("Speedmaster parsed wrong: %+v", sp)
	}
	if len(sp.ImageURLs) != 1 || sp.ImageURLs[0] != "https://preview.redd.it/x9y8z7w6.jpeg?auto=webp&s=abc" {
		t.Errorf("Speedmaster preview: %v", sp.ImageURLs)
	}

	// price only in the body: the asking line wins over "Retail was $6,300"
	gs := got["1g5t9dd"]
	if gs.BrandSlug != "grand-seiko" || gs.Price == nil || *gs.Price != 4200 || len(gs.ImageURLs) != 0 || string(gs.Attributes) == "" {
		t.Errorf("Grand Seiko parsed wrong: %+v attrs=%s", gs, gs.Attributes)
	}

	// no flair, "[WTS]" in the title; unworn
	prx := got["1g5s4ee"]
	if prx.Price == nil || *prx.Price != 495 || prx.Condition != model.ConditionUnworn || prx.ReferenceNumber != "T137.407.11.041.00" {
		t.Errorf("PRX parsed wrong: %+v", prx)
	}

	no := got["1g5x0h9"]
	if no.Price == nil || *no.Price != 1650 || no.Currency != "EUR" || no.ReferenceNumber != "" {
		t.Errorf("Nomos parsed wrong: price=%v cur=%s ref=%q", no.Price, no.Currency, no.ReferenceNumber)
	}
}

func TestPrice(t *testing.T) {
	cases := []struct {
		in   string
		want float64
		cur  string
		ok   bool
	}{
		{"[WTS] Rolex 16610 - $8,500 shipped", 8500, "USD", true},
		{"Seiko SKX007 $250", 250, "USD", true},
		{"Tudor Pelagos 25600TN $3.9k", 3900, "USD", true},
		{"IWC Mark XVIII 3270 US$ 2,950", 2950, "USD", true},
		{"Sinn 556 950 USD", 950, "USD", true},
		{"Oris Aquis £1,100", 1100, "GBP", true},
		{"Longines Legend Diver €1.450", 1450, "EUR", true},
		{"Hamilton Khaki A$700", 700, "AUD", true},
		{"Rolex 16610 2005 full set", 0, "", false}, // bare numbers are references and years
		{"$5 off shipping", 0, "", false},
	}
	for _, c := range cases {
		v, cur, ok := Price(c.in)
		if ok != c.ok || v != c.want || cur != c.cur {
			t.Errorf("Price(%q) = %v %s %v, want %v %s %v", c.in, v, cur, ok, c.want, c.cur, c.ok)
		}
	}
}
