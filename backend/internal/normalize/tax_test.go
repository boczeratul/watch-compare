package normalize

import "testing"

func TestPriceExcludingTax(t *testing.T) {
	// Every pair below was read off a Jackroad product page, which prints the tax-included price
	// and its own TAXFREE figure side by side.
	published := map[float64]float64{
		898000:  816364,
		498000:  452728,
		478000:  434546,
		568000:  516364,
		688000:  625455,
		798000:  725455,
		1700000: 1545455,
	}
	for incl, want := range published {
		if got := PriceExcludingTax(incl, JapanConsumptionTax); got != want {
			t.Errorf("PriceExcludingTax(%.0f) = %.0f, want %.0f (as published by the dealer)", incl, got, want)
		}
	}
	// degenerate inputs are returned unchanged rather than producing a bogus discount
	for _, in := range []float64{0, -1} {
		if got := PriceExcludingTax(in, JapanConsumptionTax); got != in {
			t.Errorf("PriceExcludingTax(%v) = %v, want unchanged", in, got)
		}
	}
	if got := PriceExcludingTax(1000, 0); got != 1000 {
		t.Errorf("zero rate should be a no-op, got %v", got)
	}
}
