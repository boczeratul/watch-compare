package xintai

import (
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

// Hand-written in the shape a page reader showed (2026-10); the site is unreachable from the dev
// sandbox, so there is no captured fixture.
const listHTML = `<html><body>
<ul class="menu"><li><a href="goods.php?Ty=1">腕錶類</a></li><li><a href="goods.php?Ty=2">珠寶類</a></li></ul>
<div class="row">
 <div class="col"><div class="item">
  <a href="goods_info.php?SerialNo=158&Ty=1"><img src="upload/products/20251016164743984971.jpg"></a>
  <h4><a href="goods_info.php?SerialNo=158&Ty=1">勞力士ROLEX 116713LN GMT-MASTER II 盒單全 台灣AD</a></h4>
  <p class="price">NTD 365,000</p></div></div>
 <div class="col"><div class="item">
  <a href="goods_info.php?SerialNo=155&Ty=1"><img src="upload/products/20250814123223798558.jpg"></a>
  <h4><a href="goods_info.php?SerialNo=155&Ty=1">伯爵 PIAGET 黃k金 滿天星全鑽錶 精鑲鑽圈 原價148萬(已售出)</a></h4>
  <p class="price">NTD 999,999</p></div></div>
 <div class="col"><div class="item">
  <a href="goods_info.php?SerialNo=153&Ty=1"><img src="upload/products/20250807191452251242.jpg"></a>
  <h4><a href="goods_info.php?SerialNo=153&Ty=1">勞力士 ROLEX 116503 daytona 迪通拿 半金 白熊貓 近全新無傷 特價中</a></h4>
  <p class="price">NTD 550,000</p></div></div>
</div>
<div class="pager"><a href="goods.php?Ty=1&currentPage=2">2</a></div>
</body></html>`

func TestParseList(t *testing.T) {
	doc, err := crawler.ParseHTML([]byte(listHTML), "https://www.89675555.com/goods.php?Ty=1")
	if err != nil {
		t.Fatal(err)
	}
	items := ParseList(doc)
	if len(items) != 2 {
		t.Fatalf("want 2 unsold items, got %d: %+v", len(items), items)
	}
	l := items[0]
	if l.ExternalID != "158" || l.Price == nil || *l.Price != 365000 || l.BrandSlug != "rolex" {
		t.Errorf("first: %+v", l)
	}
	if l.URL != "https://www.89675555.com/goods_info.php?SerialNo=158&Ty=1" {
		t.Errorf("url: %s", l.URL)
	}
	if len(l.ImageURLs) != 1 || l.ImageURLs[0] != "https://www.89675555.com/upload/products/20251016164743984971.jpg" {
		t.Errorf("images: %v", l.ImageURLs)
	}
	if l.Condition != model.ConditionGood {
		t.Errorf("pawn default condition: %s", l.Condition)
	}
	if items[1].Condition != model.ConditionVeryGood {
		t.Errorf("近全新 should be very good, got %s", items[1].Condition)
	}
}
