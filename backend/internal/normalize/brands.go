// Package normalize turns messy, multilingual marketplace text into canonical fields.
package normalize

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// BrandEntry maps a canonical brand to every alias we have seen in the wild
// (English, Traditional Chinese, Japanese, common misspellings).
type BrandEntry struct {
	Slug    string
	Name    string
	Aliases []string
}

// Brands is the canonical brand dictionary. Order matters only for ties; the
// matcher prefers the longest alias found in the input.
var Brands = []BrandEntry{
	{"rolex", "Rolex", []string{"rolex", "勞力士", "劳力士", "ロレックス"}},
	{"tudor", "Tudor", []string{"tudor", "帝舵", "チューダー", "チュードル"}},
	{"omega", "Omega", []string{"omega", "歐米茄", "欧米茄", "オメガ"}},
	{"patek-philippe", "Patek Philippe", []string{"patek philippe", "patek", "百達翡麗", "百达翡丽", "パテック フィリップ", "パテックフィリップ", "パテック・フィリップ"}},
	{"audemars-piguet", "Audemars Piguet", []string{"audemars piguet", "audemars", "愛彼", "爱彼", "オーデマ ピゲ", "オーデマピゲ", "オーデマ・ピゲ"}},
	{"vacheron-constantin", "Vacheron Constantin", []string{"vacheron constantin", "vacheron", "江詩丹頓", "江诗丹顿", "ヴァシュロン コンスタンタン", "ヴァシュロン・コンスタンタン", "ヴァシュロンコンスタンタン"}},
	{"a-lange-sohne", "A. Lange & Söhne", []string{"a. lange & söhne", "a.lange & sohne", "a.lange&sohne", "a. lange & sohne", "lange & sohne", "lange & söhne", "a.lange.&sohne", "朗格", "a.ランゲ&ゾーネ", "ランゲ&ゾーネ", "ランゲ＆ゾーネ"}},
	{"cartier", "Cartier", []string{"cartier", "卡地亞", "卡地亚", "カルティエ"}},
	{"breitling", "Breitling", []string{"breitling", "百年靈", "百年灵", "ブライトリング"}},
	{"iwc", "IWC", []string{"iwc", "iwc schaffhausen", "萬國", "万国", "インターナショナル・ウォッチ・カンパニー"}},
	{"jaeger-lecoultre", "Jaeger-LeCoultre", []string{"jaeger-lecoultre", "jaeger lecoultre", "jaeger - le coulter", "jaeger le coultre", "jlc", "積家", "积家", "ジャガー・ルクルト", "ジャガールクルト", "ジャガー ルクルト"}},
	{"panerai", "Panerai", []string{"panerai", "officine panerai", "沛納海", "沛纳海", "パネライ"}},
	{"hublot", "Hublot", []string{"hublot", "宇舶", "ウブロ"}},
	{"tag-heuer", "TAG Heuer", []string{"tag heuer", "tagheuer", "泰格豪雅", "豪雅", "タグ・ホイヤー", "タグホイヤー", "タグ ホイヤー"}},
	{"zenith", "Zenith", []string{"zenith", "真力時", "先力時", "真力时", "ゼニス"}},
	{"breguet", "Breguet", []string{"breguet", "寶璣", "宝玑", "ブレゲ"}},
	{"blancpain", "Blancpain", []string{"blancpain", "寶鉑", "宝珀", "ブランパン"}},
	{"chopard", "Chopard", []string{"chopard", "蕭邦", "萧邦", "ショパール"}},
	{"girard-perregaux", "Girard-Perregaux", []string{"girard-perregaux", "girard perregaux", "芝柏", "ジラール・ペルゴ", "ジラールペルゴ"}},
	{"glashutte-original", "Glashütte Original", []string{"glashütte original", "glashutte original", "glashutte", "glashütte", "格拉蘇蒂", "格拉苏蒂", "グラスヒュッテ・オリジナル", "グラスヒュッテオリジナル"}},
	{"longines", "Longines", []string{"longines", "浪琴", "ロンジン"}},
	{"montblanc", "Montblanc", []string{"montblanc", "萬寶龍", "万宝龙", "モンブラン"}},
	{"oris", "Oris", []string{"oris", "豪利時", "豪利时", "オリス"}},
	{"parmigiani-fleurier", "Parmigiani Fleurier", []string{"parmigiani fleurier", "parmigiani", "帕瑪強尼", "帕玛强尼", "パルミジャーニ・フルリエ", "パルミジャーニ"}},
	{"piaget", "Piaget", []string{"piaget", "伯爵", "ピアジェ"}},
	{"ulysse-nardin", "Ulysse Nardin", []string{"ulysse nardin", "雅典", "ユリス・ナルダン", "ユリスナルダン"}},
	{"bulgari", "Bulgari", []string{"bulgari", "bvlgari", "寶格麗", "宝格丽", "ブルガリ"}},
	{"chanel", "Chanel", []string{"chanel", "香奈兒", "香奈儿", "シャネル"}},
	{"maurice-lacroix", "Maurice Lacroix", []string{"maurice lacroix", "艾美", "モーリス・ラクロア", "モーリスラクロア"}},
	{"h-moser-cie", "H. Moser & Cie.", []string{"h. moser & cie", "h. moser & cie.", "h.moser & cie", "moser", "亨利慕時", "亨利慕时", "h.モーザー", "モーザー"}},
	{"bell-ross", "Bell & Ross", []string{"bell & ross", "bell&ross", "柏萊士", "柏莱士", "ベル&ロス", "ベル＆ロス"}},
	{"jaquet-droz", "Jaquet Droz", []string{"jaquet droz", "雅克德羅", "雅克德罗", "ジャケ・ドロー", "ジャケドロー"}},
	{"richard-mille", "Richard Mille", []string{"richard mille", "理查德 米勒", "理查德米勒", "理查米爾", "リシャール・ミル", "リシャールミル"}},
	{"grand-seiko", "Grand Seiko", []string{"grand seiko", "grandseiko", "特級精工", "冠藍獅", "グランドセイコー", "グランド セイコー"}},
	{"seiko", "Seiko", []string{"seiko", "精工", "セイコー"}},
	{"franck-muller", "Franck Muller", []string{"franck muller", "法蘭穆勒", "法穆兰", "フランク ミュラー", "フランク・ミュラー", "フランクミュラー"}},
	{"jacob-co", "Jacob & Co.", []string{"jacob & co", "jacob &co", "jacob&co", "傑克豹", "ジェイコブ"}},
	{"roger-dubuis", "Roger Dubuis", []string{"roger dubuis", "羅杰杜彼", "罗杰杜彼", "ロジェ・デュブイ", "ロジェデュブイ"}},
	{"daniel-roth", "Daniel Roth", []string{"daniel roth", "丹尼爾·羅斯", "ダニエル・ロート"}},
	{"bovet", "Bovet", []string{"bovet", "播威", "ボヴェ"}},
	{"hautlence", "Hautlence", []string{"hautlence", "豪朗時", "オートランス"}},
	{"mido", "Mido", []string{"mido", "美度", "ミドー"}},
	{"tissot", "Tissot", []string{"tissot", "天梭", "ティソ"}},
	{"hamilton", "Hamilton", []string{"hamilton", "漢米爾頓", "汉米尔顿", "ハミルトン"}},
	{"rado", "Rado", []string{"rado", "雷達", "雷达", "ラドー"}},
	{"nomos", "NOMOS Glashütte", []string{"nomos", "nomos glashütte", "nomos glashutte", "ノモス"}},
	{"sinn", "Sinn", []string{"sinn", "ジン"}},
	{"citizen", "Citizen", []string{"citizen", "星辰", "シチズン"}},
	{"casio", "Casio", []string{"casio", "g-shock", "卡西歐", "卡西欧", "カシオ"}},
	{"hermes", "Hermès", []string{"hermès", "hermes", "愛馬仕", "爱马仕", "エルメス"}},
	{"louis-vuitton", "Louis Vuitton", []string{"louis vuitton", "路易威登", "ルイ・ヴィトン", "ルイヴィトン"}},
	{"van-cleef-arpels", "Van Cleef & Arpels", []string{"van cleef & arpels", "van cleef", "梵克雅寶", "梵克雅宝", "ヴァン クリーフ&アーペル", "ヴァンクリーフ&アーペル"}},
	{"harry-winston", "Harry Winston", []string{"harry winston", "海瑞溫斯頓", "海瑞温斯顿", "ハリー・ウィンストン", "ハリーウィンストン"}},
	{"fp-journe", "F.P. Journe", []string{"f.p. journe", "f.p.journe", "fp journe", "journe", "f.p.ジュルヌ", "ジュルヌ"}},
	{"mb-f", "MB&F", []string{"mb&f", "mb & f"}},
	{"urwerk", "Urwerk", []string{"urwerk", "ウルベルク"}},
	{"greubel-forsey", "Greubel Forsey", []string{"greubel forsey", "高珀富斯", "グルーベル フォルセイ"}},
	{"laurent-ferrier", "Laurent Ferrier", []string{"laurent ferrier", "ローラン・フェリエ"}},
	{"baume-mercier", "Baume & Mercier", []string{"baume & mercier", "baume et mercier", "名士", "ボーム&メルシエ", "ボーム＆メルシエ"}},
	{"frederique-constant", "Frederique Constant", []string{"frederique constant", "frédérique constant", "康斯登", "フレデリック・コンスタント"}},
	{"bremont", "Bremont", []string{"bremont", "ブレモン"}},
	{"eberhard", "Eberhard & Co.", []string{"eberhard", "eberhard & co", "エベラール"}},
	{"corum", "Corum", []string{"corum", "崑崙", "昆仑", "コルム"}},
	{"ebel", "Ebel", []string{"ebel", "玉寶", "エベル"}},
	{"gucci", "Gucci", []string{"gucci", "グッチ"}},
	{"dior", "Dior", []string{"dior", "christian dior", "ディオール"}},
	{"tiffany", "Tiffany & Co.", []string{"tiffany", "tiffany & co", "ティファニー"}},
	{"orient", "Orient", []string{"orient", "東方", "オリエント"}},
	{"credor", "Credor", []string{"credor", "クレドール"}},
	{"kurono", "Kurono Tokyo", []string{"kurono", "kurono tokyo", "クロノトウキョウ"}},
}

