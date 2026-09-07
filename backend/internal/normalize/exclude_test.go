package normalize

import "testing"

func TestExclusionReason(t *testing.T) {
	cases := map[string]string{
		"Vagenari 勞力士專用 橡膠錶帶 黑色":               "brand:vagenari",
		"VAGENARI RUBBER STRAP FOR SUBMARINER": "brand:vagenari",
		"Rolex 116610LN 錶節 一節":                 "item:錶節",
		"Rolex Submariner 116610LN 黑水鬼":        "",
		"Omega Speedmaster 3570.50":            "",
		"":                                     "",
	}
	for in, want := range cases {
		if got := ExclusionReason(in); got != want {
			t.Errorf("ExclusionReason(%q) = %q, want %q", in, got, want)
		}
	}
	if got := ExclusionReason("Rolex 116610LN", "Vagenari"); got != "brand:vagenari" {
		t.Errorf("category name not checked: %q", got)
	}
}
