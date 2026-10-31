// Package egps crawls https://www.egps.com.tw (永生／永久名錶, Taichung; TWD) — an alapower shop on
// the products.asp / products_open.asp variant. Its home page also links jewellery and watch
// boxes under products.asp?cat=, so only the two watch categories (品牌男錶 cat=2, 品牌女錶 cat=1)
// are crawled; brand sub-categories (subcat=) are not needed since the parents list every piece.
package egps

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
func (Source) Key() string { return "egps" }

// Crawl delegates to the shared alapower parser.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	return alapower.Site{
		FetchDetails:  s.FetchDetails,
		BaseURL:       "https://www.egps.com.tw/",
		SellerName:    "永生／永久名錶",
		ListPath:      "products.asp",
		CategoryQuery: "&type=open",
		Categories:    []alapower.Category{{ID: "2", Name: "品牌男錶"}, {ID: "1", Name: "品牌女錶"}},
		SkipTitles:    []string{"公告"}, // shop notices are posted as products with no price
	}.Crawl(ctx, env, emit)
}