var (
	brandBySlug   = map[string]*BrandEntry{}
	aliasMatchers []aliasMatcher
)

type aliasMatcher struct {
	alias string
	entry *BrandEntry
	ascii bool // alias is ascii: require word boundary
}

func init() {
	for i := range Brands {
		b := &Brands[i]
		brandBySlug[b.Slug] = b
		for _, a := range b.Aliases {
			aliasMatchers = append(aliasMatchers, aliasMatcher{alias: fold(a), entry: b, ascii: isASCII(a)})
		}
	}
	// Longest alias first so "grand seiko" beats "seiko", "tudor" beats nothing, etc.
	sort.SliceStable(aliasMatchers, func(i, j int) bool {
		return len([]rune(aliasMatchers[i].alias)) > len([]rune(aliasMatchers[j].alias))
	})
}

// BrandBySlug returns the canonical brand for a slug.
func BrandBySlug(slug string) (*BrandEntry, bool) {
	b, ok := brandBySlug[slug]
	return b, ok
}

// DetectBrand finds the best-matching brand in free text (title, category name, etc).
// Returns nil when nothing matches.
func DetectBrand(texts ...string) *BrandEntry {
	for _, t := range texts {
		f := fold(t)
		if f == "" {
			continue
		}
		for _, m := range aliasMatchers {
			if m.ascii {
				if containsWord(f, m.alias) {
					return m.entry
				}
			} else if strings.Contains(f, m.alias) {
				return m.entry
			}
		}
	}
	return nil
}

var multiSpace = regexp.MustCompile(`\s+`)

// fold lowercases, NFKC-normalizes (full-width → half-width) and collapses whitespace.
func fold(s string) string {
	s = norm.NFKC.String(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "‧", " ")
	s = strings.ReplaceAll(s, "·", " ")
	s = strings.ReplaceAll(s, "・", " ")
	s = multiSpace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// containsWord reports whether needle occurs in hay delimited by non-alphanumerics.
func containsWord(hay, needle string) bool {
	idx := 0
	for {
		i := strings.Index(hay[idx:], needle)
		if i < 0 {
			return false
		}
		start := idx + i
		end := start + len(needle)
		before := start == 0 || !isAlnum(rune(hay[start-1]))
		after := end == len(hay) || !isAlnum(rune(hay[end]))
		if before && after {
			return true
		}
		idx = start + 1
		if idx >= len(hay) {
			return false
		}
	}
}

func isAlnum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}
