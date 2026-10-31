package normalize

import (
	"regexp"
	"strings"
)

// NonWatchBrands are makers that appear in dealers' brand menus but do not make watches; their
// items (straps, buckles, bracelets) are dropped by the crawler. Matched as whole words in titles,
// brand lines and category names.
var NonWatchBrands = []string{
	"vagenari", // rubber straps
}

// NonWatchTitleMarkers are substrings that identify an item as a part or accessory rather than a
// watch. Taiwanese dealers also use them when describing a whole watch ("錶節齊全", "附原廠錶盒"),
// so they only count when the title does not read as a watch (see wholeWatchMarkers) and not in a
// "comes with" phrase (see accompanyRe).
var NonWatchTitleMarkers = []string{
	"錶節",                    // bracelet links
	"原廠錶帶", "副廠錶帶", "副廠膠錶帶", // straps sold alone (a bare 錶帶 usually describes the watch's own)
	"錶盒",                              // boxes
	"攜帶盒", "收納盒", "收納袋", "保護袋", "攜行袋", // travel cases and pouches
}

// wholeWatchMarkers are words only a listing for a complete watch carries: movement, case size,
// dial and warranty-card details. A case diameter (28mm or more; straps are 16-24mm wide) counts
// too.
var wholeWatchMarkers = []string{
	"錶徑", "上鍊", "上鏈", "機芯", "保單", "保卡", "面盤", "錶盤", "石英", "防水", "腕錶",
}

// accompanyRe matches "comes with / paired with ..." phrases ("搭配vagenari維瑞亞橡膠錶帶",
// "附原廠錶盒", "贈錶帶"), which name extras of a watch rather than the item itself.
var accompanyRe = regexp.MustCompile(`(?:搭配|附|贈|送)[^\s,，、。/]{0,16}`)

// ExclusionReason reports why an item is not a watch, or "" when it is one. texts are the
// title, brand line and category name of a listing.
func ExclusionReason(texts ...string) string {
	for _, t := range texts {
		for _, m := range wholeWatchMarkers {
			if strings.Contains(fold(t), m) {
				return ""
			}
		}
		if d := ParseDiameter(t); d != nil && *d >= 28 {
			return ""
		}
	}
	for _, t := range texts {
		f := accompanyRe.ReplaceAllString(fold(t), " ")
		if strings.TrimSpace(f) == "" {
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
