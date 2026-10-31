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
		// Taiwanese dealers' titles for whole watches mention links, straps and boxes too
		"ROLEX 勞力士 Submariner 116613LB 自動上鍊 40MM 2017年保單 二手美品 少一節錶節":           "",
		"ROLEX勞力士 116508 Daytona 2018/03 錶節齊全 錶徑40mm 自動上鍊":                     "",
		"ROLEX 勞力士 Submariner Date 126613LB 藍水鬼 搭配Vagenari維瑞亞橡膠錶帶 半金款":         "",
		"勞力士 ROLEX 116710LN 附原廠錶盒 盒單全":                                         "",
		"Cartier 卡地亞 WSSA0096 Santos de Cartier 全新26/09現貨 黑面 大型款 快拆式錶帶 39.8mm": "",
		"BVLGARI 寶格麗 Rettangolo 系列 RT45S 不鏽鋼橡膠錶帶":                              "",
		// accessories (watchstore)
		"FM V41/V45 全新原廠錶帶":                          "item:原廠錶帶",
		"AP愛彼 15300全新副廠膠錶帶":                          "item:副廠膠錶帶",
		"勞力士ROLEX 原廠錶盒 全新 小盒":                        "item:錶盒",
		"高質感攜帶盒 荔枝紋皮革(藍 綠 灰) 絨布內裡":                   "item:攜帶盒",
		"勞力士 Rolex 手錶攜帶盒 手錶收納袋 手錶保護袋 手錶攜行袋 加厚款 原廠全新": "item:攜帶盒",
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
