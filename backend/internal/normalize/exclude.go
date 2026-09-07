package normalize

import "strings"

// NonWatchBrands are makers that appear in dealers' brand menus but do not make watches; their
// items (straps, buckles, bracelets) are dropped by the crawler. Matched as whole words in titles,
// brand lines and category names.
var NonWatchBrands = []string{
	"vagenari", // rubber straps
}

// NonWatchTitleMarkers are substrings that identify an item as a part rather than a watch.
var NonWatchTitleMarkers = []string{
	"錶節", // bracelet links
}

// ExclusionReason reports why an item is not a watch, or "" when it is one. texts are the
// title, brand line and category name of a listing.
func ExclusionReason(texts ...string) string {
	for _, t := range texts {
		f := fold(t)
		if f == "" {
			continue
		}
		for _, b := range NonWatchBrands {
			if containsWord(f, b) {
				return "brand:" + b
			}
		}
		for _, m := range NonWatchTitleMarkers {
			if strings.Contains(f, m) {
				return "item:" + m
			}
		}
	}
	return ""
}
