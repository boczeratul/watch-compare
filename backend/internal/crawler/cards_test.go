package crawler

import (
	"regexp"
	"testing"
)

func TestLinkCards(t *testing.T) {
	html := `<html><body><table><tr>
<td id="a"><div><a href="x.asp?id=1"><img src="p/1.jpg"></a></div><div><a href="x.asp?id=1">Title one long</a></div><p>$100</p></td>
<td id="b"><a href="x.asp?id=2"><img src="p/2.jpg"></a><a href="x.asp?id=2">Two</a><p>$200</p></td>
</tr></table></body></html>`
	doc, err := ParseHTML([]byte(html), "https://shop.tw/list.asp")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`x\.asp\?id=(\d+)`)
	cards := LinkCards(doc, re)
	if len(cards) != 2 || cards[0].ID != "1" || cards[1].ID != "2" {
		t.Fatalf("cards: %+v", cards)
	}
	if id, _ := cards[0].Card.Attr("id"); id != "a" {
		t.Errorf("card 1 should be td#a, got %q", id)
	}
	if got := LongestLinkText(cards[0].Card, re, "1"); got != "Title one long" {
		t.Errorf("title: %q", got)
	}
	if imgs := CardImages(doc, cards[1].Card, regexp.MustCompile(`p/`)); len(imgs) != 1 || imgs[0] != "https://shop.tw/p/2.jpg" {
		t.Errorf("images: %v", imgs)
	}
}
