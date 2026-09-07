package normalize

import (
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

func TestDetectBrand(t *testing.T) {
	cases := map[string]string{
		"BREITLING 百年靈 Navitimer B01 Chronograph 43": "breitling",
		"ロレックス サブマリーナ デイト 16613 ブルー メンズ 時計":          "rolex",
		"Grand Seiko SBGA211 Snowflake":              "grand-seiko",
		"Seiko Prospex SPB143":                       "seiko",
		"PATEK PHILIPPE Nautilus 5980/60G-001":       "patek-philippe",
		"A.LANGE.&SOHNE 朗格 Lange 1":                  "a-lange-sohne",
		"Jaeger - Le coulter‧積家":                     "jaeger-lecoultre",
		"Omegaville watches":                         "",
		"TUDOR Black Bay 58 79030N":                  "tudor",
		"オーデマ ピゲ ロイヤルオーク 15500ST":                    "audemars-piguet",
	}
	for in, want := range cases {
		got := ""
		if b := DetectBrand(in); b != nil {
			got = b.Slug
		}
		if got != want {
			t.Errorf("DetectBrand(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParsePrice(t *testing.T) {
	cases := map[string]float64{
		"NT$ 5,780,000元": 5780000,
		"￥1,700,000（税込）": 1700000,
		"€ 12.345,00":    12345,
		"$1,234.56":      1234.56,
		"US $9,999.00":   9999,
		"12.345 €":       12345,
		"398,000":        398000,
	}
	for in, want := range cases {
		got, ok := ParsePrice(in)
		if !ok || got != want {
			t.Errorf("ParsePrice(%q) = %v,%v want %v", in, got, ok, want)
		}
	}
	if _, ok := ParsePrice("SOLD OUT"); ok {
		t.Error("expected no price for SOLD OUT")
	}
}

func TestExtractReference(t *testing.T) {
	cases := map[string]string{
		"Rolex Submariner Date 116610LN 2019 full set": "116610LN",
		"PATEK PHILIPPE Nautilus 5980/60G-001 2026年保單": "5980/60G-001",
		"Omega Speedmaster 311.30.42.30.01.005":        "311.30.42.30.01.005",
		"BREITLING Navitimer RB0138211B1P1 2026年保卡":    "RB0138211B1P1",
		"IWC Portugieser IW371605 Chronograph":         "IW371605",
		"Tudor Black Bay 58 79030N":                    "79030N",
		"ロレックス デイトジャスト 69178 ブラック レディース":               "69178",
	}
	for in, want := range cases {
		if got := ExtractReference(in); got != want {
			t.Errorf("ExtractReference(%q) = %q want %q", in, got, want)
		}
	}
}

func TestCondition(t *testing.T) {
	cases := map[string]model.Condition{
		"中古":        model.ConditionGood,
		"新品":        model.ConditionNew,
		"未使用品":      model.ConditionUnworn,
		"Pre-owned": model.ConditionGood,
		"A":         model.ConditionVeryGood,
		"":          model.ConditionUnknown,
	}
	for in, want := range cases {
		if got := Condition(in); got != want {
			t.Errorf("Condition(%q) = %q want %q", in, got, want)
		}
	}
}

func TestParseDiameter(t *testing.T) {
	if d := ParseDiameter("ケース径：26 mm"); d == nil || *d != 26 {
		t.Errorf("got %v", d)
	}
	if d := ParseDiameter("自動上鍊 40.5mm"); d == nil || *d != 40.5 {
		t.Errorf("got %v", d)
	}
}

func TestDetectModel(t *testing.T) {
	cases := map[string]string{
		"ロレックス サブマリーナ デイト 16613 ブルー":       "Submariner",
		"勞力士 綠水鬼 126610LV":                 "Submariner",
		"Rolex GMT-Master II 126710BLNR":   "GMT-Master II",
		"オーデマ ピゲ ロイヤルオーク オフショア 26470ST":    "Royal Oak Offshore",
		"PATEK PHILIPPE Nautilus 5711/1A":  "Nautilus",
		"Grand Seiko SBGA211":              "",
		"BREITLING 百年靈 Navitimer B01 航空計時": "Navitimer",
	}
	for in, want := range cases {
		if got := DetectModel(in); got != want {
			t.Errorf("DetectModel(%q) = %q want %q", in, got, want)
		}
	}
}

func TestCanonicalizeQuery(t *testing.T) {
	cases := map[string]string{
		"勞力士 水鬼":           "Rolex Submariner",
		"サブマリーナ 116610":    "Submariner 116610",
		"rolex submariner": "rolex submariner",
		"オメガ スピードマスター":     "Omega Speedmaster",
		"116610LN":         "116610LN",
	}
	for in, want := range cases {
		if got := CanonicalizeQuery(in); got != want {
			t.Errorf("CanonicalizeQuery(%q) = %q want %q", in, got, want)
		}
	}
}

func TestDetectBoxPapers(t *testing.T) {
	b := func(v bool) *bool { return &v }
	cases := []struct {
		in          string
		box, papers *bool
	}{
		{"ROLEX 126610LN 有盒單 2023", b(true), b(true)},
		{"原廠盒單 保固中", b(true), b(true)},
		{"無盒單 單錶", b(false), b(false)},
		{"有盒無單", b(true), b(false)},
		{"只有保卡 無盒", b(false), b(true)},
		{"箱・保証書あり", b(true), b(true)},
		{"箱なし 保証書あり", b(false), b(true)},
		{"Full set with box and papers", b(true), b(true)},
		{"Watch only, no box no papers", b(false), b(false)},
		{"Submariner 2019 excellent", nil, nil},
	}
	for _, c := range cases {
		gb, gp := DetectBoxPapers(c.in)
		if !eqb(gb, c.box) || !eqb(gp, c.papers) {
			t.Errorf("DetectBoxPapers(%q) = %v,%v want %v,%v", c.in, fb(gb), fb(gp), fb(c.box), fb(c.papers))
		}
	}
}

func eqb(a, b *bool) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func fb(v *bool) string {
	if v == nil {
		return "nil"
	}
	if *v {
		return "true"
	}
	return "false"
}

func TestDialColor(t *testing.T) {
	cases := map[string]string{
		"ROLEX 勞力士 Submariner 黑水鬼 116610LN 黑面 有盒單":         "black",
		"PATEK PHILIPPE Nautilus 5711/1A-010 藍色面盤":         "blue",
		"ロレックス サブマリーナ デイト 16613 ブルー メンズ 時計":                "blue",
		"ロレックス GMTマスターII Ref.16710 ステンレススチール ブラック文字盤 美中古":  "black",
		"オメガ シーマスター ホワイトシェル文字盤 レディース":                      "mother_of_pearl",
		"Omega Speedmaster Professional black dial 2021":   "black",
		"Rolex Datejust 126334 Dial: Slate":                "grey",
		"Rolex Day-Date 18k rose gold with champagne dial": "champagne",
		"Rolex Day-Date 18k rose gold":                     "",
		"Tudor Black Bay 58 79030N":                        "",
		"勞力士 綠水鬼 126610LV 綠面 2022":                         "green",
		"IWC 萬國 葡萄牙 銀面 IW371605":                           "silver",
	}
	for in, want := range cases {
		if got := DialColor(in); got != want {
			t.Errorf("DialColor(%q) = %q want %q", in, got, want)
		}
	}
}
