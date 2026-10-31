package crawler

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// LinkCard is one product card on a list page: the product id taken from its detail link and the
// largest element around that link that holds no link to another product.
type LinkCard struct {
	ID   string
	Card *goquery.Selection
}

// LinkCards locates product cards on table-soup list pages without relying on class names.
// idRe must have the product id as its first submatch and is matched against every a[href].
// For each id (in page order) the first link is walked up the tree while the parent element
// links to no other product id, so the card holds the title, price and image cells that sit
// next to the link. Cards come back in page order, each id once.
func LinkCards(doc *goquery.Document, idRe *regexp.Regexp) []LinkCard {
	var order []string
	first := map[string]*goquery.Selection{}
	doc.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		m := idRe.FindStringSubmatch(href)
		if m == nil {
			return
		}
		if _, ok := first[m[1]]; !ok {
			first[m[1]] = a
			order = append(order, m[1])
		}
	})
	out := make([]LinkCard, 0, len(order))
	for _, id := range order {
		node := first[id]
		for {
			parent := node.Parent()
			if parent.Length() == 0 || goquery.NodeName(parent) == "body" || goquery.NodeName(parent) == "html" {
				break
			}
			if linksOtherID(parent, idRe, id) {
				break
			}
			node = parent
		}
		out = append(out, LinkCard{ID: id, Card: node})
	}
	return out
}

func linksOtherID(s *goquery.Selection, idRe *regexp.Regexp, id string) bool {
	other := false
	s.Find("a[href]").EachWithBreak(func(_ int, a *goquery.Selection) bool {
		href, _ := a.Attr("href")
		if m := idRe.FindStringSubmatch(href); m != nil && m[1] != id {
			other = true
			return false
		}
		return true
	})
	return other
}

// LongestLinkText returns the longest text among the card's links matching idRe for id; on these
// sites the image link is empty and the title link carries the full title.
func LongestLinkText(card *goquery.Selection, idRe *regexp.Regexp, id string) string {
	best := ""
	card.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		if m := idRe.FindStringSubmatch(href); m == nil || m[1] != id {
			return
		}
		if t := Text(a); len([]rune(t)) > len([]rune(best)) {
			best = t
		}
	})
	return best
}

var bgURLRe = regexp.MustCompile(`url\(\s*['"]?([^'")]+)['"]?\s*\)`)

// CardImages returns the card's image URLs (absolute, in order, de-duplicated) whose path matches
// pathRe. It looks at img src / data-src / data-original, the legacy background attribute and
// inline background-image styles, so spacer-gif cards with a background photo still work.
func CardImages(doc *goquery.Document, card *goquery.Selection, pathRe *regexp.Regexp) []string {
	var out []string
	seen := map[string]bool{}
	add := func(u string) {
		u = strings.TrimSpace(u)
		if u == "" || !pathRe.MatchString(u) || strings.Contains(strings.ToLower(u), "spacer") {
			return
		}
		abs := AbsURL(doc, u)
		if !seen[abs] {
			seen[abs] = true
			out = append(out, abs)
		}
	}
	visit := func(s *goquery.Selection) {
		for _, attr := range []string{"src", "data-src", "data-original", "background"} {
			if v, ok := s.Attr(attr); ok {
				add(v)
			}
		}
		if st, ok := s.Attr("style"); ok {
			for _, m := range bgURLRe.FindAllStringSubmatch(st, -1) {
				add(m[1])
			}
		}
	}
	visit(card)
	card.Find("img, td, div, span, a, table").Each(func(_ int, s *goquery.Selection) { visit(s) })
	return out
}
