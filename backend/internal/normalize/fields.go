package normalize

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

var (
	numberRe   = regexp.MustCompile(`[0-9][0-9,.]*`)
	diameterRe = regexp.MustCompile(`(?i)(\d{2}(?:[.,]\d)?)\s*(?:mm|ミリ|公釐|毫米)`)
	yearRe     = regexp.MustCompile(`\b((?:19|20)\d{2})\b`)
	// Reference numbers: mixed alnum tokens like 116610LN, 5711/1A-010, 311.30.42.30.01.005, IW371605, RB0138211B1P1
	refRe = regexp.MustCompile(`\b([A-Z]{0,3}\d{3,8}(?:(?:[A-Z]{1,4}\d{0,4}){1,3}|(?:[./-][A-Z0-9]{1,6}){1,6})?)\b`)
)

// ParsePrice extracts the first number in s, tolerating thousands separators and
// currency symbols. ok is false when no number is found.
func ParsePrice(s string) (float64, bool) {
	s = strings.ReplaceAll(s, "，", ",")
	m := numberRe.FindString(s)
	if m == "" {
		return 0, false
	}
	// Handle "1.234.567,89" (EU) vs "1,234,567.89" (US) vs "398,000" (TW/JP).
	lastComma := strings.LastIndex(m, ",")
	lastDot := strings.LastIndex(m, ".")
	switch {
	case lastComma > lastDot && len(m)-lastComma-1 == 2: // comma is decimal separator
		m = strings.ReplaceAll(m, ".", "")
		m = strings.ReplaceAll(m, ",", ".")
	default:
		m = strings.ReplaceAll(m, ",", "")
		// A dot followed by exactly 3 digits and no other dot is a thousands separator (EU style "1.234")
		if strings.Count(m, ".") > 1 || (lastDot >= 0 && len(m)-lastDot-1 == 3 && !strings.Contains(m, ",")) {
			m = strings.ReplaceAll(m, ".", "")
		}
	}
	v, err := strconv.ParseFloat(m, 64)
	if err != nil || v <= 0 {
		return 0, false
	}
	return v, true
}

// ParseDiameter extracts a case diameter in millimetres.
func ParseDiameter(s string) *float64 {
	m := diameterRe.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64)
	if err != nil || v < 15 || v > 70 {
		return nil
	}
	return &v
}

// ParseYear extracts a plausible production year.
func ParseYear(s string) *int {
	for _, m := range yearRe.FindAllStringSubmatch(s, -1) {
		y, _ := strconv.Atoi(m[1])
		if y >= 1900 && y <= 2100 {
			return &y
		}
	}
	return nil
}

// ExtractReference finds the most likely reference number in a title.
func ExtractReference(title string) string {
	t := strings.ToUpper(fold(title))
	best := ""
	for _, m := range refRe.FindAllStringSubmatch(t, -1) {
		c := m[1]
		if isPlainYear(c) || len(c) < 4 {
			continue
		}
		// prefer tokens containing both letters and digits, then longest
		if score(c) > score(best) {
			best = c
		}
	}
	return best
}

func isPlainYear(s string) bool {
	if len(s) != 4 {
		return false
	}
	y, err := strconv.Atoi(s)
	return err == nil && y >= 1900 && y <= 2100
}

func score(s string) int {
	if s == "" {
		return -1
	}
	hasLetter, hasDigit := false, false
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			hasLetter = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}
	sc := len(s)
	if hasLetter && hasDigit {
		sc += 10
	}
	if strings.ContainsAny(s, "./-") {
		sc += 5
	}
	return sc
}

// Condition maps a marketplace condition label onto the normalized scale.
func Condition(s string) model.Condition {
	f := fold(s)
	// Dealers pad CJK labels for alignment ("中　古"); compare CJK markers without spaces.
	if hasCJK(f) {
		f = strings.ReplaceAll(f, " ", "")
	}
	switch {
	case f == "":
		return model.ConditionUnknown
	case strings.Contains(f, "unworn") || strings.Contains(f, "未使用") || strings.Contains(f, "全新") || strings.Contains(f, "brand new") || strings.Contains(f, "new with tags") || strings.Contains(f, "nos"):
		return model.ConditionUnworn
	case strings.Contains(f, "new") || strings.Contains(f, "新品") || strings.Contains(f, "新品仕上げ") || f == "n" || f == "s":
		return model.ConditionNew
	case strings.Contains(f, "very good") || strings.Contains(f, "excellent") || strings.Contains(f, "美品") || strings.Contains(f, "極美") || f == "a" || strings.Contains(f, "ランクa") || strings.Contains(f, "sa"):
		return model.ConditionVeryGood
	case strings.Contains(f, "good") || strings.Contains(f, "中古") || strings.Contains(f, "二手") || strings.Contains(f, "pre-owned") || strings.Contains(f, "preowned") || strings.Contains(f, "used") || f == "b" || f == "ab":
		return model.ConditionGood
	case strings.Contains(f, "fair") || strings.Contains(f, "worn") || f == "c":
		return model.ConditionFair
	case strings.Contains(f, "poor") || strings.Contains(f, "parts") || strings.Contains(f, "repair") || f == "d" || strings.Contains(f, "ジャンク"):
		return model.ConditionPoor
	}
	return model.ConditionUnknown
}

