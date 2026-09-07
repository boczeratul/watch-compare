package crawler

import (
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

func f(v float64) *float64 { return &v }

func TestApplyTaxFreePrice(t *testing.T) {
	// A Japanese source lists 税込; the tax-free price is derived with the dealer's own formula.
	jp := model.Listing{Price: f(1700000)}
	ApplyTaxFreePrice(&jp, "JP")
	if jp.PriceExclTax == nil || *jp.PriceExclTax != 1545455 {
		t.Errorf("JP derivation = %v, want 1545455 (Jackroad publishes this figure)", jp.PriceExclTax)
	}
	// Non-Japanese sources are untouched: Taiwanese and US prices need no adjustment here.
	tw := model.Listing{Price: f(398000)}
	ApplyTaxFreePrice(&tw, "TW")
	if tw.PriceExclTax != nil {
		t.Errorf("TW listing got a tax-free price: %v", *tw.PriceExclTax)
	}
	// An adapter that parsed the published figure wins over the derivation.
	parsed := model.Listing{Price: f(1700000), PriceExclTax: f(1234567)}
	ApplyTaxFreePrice(&parsed, "JP")
	if *parsed.PriceExclTax != 1234567 {
		t.Errorf("published value was overwritten: %v", *parsed.PriceExclTax)
	}
	// No price at all stays no price.
	none := model.Listing{}
	ApplyTaxFreePrice(&none, "JP")
	if none.PriceExclTax != nil {
		t.Errorf("got %v for a listing without a price", *none.PriceExclTax)
	}
}

func TestComparisonPrice(t *testing.T) {
	cases := []struct {
		name string
		in   model.Listing
		want float64
		ok   bool
	}{
		{"tax-free wins", model.Listing{Price: f(1700000), PriceExclTax: f(1545455)}, 1545455, true},
		{"falls back to listed", model.Listing{Price: f(398000)}, 398000, true},
		{"no price", model.Listing{}, 0, false},
	}
	for _, c := range cases {
		got, ok := ComparisonPrice(&c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("%s: got %v,%v want %v,%v", c.name, got, ok, c.want, c.ok)
		}
	}
}
