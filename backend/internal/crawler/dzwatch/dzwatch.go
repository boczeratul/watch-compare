// Package dzwatch crawls https://www.dzwatch.tw (大眾名錶, Kaohsiung; TWD) — an alapower shop
// with the same markup as RD Watch. The menu also lists jewellery (GIA 鑽石, 玉石, 珍珠 …) and
// watch parts under product.asp?cat=, which are skipped.
package dzwatch

import (
	"context"

	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler/alapower"
)

// Source implements crawler.Source.
type Source struct {
	FetchDetails bool
}

// Key returns the source key.
func (Source) Key() string { return "dzwatch" }

// Crawl delegates to the shared alapower parser.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	return alapower.Site{
		FetchDetails:   s.FetchDetails,
		BaseURL:        "https://www.dzwatch.tw/",
		SellerName:     "大眾名錶",
		SkipCategories: []string{"配件", "鑽", "寶石", "玉石", "珍珠", "珊瑚", "飾品", "其他"},
	}.Crawl(ctx, env, emit)
}