func hasCJK(s string) bool {
	for _, r := range s {
		if r >= 0x2E80 && r <= 0x9FFF || r >= 0xAC00 && r <= 0xD7AF || r >= 0xFF00 && r <= 0xFFEF {
			return true
		}
	}
	return false
}

// MovementType maps a movement label to the normalized set.
func MovementType(s string) model.Movement {
	f := fold(s)
	switch {
	case strings.Contains(f, "automatic") || strings.Contains(f, "自動") || strings.Contains(f, "自动") || strings.Contains(f, "self-winding") || strings.Contains(f, "selfwinding") || strings.Contains(f, "automatik"):
		return model.MovementAutomatic
	case strings.Contains(f, "manual") || strings.Contains(f, "hand-wound") || strings.Contains(f, "hand wound") || strings.Contains(f, "handaufzug") || strings.Contains(f, "手動") || strings.Contains(f, "手巻") || strings.Contains(f, "手上鍊") || strings.Contains(f, "手上链"):
		return model.MovementManual
	case strings.Contains(f, "quartz") || strings.Contains(f, "クォーツ") || strings.Contains(f, "クオーツ") || strings.Contains(f, "石英") || strings.Contains(f, "solar") || strings.Contains(f, "ソーラー") || strings.Contains(f, "eco-drive") || strings.Contains(f, "電池"):
		return model.MovementQuartz
	}
	return model.MovementUnknown
}

// GenderType maps a gender label to the normalized set.
func GenderType(s string) model.Gender {
	f := fold(s)
	switch {
	case strings.Contains(f, "unisex") || strings.Contains(f, "ユニセックス") || strings.Contains(f, "中性"):
		return model.GenderUnisex
	case strings.Contains(f, "women") || strings.Contains(f, "ladies") || strings.Contains(f, "lady") || strings.Contains(f, "female") || strings.Contains(f, "レディース") || strings.Contains(f, "女錶") || strings.Contains(f, "女性") || strings.Contains(f, "女士"):
		return model.GenderWomen
	case strings.Contains(f, "men") || strings.Contains(f, "gents") || strings.Contains(f, "male") || strings.Contains(f, "メンズ") || strings.Contains(f, "男錶") || strings.Contains(f, "男性") || strings.Contains(f, "男士"):
		return model.GenderMen
	}
	return model.GenderUnknown
}

var (
	// "單錶" (watch only) must start a token: with CJK spaces squeezed out, "有盒單 錶況佳" would
	// otherwise read as "…盒單錶況佳" and look like a watch-only listing.
	watchOnlyTokenRe = regexp.MustCompile(`(^|[\s,，、;；:：/|()（）\[\]【】])(單錶|单表)`)
	// "一錶一紙" / "一表一證書" / "一錶一卡": the watch plus exactly one accessory.
	oneWatchOneRe = regexp.MustCompile(`一[錶表]一(盒|紙|纸|證書|证书|證|证|保卡|卡|單|单)`)
)

