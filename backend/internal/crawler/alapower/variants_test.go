package alapower

import (
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

// Hand-written pages in the shape a page reader showed for the 2026-10 variants (the sites are
// unreachable from the dev sandbox, so no captured fixture). They pin the class-free card lookup.

func card(href, img, title, extra string) string {
	return `<td valign="top" width="230"><table>
<tr><td><a href="` + href + `"><img src="` + img + `" border="0"></a></td></tr>
<tr><td><a href="` + href + `"><span style="font-size:12px;">` + title + `</span></a></td></tr>
` + extra + `</table></td>`
}

func page(cards ...string) string {
	h := `<html><body><table><tr><td><a href="product.asp?cat=47">Rolex 勞力士</a></td></tr></table><table><tr>`
	for _, c := range cards {
		h += c
	}
	return h + `</tr></table><a href="product.asp?index=2">2</a></body></html>`
}

func TestParseListGoodtimezoneVariant(t *testing.T) {
	html := page(
		card("https://www.goodtimezone.com.tw/product_open.asp?id=11845", "https://www.goodtimezone.com.tw/product/product_BIG/6857-107.JPG",
			"ROLEX 勞力士 16013 Datejust 原裝布紋面 壓克力鏡面 36mm",
			`<tr><td>編號 :LRB968</td></tr><tr><td>售價 : 155000 元</td></tr>`),
		card("https://www.goodtimezone.com.tw/product_open.asp?id=11834", "https://www.goodtimezone.com.tw/product/product_BIG/10204-3510.PNG",
			"ROLEX 勞力士 126500 LN Daytona 全新26/10現貨 白面 白熊貓 最新款木質保卡 40mm",
			`<tr><td>編號 :LRB965</td></tr><tr><td>售價 : 1195000 元</td></tr>`),
	)
	doc, err := crawler.ParseHTML([]byte(html), "https://www.goodtimezone.com.tw/product.asp?cat=47")
	if err != nil {
		t.Fatal(err)
	}
	items := Site{SellerName: "好時計"}.ParseList(doc, "Rolex 勞力士")
	if len(items) != 2 {
		t.Fatalf("want 2 items, got %d", len(items))
	}
	l := items[0]
	if l.ExternalID != "11845" || l.Price == nil || *l.Price != 155000 || l.BrandSlug != "rolex" {
		t.Errorf("first item: %+v", l)
	}
	if len(l.ImageURLs) != 1 || l.ImageURLs[0] != "https://www.goodtimezone.com.tw/product/product_BIG/6857-107.JPG" {
		t.Errorf("image: %v", l.ImageURLs)
	}
	if items[1].Condition != model.ConditionUnworn {
		t.Errorf("全新 should be unworn, got %s", items[1].Condition)
	}
	if !hasNextPage(doc, 1) {
		t.Error("index= pager not detected")
	}
}

func TestParseList369Variant(t *testing.T) {
	// spacer gif in the img, real photo as the cell background; 錶況 / 錶徑 lines
	c := `<td valign="top" width="230"><table>
<tr><td background="product/product_big/4271-1.JPG"><a href="product_open.asp?ID=4271"><img src="images/spacer.gif"></a></td></tr>
<tr><td><a href="product_open.asp?ID=4271">ROLEX 勞力士 DATEJUST 126300 126300-0014 綠色羅馬面 2026保單</a></td></tr>
<tr><td>編號 : TG899</td></tr><tr><td>錶況 : 未使用品</td></tr><tr><td>錶徑 : 41mm</td></tr><tr><td>售價 : 368,000元</td></tr>
</table></td>`
	doc, _ := crawler.ParseHTML([]byte(page(c)), "https://www.369rolexwatch.com/product.asp?cat=47")
	items := Site{SellerName: "金宸名錶"}.ParseList(doc, "Rolex")
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	l := items[0]
	if l.Price == nil || *l.Price != 368000 {
		t.Errorf("price: %v", l.Price)
	}
	if l.Condition != model.ConditionUnworn {
		t.Errorf("condition: %s", l.Condition)
	}
	if l.CaseDiameterMM == nil || *l.CaseDiameterMM != 41 {
		t.Errorf("diameter: %v", l.CaseDiameterMM)
	}
	if len(l.ImageURLs) != 1 || l.ImageURLs[0] != "https://www.369rolexwatch.com/product/product_big/4271-1.JPG" {
		t.Errorf("image: %v", l.ImageURLs)
	}
}

func TestParseListEgpsVariant(t *testing.T) {
	html := page(
		card("products_open.asp?ID=199139", "product/product_big/DE1CA8B5.JPG",
			"【營業時間調整公告】 2026年10月31日永生五權店、永久文心店 因內部盤點", `<tr><td>售價 : 歡迎來電洽詢</td></tr>`),
		card("products_open.asp?ID=199579", "product/product_big/46215-461.JPG",
			"PANERAI 沛納海 Luminor 1950 Rattrapante 8 Days Titanio PAM00530 2026年保單 手上鍊 47mm",
			`<tr><td>錶況 : 9成8新 附原廠盒單</td></tr><tr><td>售價 : 298,000元</td></tr>`),
	)
	doc, _ := crawler.ParseHTML([]byte(html), "https://www.egps.com.tw/products.asp?cat=2&type=open")
	items := Site{SellerName: "永生／永久名錶", SkipTitles: []string{"公告"}}.ParseList(doc, "品牌男錶")
	if len(items) != 1 || items[0].ExternalID != "199579" || items[0].BrandSlug != "panerai" || items[0].Price == nil {
		t.Fatalf("want only the Panerai, got %+v", items)
	}
}

func TestCategoryURL(t *testing.T) {
	s := Site{BaseURL: "https://www.egps.com.tw/", ListPath: "products.asp", CategoryQuery: "&type=open"}
	if got, want := s.categoryURL("2", 3), "https://www.egps.com.tw/products.asp?cat=2&type=open&page=3"; got != want {
		t.Errorf("got %s want %s", got, want)
	}
	s = Site{BaseURL: "https://www.369rolexwatch.com/", PageParams: []string{"page", "index"}}
	if got, want := s.categoryURL("47", 2), "https://www.369rolexwatch.com/product.asp?cat=47&page=2&index=2"; got != want {
		t.Errorf("got %s want %s", got, want)
	}
}

func TestDiscoverCategoriesSkipsJewellery(t *testing.T) {
	html := `<html><body>
<a href="product.asp?cat=145">Rolex‧勞力士</a><a href="product.asp?cat=165">勞力士對錶</a>
<a href="product.asp?cat=146">手錶配件(太陽圈、面盤)</a><a href="product.asp?cat=48">GIA鑽石 1克拉以上</a>
<a href="product.asp?cat=88">玉石 手鐲 項鍊 手鍊 戒指</a><a href="product.asp?cat=149">南洋珍珠項鍊 戒指 墜子</a>
<a href="product.asp?cat=151">BREGUET 寶璣</a><a href="product.asp?cat=147">其他</a></body></html>`
	doc, _ := crawler.ParseHTML([]byte(html), "https://www.dzwatch.tw/")
	s := Site{SkipCategories: []string{"配件", "鑽", "寶石", "玉石", "珍珠", "珊瑚", "飾品", "其他"}}
	var ids []string
	for _, c := range s.DiscoverCategories(doc, nil) {
		ids = append(ids, c.ID)
	}
	if len(ids) != 3 || ids[0] != "145" || ids[1] != "165" || ids[2] != "151" {
		t.Errorf("categories: %v", ids)
	}
}
