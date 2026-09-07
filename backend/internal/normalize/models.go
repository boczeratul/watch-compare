package normalize

import (
	"sort"
	"strings"
)

// ModelEntry maps a canonical (English) model/collection name to aliases used by Japanese and
// Chinese dealers. Matching a model in a title lets "Submariner" find "サブマリーナ" and "水鬼".
type ModelEntry struct {
	Name    string
	Aliases []string
}

// Models is the model alias dictionary. Longest alias wins; canonical names are aliases too.
var Models = []ModelEntry{
	// Rolex
	{"Submariner", []string{"submariner", "サブマリーナ", "サブマリーナー", "水鬼", "潛航者", "潜航者"}},
	{"Datejust", []string{"datejust", "デイトジャスト", "日誌型", "蠔式日誌"}},
	{"Daytona", []string{"daytona", "cosmograph daytona", "デイトナ", "迪通拿", "地通拿"}},
	{"GMT-Master II", []string{"gmt-master ii", "gmt master ii", "gmtマスターii", "gmtマスター2", "gmt-master", "gmtマスター", "格林威治"}},
	{"Explorer II", []string{"explorer ii", "エクスプローラーii", "エクスプローラー2", "探險家二", "探險家2", "探险家二"}},
	{"Explorer", []string{"explorer", "エクスプローラー", "探險家", "探险家", "探一"}},
	{"Sea-Dweller", []string{"sea-dweller", "sea dweller", "シードゥエラー", "海使", "深海使"}},
	{"Deepsea", []string{"deepsea", "ディープシー", "深海"}},
	{"Yacht-Master", []string{"yacht-master", "yacht master", "ヨットマスター", "遊艇", "游艇"}},
	{"Oyster Perpetual", []string{"oyster perpetual", "オイスターパーペチュアル", "蠔式恆動"}},
	{"Day-Date", []string{"day-date", "day date", "デイデイト", "星期日曆", "星期日历"}},
	{"Milgauss", []string{"milgauss", "ミルガウス", "閃電"}},
	{"Air-King", []string{"air-king", "air king", "エアキング", "空中霸王"}},
	{"Sky-Dweller", []string{"sky-dweller", "スカイドゥエラー", "天行者"}},
	{"Cellini", []string{"cellini", "チェリーニ", "徹利尼"}},
	{"Lady-Datejust", []string{"lady-datejust", "lady datejust", "レディデイトジャスト", "女裝日誌"}},
	// Tudor
	{"Black Bay", []string{"black bay", "ブラックベイ", "碧灣", "碧湾", "啟承碧灣"}},
	{"Pelagos", []string{"pelagos", "ペラゴス", "領潛"}},
	{"Ranger", []string{"ranger", "レンジャー"}},
	// Omega
	{"Speedmaster", []string{"speedmaster", "スピードマスター", "超霸"}},
	{"Seamaster", []string{"seamaster", "シーマスター", "海馬", "海马"}},
	{"Constellation", []string{"constellation", "コンステレーション", "星座"}},
	{"De Ville", []string{"de ville", "デ・ヴィル", "デビル", "碟飛"}},
	{"Aqua Terra", []string{"aqua terra", "アクアテラ"}},
	{"Planet Ocean", []string{"planet ocean", "プラネットオーシャン", "海洋宇宙"}},
	// Patek Philippe
	{"Nautilus", []string{"nautilus", "ノーチラス", "鸚鵡螺", "鹦鹉螺", "金鷹"}},
	{"Aquanaut", []string{"aquanaut", "アクアノート", "手雷"}},
	{"Calatrava", []string{"calatrava", "カラトラバ"}},
	{"Grand Complications", []string{"grand complications", "グランドコンプリケーション"}},
	{"Complications", []string{"complications", "コンプリケーション"}},
	{"Twenty~4", []string{"twenty~4", "twenty-4", "twenty 4", "トゥエンティ〜4", "トゥエンティー4"}},
	// Audemars Piguet
	{"Royal Oak Offshore", []string{"royal oak offshore", "ロイヤルオークオフショア", "ロイヤルオーク オフショア", "皇家橡樹離岸"}},
	{"Royal Oak", []string{"royal oak", "ロイヤルオーク", "ロイヤル オーク", "皇家橡樹", "皇家橡树"}},
	{"Code 11.59", []string{"code 11.59", "コード11.59"}},
	// Vacheron Constantin
	{"Overseas", []string{"overseas", "オーヴァーシーズ", "オーバーシーズ", "縱橫四海"}},
	{"Patrimony", []string{"patrimony", "パトリモニー", "傳襲"}},
	{"Traditionnelle", []string{"traditionnelle", "トラディショナル"}},
	{"Historiques", []string{"historiques", "ヒストリーク"}},
	// A. Lange & Söhne
	{"Lange 1", []string{"lange 1", "lange1", "ランゲ1", "ランゲ・ワン"}},
	{"Saxonia", []string{"saxonia", "サクソニア"}},
	{"Datograph", []string{"datograph", "ダトグラフ"}},
	{"Zeitwerk", []string{"zeitwerk", "ツァイトヴェルク"}},
	{"Odysseus", []string{"odysseus", "オデュッセウス"}},
	{"1815", []string{"1815"}},
	// Cartier
	{"Santos", []string{"santos", "サントス", "山度士"}},
	{"Tank", []string{"tank", "タンク", "坦克"}},
	{"Ballon Bleu", []string{"ballon bleu", "バロンブルー", "藍氣球", "蓝气球"}},
	{"Pasha", []string{"pasha", "パシャ"}},
	{"Panthère", []string{"panthère", "panthere", "パンテール", "美洲豹"}},
	{"Calibre", []string{"calibre de cartier", "カリブル"}},
	// Breitling
	{"Navitimer", []string{"navitimer", "ナビタイマー", "航空計時"}},
	{"Superocean", []string{"superocean", "スーパーオーシャン", "超級海洋"}},
	{"Chronomat", []string{"chronomat", "クロノマット", "機械計時"}},
	{"Avenger", []string{"avenger", "アベンジャー", "復仇者"}},
	{"Premier", []string{"premier", "プレミエ"}},
	// IWC
	{"Portugieser", []string{"portugieser", "portuguese", "ポルトギーゼ", "葡萄牙"}},
	{"Pilot's Watch", []string{"pilot's watch", "pilots watch", "pilot watch", "パイロットウォッチ", "パイロット・ウォッチ", "飛行員"}},
	{"Portofino", []string{"portofino", "ポートフィノ", "柏濤菲諾"}},
	{"Ingenieur", []string{"ingenieur", "インヂュニア", "インジュニア", "工程師"}},
	{"Aquatimer", []string{"aquatimer", "アクアタイマー"}},
	// Jaeger-LeCoultre
	{"Reverso", []string{"reverso", "レベルソ", "翻轉"}},
	{"Master Control", []string{"master control", "マスターコントロール"}},
	{"Polaris", []string{"polaris", "ポラリス"}},
	// Panerai
	{"Luminor", []string{"luminor", "ルミノール", "廬米諾"}},
	{"Radiomir", []string{"radiomir", "ラジオミール"}},
	{"Submersible", []string{"submersible", "サブマーシブル"}},
	// Hublot
	{"Big Bang", []string{"big bang", "ビッグバン", "ビッグ・バン", "大爆炸"}},
	{"Classic Fusion", []string{"classic fusion", "クラシックフュージョン", "クラシック・フュージョン", "經典融合"}},
	// TAG Heuer
	{"Carrera", []string{"carrera", "カレラ", "卡萊拉"}},
	{"Monaco", []string{"monaco", "モナコ", "摩納哥"}},
	{"Aquaracer", []string{"aquaracer", "アクアレーサー", "競潛"}},
	{"Autavia", []string{"autavia", "オータヴィア"}},
	// Zenith
	{"El Primero", []string{"el primero", "エル・プリメロ", "エルプリメロ"}},
	{"Chronomaster", []string{"chronomaster", "クロノマスター"}},
	{"Defy", []string{"defy", "デファイ"}},
	{"Pilot", []string{"パイロット"}},
	// Breguet / Blancpain
	{"Classique", []string{"classique", "クラシック"}},
	{"Marine", []string{"marine", "マリーン"}},
	{"Type XX", []string{"type xx", "タイプxx", "type 20"}},
	{"Fifty Fathoms", []string{"fifty fathoms", "フィフティファゾムス", "五十噚"}},
	{"Villeret", []string{"villeret", "ヴィルレ"}},
	// Grand Seiko / Seiko
	{"Heritage Collection", []string{"heritage collection", "ヘリテージコレクション"}},
	{"Elegance Collection", []string{"elegance collection", "エレガンスコレクション"}},
	{"Sport Collection", []string{"sport collection", "スポーツコレクション"}},
	{"Prospex", []string{"prospex", "プロスペックス"}},
	{"Presage", []string{"presage", "プレザージュ"}},
	{"Astron", []string{"astron", "アストロン"}},
	// Others
	{"Overseas", []string{"overseas"}},
	{"Laureato", []string{"laureato", "ロレアート", "桂冠"}},
	{"Alpine Eagle", []string{"alpine eagle", "アルパインイーグル"}},
	{"Happy Sport", []string{"happy sport", "ハッピースポーツ", "快樂鑽石"}},
	{"Octo", []string{"octo", "オクト"}},
	{"Serpenti", []string{"serpenti", "セルペンティ"}},
	{"J12", []string{"j12"}},
	{"Première", []string{"première", "premiere", "プルミエール"}},
	{"Vanguard", []string{"vanguard", "ヴァンガード"}},
	{"Aquis", []string{"aquis", "アクイス"}},
	{"Big Crown", []string{"big crown", "ビッグクラウン"}},
	{"Tangente", []string{"tangente", "タンジェント"}},
	{"Khaki", []string{"khaki", "カーキ"}},
	{"PRX", []string{"prx"}},
	{"Endeavour", []string{"endeavour", "エンデバー"}},
	{"Streamliner", []string{"streamliner", "ストリームライナー"}},
	{"RM 011", []string{"rm011", "rm 011"}},
	{"RM 035", []string{"rm035", "rm 035"}},
	{"Polo", []string{"polo", "ポロ"}},
	{"Altiplano", []string{"altiplano", "アルティプラノ"}},
	{"Freak", []string{"freak", "フリーク"}},
	{"Marine Chronometer", []string{"marine chronometer", "マリーンクロノメーター"}},
	{"Tonda", []string{"tonda", "トンダ"}},
	{"BR 03", []string{"br 03", "br03"}},
	{"BR 05", []string{"br 05", "br05"}},
}

