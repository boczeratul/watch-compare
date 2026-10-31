// Package goodtimezone crawls https://www.goodtimezone.com.tw (好時計, Taipei; TWD) — an
// alapower shop with lowercase product_open.asp?id= links, /product/product_BIG/ photos and an
// index=<p> pager.
package goodtimezone

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
func (Source) Key() string { return "goodtimezone" }

// Crawl delegates to the shared alapower parser.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	return alapower.Site{
		FetchDetails:   s.FetchDetails,
		BaseURL:        "https://www.goodtimezone.com.tw/",
		SellerName:     "好時計",
		SessionPager:   true,
		SkipCategories: []string{"配件", "錶帶"},
	}.Crawl(ctx, env, emit)
}
