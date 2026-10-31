package watchstore

import (
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

// Hand-written in the shape a page reader showed (2026-10); the site is unreachable from the dev
// sandbox, so there is no captured fixture.
const listHTML = `<html><body>
<a href="sublist.asp?larcode_e=&LarCode=ROLEX%20%E5%8B%9E%E5%8A%9B%E5%A3%AB">ROLEX 勞力士</a>
<table><tr>
<td><table>
 <tr><td><a href="prod_detail.asp?prodnum=1293"><img src="product/20269817461_S.jpg"></a></td></tr>
 <tr><td><a href="prod_detail.asp?prodnum=1293">ROLEX 勞力士 Daytona 迪通拿 126503 新款半金白迪 金屬外圈 最新機芯4131自動上鍊 錶徑40MM 2026年新式木紋保卡 未使用品</a></td></tr>
 <tr><td>價格：$845,000</td></tr>
 <tr><td><a href="order/add.asp?prodid=1293">詢問</a></td></tr>
</table></td>
<td><table>
 <tr><td><a href="prod_detail.asp?prodnum=1403"><img src="product/2024116163146_S.jpg"></a></td></tr>
 <tr><td><a href="prod_detail.asp?prodnum=1403">ROLEX 勞力士 Sea-Dweller 126603 海使者 金字海使 半金款 2023年保單 二手美品 排氦氣閥 潛水錶 自動上鍊 43mm</a></td></tr>
 <tr><td>價格：$523,000</td></tr>
</table></td>
</tr></table>
<a href="?keywords=&larcode=&newsclass=&page=2">下一頁</a>
</body></html>`

func TestParseList(t *testing.T) {
	doc, err := crawler.ParseHTML([]byte(listHTML), "https://www.watchstore.tw/")
	if err != nil {
		t.Fatal(err)
	}
	items := ParseList(doc)
	if len(items) != 2 {
		t.Fatalf("want 2 items, got %d", len(items))
	}
	l := items[0]
	if l.ExternalID != "1293" || l.Price == nil || *l.Price != 845000 || l.BrandSlug != "rolex" || l.Condition != model.ConditionUnworn {
		t.Errorf("first: %+v", l)
	}
	if len(l.ImageURLs) != 1 || l.ImageURLs[0] != "https://www.watchstore.tw/product/20269817461_S.jpg" {
		t.Errorf("images: %v", l.ImageURLs)
	}
	if items[1].Condition != model.ConditionVeryGood {
		t.Errorf("二手美品 should be very good, got %s", items[1].Condition)
	}
}