var modelMatchers []aliasMatcherModel

type aliasMatcherModel struct {
	alias string
	name  string
	ascii bool
}

func init() {
	for _, m := range Models {
		for _, a := range m.Aliases {
			modelMatchers = append(modelMatchers, aliasMatcherModel{alias: fold(a), name: m.Name, ascii: isASCII(a)})
		}
	}
	sort.SliceStable(modelMatchers, func(i, j int) bool {
		return len([]rune(modelMatchers[i].alias)) > len([]rune(modelMatchers[j].alias))
	})
}

// DetectModel returns the canonical English model name found in the texts, or "".
func DetectModel(texts ...string) string {
	for _, t := range texts {
		f := fold(t)
		if f == "" {
			continue
		}
		for _, m := range modelMatchers {
			if m.ascii {
				if containsWord(f, m.alias) {
					return m.name
				}
			} else if strings.Contains(f, m.alias) {
				return m.name
			}
		}
	}
	return ""
}

// CanonicalizeQuery rewrites a free-text search so brand and model aliases in any supported
// language become their English canonical names ("勞力士 水鬼 2020" → "Rolex Submariner 2020").
// Returns the same string when nothing was recognized.
func CanonicalizeQuery(q string) string {
	f := fold(q)
	if f == "" {
		return q
	}
	out := f
	for _, m := range aliasMatchers {
		if m.ascii {
			continue // keep ascii words as typed; tsvector handles them
		}
		if strings.Contains(out, m.alias) {
			out = strings.ReplaceAll(out, m.alias, " "+m.entry.Name+" ")
		}
	}
	for _, m := range modelMatchers {
		if m.ascii {
			continue
		}
		if strings.Contains(out, m.alias) {
			out = strings.ReplaceAll(out, m.alias, " "+m.name+" ")
		}
	}
	out = strings.Join(strings.Fields(out), " ")
	if out == f {
		return q
	}
	return out
}
