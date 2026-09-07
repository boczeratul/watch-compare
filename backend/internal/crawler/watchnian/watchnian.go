// Package watchnian crawls https://watchnian.com (Japan, JPY).
//
// Site structure (verified 2026-09): ebisumart shop, UTF-8. Note the bare host: the
// www. host returned CloudFront 502s during verification.
//   - All watches: /shop/r/rwatch/  → page n is /shop/r/rwatch_p<n>/
//   - Cards: dl.block-thumbnail-t--goods > a[href="/shop/g/g<id>/"][title][data-category1="ロレックス(rl)"]
//     img.lazyload[data-src], .block-thumbnail-t--price .num, .soldout,
//     .block-thumbnail-t--conditions span (中古/新品, rank A/B, メンズ/レディース)
package watchnian

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

const (
	baseURL = "https://watchnian.com"
	listKey = "rwatch"
)

// Source implements crawler.Source.
type Source struct{}

// Key returns the source key.
func (Source) Key() string { return "watchnian" }

var (
	idRe    = regexp.MustCompile(`/shop/g/g([A-Za-z0-9_-]+)/`)
	brandRe = regexp.MustCompile(`^(.*?)\(([a-z0-9-]+)\)$`)
	rankRe  = regexp.MustCompile(`^(S|SA|A|AB|B|BC|C|D|N)$`)
)

// Crawl pages through the all-watches list.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	seen := map[string]bool{}
	for page := 1; page <= env.MaxPages; page++ {
		u := fmt.Sprintf("%s/shop/r/%s/", baseURL, listKey)
		if page > 1 {
			u = fmt.Sprintf("%s/shop/r/%s_p%d/", baseURL, listKey, page)
		}
		doc, err := env.Fetcher.Doc(ctx, u)
		if err != nil {
			if crawler.IsNotFound(err) && page > 1 {
				return nil
			}
			return fmt.Errorf("page %d: %w", page, err)
		}
		items := ParseList(doc)
		if len(items) == 0 {
			if page == 1 {
				return fmt.Errorf("no items on first page (markup changed?)")
			}
			return nil
		}
		fresh := 0
		for _, l := range items {
			if seen[l.ExternalID] {
				continue
			}
			seen[l.ExternalID] = true
			fresh++
			if err := emit(l); err != nil {
				return err
			}
		}
		if fresh == 0 {
			return nil
		}
	}
	return nil
}

// ParseList extracts listings from a list page. Exported for fixture tests.
func ParseList(doc *goquery.Document) []model.Listing {
	var out []model.Listing
	// Restrict to the main goods list when present; fall back to any card.
	scope := doc.Find(".block-goods-list--items, .block-goods-list, .block-thumbnail-t").Not(".block-top-ranking-slider *")
	if scope.Length() == 0 {
		scope = doc.Selection
	}
	scope.Find("dl.block-thumbnail-t--goods").Each(func(_ int, dl *goquery.Selection) {
		if dl.ParentsFiltered(".block-top-ranking-slider, .swiper").Length() > 0 {
			return
		}
		a := dl.Find("a[href]").First()
		href, _ := a.Attr("href")
		m := idRe.FindStringSubmatch(href)
		if m == nil {
			return
		}
		if dl.Find(".soldout").Length() > 0 {
			return
		}
		title, _ := a.Attr("title")
		if title == "" {
			title = crawler.Text(dl.Find(".block-thumbnail-t--goods-name"))
		}
		l := model.Listing{
			ExternalID:      m[1],
			URL:             crawler.AbsURL(doc, href),
			Title:           normalize.CleanText(strings.NewReplacer("【中古】", "", "【新品】", "", "【wristwatch】", "", "【未使用】", "").Replace(title)),
			Currency:        "JPY",
			SellerType:      "dealer",
			SellerName:      "Watchnian",
			LocationCountry: "JP",
		}
		cat1, _ := a.Attr("data-category1")
		if bm := brandRe.FindStringSubmatch(cat1); bm != nil {
			if b := normalize.DetectBrand(bm[1]); b != nil {
				l.BrandSlug, l.BrandName = b.Slug, b.Name
			}
		}
		if src, ok := dl.Find("img").First().Attr("data-src"); ok && src != "" {
			// Watchnian serves the main image only as /img/goods/S/<id>-1.jpg (no larger variant).
			l.ImageURLs = []string{crawler.AbsURL(doc, src)}
		}
		price := dl.Find(".js-enhanced-ecommerce-goods-price .num").First()
		if price.Length() == 0 {
			price = dl.Find(".block-thumbnail-t--price .num").First()
		}
		if p, ok := normalize.ParsePrice(crawler.Text(price)); ok {
			l.Price = &p
		}
		var conds []string
		dl.Find(".block-thumbnail-t--conditions span").Each(func(_ int, sp *goquery.Selection) {
			t := crawler.Text(sp)
			conds = append(conds, t)
			switch {
			case rankRe.MatchString(t):
				l.Condition = rankToCondition(t, l.Condition)
			case strings.Contains(t, "未使用"):
				l.Condition = model.ConditionUnworn
			case strings.Contains(t, "新品"):
				l.Condition = model.ConditionNew
			case strings.Contains(t, "中古") && l.Condition == "":
				l.Condition = model.ConditionGood
			case strings.Contains(t, "メンズ"):
				l.Gender = model.GenderMen
			case strings.Contains(t, "レディース"):
				l.Gender = model.GenderWomen
			case strings.Contains(t, "ユニセックス"):
				l.Gender = model.GenderUnisex
			}
		})
		attrs := map[string]any{"conditions": conds}
		for _, k := range []string{"data-category2", "data-category3", "data-category4"} {
			if v, ok := a.Attr(k); ok && v != "" {
				attrs[strings.TrimPrefix(k, "data-")] = v
			}
		}
		if cat2, ok := a.Attr("data-category2"); ok {
			if bm := brandRe.FindStringSubmatch(cat2); bm != nil {
				l.Model = bm[1]
			}
		}
		l.Attributes, _ = json.Marshal(attrs)
		out = append(out, l)
	})
	return out
}

// rankToCondition maps Japanese dealer grades (S/A/B/C) to the normalized scale.
func rankToCondition(rank string, current model.Condition) model.Condition {
	switch rank {
	case "N", "S":
		if current == model.ConditionUnworn {
			return current
		}
		return model.ConditionNew
	case "SA", "A":
		return model.ConditionVeryGood
	case "AB", "B":
		return model.ConditionGood
	case "BC", "C":
		return model.ConditionFair
	case "D":
		return model.ConditionPoor
	}
	return current
}
