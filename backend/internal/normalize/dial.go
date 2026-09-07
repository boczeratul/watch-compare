package normalize

import (
	"regexp"
	"sort"
	"strings"
)

// DialColors is the canonical dial colour set with aliases in English, Traditional Chinese and
// Japanese. Chinese aliases must be followed by 面 / 面盤 / 錶面 (so "黑" alone never matches),
// English aliases by "dial"; Japanese katakana colours may stand alone because dealers list them
// bare in titles ("... 16613 ブルー メンズ").
var DialColors = []struct {
	Key string
	EN  []string // matched as "<alias> dial" / "dial: <alias>"
	ZH  []string // matched as "<alias>面", "<alias>色面盤", "<alias>錶面"
	JA  []string // matched bare or followed by 文字盤 / ダイヤル
}{
	{"black", []string{"black"}, []string{"黑", "黑色"}, []string{"ブラック", "黒"}},
	{"white", []string{"white"}, []string{"白", "白色"}, []string{"ホワイト", "白"}},
	{"silver", []string{"silver"}, []string{"銀", "銀色"}, []string{"シルバー", "銀"}},
	{"blue", []string{"blue", "navy"}, []string{"藍", "藍色", "深藍", "海軍藍", "牛仔藍"}, []string{"ブルー", "ネイビー", "青"}},
	{"green", []string{"green", "olive"}, []string{"綠", "綠色", "墨綠", "橄欖綠"}, []string{"グリーン", "オリーブ", "緑"}},
	{"grey", []string{"grey", "gray", "slate", "anthracite", "rhodium"}, []string{"灰", "灰色", "深灰", "板岩灰", "銠"}, []string{"グレー", "スレート", "アンスラサイト", "ロジウム", "灰"}},
	{"champagne", []string{"champagne"}, []string{"香檳", "香檳色", "香檳金"}, []string{"シャンパン", "シャンパーニュ"}},
	{"gold", []string{"gold", "golden"}, []string{"金", "金色"}, []string{"ゴールド", "金"}},
	{"brown", []string{"brown", "chocolate", "tobacco"}, []string{"咖啡", "咖啡色", "棕", "棕色", "褐", "巧克力"}, []string{"ブラウン", "チョコレート", "茶"}},
	{"red", []string{"red", "burgundy", "bordeaux"}, []string{"紅", "紅色", "酒紅"}, []string{"レッド", "バーガンディ", "ボルドー", "赤"}},
	{"pink", []string{"pink", "rose"}, []string{"粉", "粉色", "粉紅", "玫瑰"}, []string{"ピンク", "ローズ"}},
	{"salmon", []string{"salmon", "copper"}, []string{"鮭魚", "鮭紅", "銅"}, []string{"サーモン", "カッパー"}},
	{"purple", []string{"purple", "violet", "lavender", "aubergine"}, []string{"紫", "紫色", "薰衣草"}, []string{"パープル", "バイオレット", "ラベンダー", "紫"}},
	{"yellow", []string{"yellow", "lemon"}, []string{"黃", "黃色", "檸檬"}, []string{"イエロー", "レモン", "黄"}},
	{"orange", []string{"orange", "coral"}, []string{"橘", "橘色", "橙", "珊瑚"}, []string{"オレンジ", "コーラル"}},
	{"mother_of_pearl", []string{"mother of pearl", "mother-of-pearl", "mop", "shell"}, []string{"珍珠母貝", "珍珠貝", "貝殼", "珍珠"}, []string{"ホワイトシェル", "ブラックシェル", "ピンクシェル", "シェル", "マザーオブパール"}},
	{"meteorite", []string{"meteorite"}, []string{"隕石"}, []string{"メテオライト", "隕石"}},
	{"cream", []string{"cream", "ivory", "eggshell", "beige"}, []string{"象牙", "米色", "奶油", "米白"}, []string{"アイボリー", "クリーム", "ベージュ"}},
	{"bronze", []string{"bronze"}, []string{"青銅", "古銅"}, []string{"ブロンズ"}},
}