// DetectBoxPapers scans free text for box / papers mentions in English, Chinese (Traditional and
// Simplified) and Japanese. Explicit negatives ("無盒單", "淨錶", "no papers", "箱なし") set false;
// nil means the text says nothing either way. For a structured accessories field, where whatever
// is not listed is absent, use DetectAccessories instead.
func DetectBoxPapers(s string) (box, papers *bool) {
	raw := fold(s)
	f := raw
	if hasCJK(f) {
		f = strings.ReplaceAll(f, " ", "")
	}
	t, no := true, false
	set := func(dst **bool, v *bool) {
		if *dst == nil {
			*dst = v
		}
	}
	// 0. "the watch plus one thing"
	if m := oneWatchOneRe.FindStringSubmatch(f); m != nil {
		if m[1] == "盒" {
			set(&box, &t)
			set(&papers, &no)
		} else {
			set(&papers, &t)
			set(&box, &no)
		}
	}
	// 1. explicit negatives first — they are more specific than the positive markers they contain.
	// Watch-only words: 淨錶 / 凈錶 (HK dealers, both spellings occur), 裸錶 (TW pawn shops), 單錶,
	// 本体のみ / 付属品なし (JP).
	for _, m := range []string{
		"淨錶", "凈錶", "淨表", "凈表", "净表", "裸錶", "裸表", "本体のみ", "時計のみ",
		"付属品なし", "付属品無し", "付属品無", "付属なし", "付属無し",
		"無盒單", "沒盒單", "無盒無單", "無盒及單", "不含盒單", "无盒单", "没盒单", "无盒无单",
		"unbox&paper", "箱・保証書なし", "箱保証書なし", "箱保証書無し",
		"no box or papers", "no box and papers", "no box/papers", "no accessories", "without accessories",
		"watch only", "head only", "naked watch",
	} {
		if strings.Contains(f, m) {
			set(&box, &no)
			set(&papers, &no)
		}
	}
	if watchOnlyTokenRe.MatchString(raw) {
		set(&box, &no)
		set(&papers, &no)
	}
	for _, m := range []string{"無盒", "沒盒", "沒有盒", "不含盒", "缺盒", "無原盒", "無原廠盒", "無錶盒", "无盒", "没盒", "没有盒", "无表盒", "no box", "without box", "箱なし", "箱無し", "箱無", "箱のみ", "box only"} {
		if strings.Contains(f, m) {
			if m == "box only" || m == "箱のみ" {
				set(&box, &t)
				set(&papers, &no)
			} else {
				set(&box, &no)
			}
		}
	}
	for _, m := range []string{
		"無單", "沒單", "無卡", "無保卡", "無保單", "無保證", "無證書", "缺單", "缺卡", "缺保卡", "沒有保卡", "沒有單",
		"无单", "没单", "无卡", "无保卡", "无保单", "无证书", "缺单", "没有保卡",
		"no papers", "no card", "no warranty", "without papers",
		"保証書なし", "保証書無し", "保証書無", "ギャランティなし", "ギャラなし", "ギャラ無", "保証書のみ", "papers only",
	} {
		if strings.Contains(f, m) {
			if m == "papers only" || m == "保証書のみ" {
				set(&papers, &t)
				set(&box, &no)
			} else {
				set(&papers, &no)
			}
		}
	}
	// 2. both-in-one markers
	for _, m := range []string{
		"full set", "fullset", "全套", "原廠盒單", "有盒單", "附盒單", "盒單齊", "盒單全", "盒單", "盒单", "盒卡", "盒證", "盒証", "盒证",
		"箱保", "箱・保証書あり", "箱保証書あり", "箱、保証書", "箱・保証書",
		"box and papers", "box & papers", "box/papers", "box, papers",
	} {
		if strings.Contains(f, m) {
			set(&box, &t)
			set(&papers, &t)
		}
	}
	// 3. single markers
	for _, m := range []string{"有盒", "附盒", "原廠盒", "原厂盒", "原盒", "錶盒", "表盒", "外盒", "內盒", "内盒", "木盒", "盒子", "box", "箱あり", "ボックス", "内箱", "外箱", "箱"} {
		if strings.Contains(f, m) {
			set(&box, &t)
		}
	}
	for _, m := range []string{
		"有單", "附單", "有单", "附单", "跟單", "ad單", "ad单", "保卡", "新卡", "舊卡", "旧卡", "跟卡", "保單", "保单",
		"保證卡", "保证卡", "保固卡", "保證書", "保证书", "保固書", "證書", "证书", "保証書", "証書",
		"papers", "warranty card", "guarantee card", "card", "ギャランティ",
	} {
		if strings.Contains(f, m) {
			set(&papers, &t)
		}
	}
	return
}

// DetectAccessories reads a structured accessories field ("配件: 淨錶", "付属: 箱", "附件: 原盒1
// 保單1"): the field lists everything that comes with the watch, so whatever it does not mention
// is absent rather than unknown. An empty field stays unknown.
func DetectAccessories(s string) (box, papers *bool) {
	f := fold(s)
	if f == "" {
		return nil, nil
	}
	box, papers = DetectBoxPapers(s)
	t, no := true, false
	anyOf := func(markers ...string) *bool {
		for _, m := range markers {
			if strings.Contains(f, m) {
				return &t
			}
		}
		return &no
	}
	if box == nil {
		box = anyOf("盒", "箱")
	}
	if papers == nil {
		papers = anyOf("卡", "證", "証", "证", "保單", "保单")
	}
	return
}

// CleanText collapses whitespace and trims marketplace noise.
func CleanText(s string) string {
	s = strings.ReplaceAll(s, " ", " ")
	s = multiSpace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
