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

// DetectBoxPapers scans free text for box / papers mentions.
func DetectBoxPapers(s string) (box, papers *bool) {
	f := fold(s)
	t := true
	if strings.Contains(f, "box") || strings.Contains(f, "箱") || strings.Contains(f, "盒") || strings.Contains(f, "ボックス") {
		box = &t
	}
	if strings.Contains(f, "papers") || strings.Contains(f, "card") || strings.Contains(f, "warranty") || strings.Contains(f, "保証書") || strings.Contains(f, "保卡") || strings.Contains(f, "保單") || strings.Contains(f, "ギャランティ") || strings.Contains(f, "証書") {
		papers = &t
	}
	if strings.Contains(f, "full set") || strings.Contains(f, "fullset") || strings.Contains(f, "原廠盒單") || strings.Contains(f, "盒單") || strings.Contains(f, "箱保") {
		box, papers = &t, &t
	}
	return
}

// CleanText collapses whitespace and trims marketplace noise.
func CleanText(s string) string {
	s = strings.ReplaceAll(s, " ", " ")
	s = multiSpace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