type dialMatcher struct {
	re  *regexp.Regexp
	key string
	pri int // lower = tried first
}

var dialMatchers []dialMatcher

func init() {
	for _, c := range DialColors {
		for _, a := range c.EN {
			a = regexp.QuoteMeta(a)
			dialMatchers = append(dialMatchers,
				dialMatcher{regexp.MustCompile(`(?i)\b` + a + `\s+(?:colou?r(?:ed)?\s+)?dial\b`), c.Key, 0},
				dialMatcher{regexp.MustCompile(`(?i)\bdial\s*[:：]?\s*` + a + `\b`), c.Key, 0},
			)
		}
		for _, a := range c.ZH {
			a = regexp.QuoteMeta(a)
			dialMatchers = append(dialMatchers, dialMatcher{regexp.MustCompile(a + `(?:色)?(?:面盤|面板|錶面|表面|面)`), c.Key, 1})
		}
		for _, a := range c.JA {
			q := regexp.QuoteMeta(a)
			// kanji colours only with 文字盤 to avoid matching 黒 in unrelated words; katakana may stand
			// alone — except ゴールド, which almost always names the case metal.
			if isKatakana(a) && a != "ゴールド" {
				dialMatchers = append(dialMatchers, dialMatcher{regexp.MustCompile(q + `(?:文字盤|ダイヤル|ダイアル)?`), c.Key, 2})
			} else {
				dialMatchers = append(dialMatchers, dialMatcher{regexp.MustCompile(q + `(?:色)?(?:文字盤|ダイヤル|ダイアル)`), c.Key, 1})
			}
		}
	}
	// Longer aliases first within the same priority so "ホワイトシェル" beats "ホワイト".
	sort.SliceStable(dialMatchers, func(i, j int) bool {
		if dialMatchers[i].pri != dialMatchers[j].pri {
			return dialMatchers[i].pri < dialMatchers[j].pri
		}
		return len(dialMatchers[i].re.String()) > len(dialMatchers[j].re.String())
	})
}

func isKatakana(s string) bool {
	for _, r := range s {
		if r < 0x30A0 || r > 0x30FF {
			return false
		}
	}
	return s != ""
}

// DialColor returns the canonical dial colour found in the texts, or "".
func DialColor(texts ...string) string {
	for _, t := range texts {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		best, bestPos := "", -1
		for _, m := range dialMatchers {
			loc := firstDialMatch(m.re, t)
			if loc == nil {
				continue
			}
			// earliest, highest-priority mention wins (titles lead with the dial colour)
			if best == "" || loc[0] < bestPos {
				best, bestPos = m.key, loc[0]
			}
		}
		if best != "" {
			return best
		}
	}
	return ""
}

// caseMaterialSuffixes follow a colour word when it describes the case, not the dial
// ("イエローゴールド", "ピンクゴールド", "rose gold"). Such matches are skipped.
var caseMaterialSuffixes = []string{"ゴールド", "ゴ-ルド", "金", "gold", "ステンレス", "スチール", "steel", "ブレス", "ストラップ", "ベゼル", "革", "レザー"}

func firstDialMatch(re *regexp.Regexp, t string) []int {
	offset := 0
	for offset < len(t) {
		loc := re.FindStringIndex(t[offset:])
		if loc == nil {
			return nil
		}
		start, end := offset+loc[0], offset+loc[1]
		rest := strings.TrimLeft(t[end:], " 　・-")
		skip := false
		for _, suf := range caseMaterialSuffixes {
			if strings.HasPrefix(strings.ToLower(rest), suf) {
				skip = true
				break
			}
		}
		if !skip {
			return []int{start, end}
		}
		offset = end
	}
	return nil
}

// DialColorKeys lists the canonical keys (for validation and the frontend).
func DialColorKeys() []string {
	out := make([]string, len(DialColors))
	for i, c := range DialColors {
		out[i] = c.Key
	}
	return out
}
