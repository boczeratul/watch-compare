// Package hongsheng crawls https://www.999watch.com (鴻昇名錶精品中心, Taichung; TWD) — an
// alapower shop with the same markup as Hourstack and RD Watch.
package hongsheng

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
func (Source) Key() string { return "999watch" }

// Crawl delegates to the shared alapower parser.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	return alapower.Site{
		BaseURL:        "https://www.999watch.com/",
		SellerName:     "鴻昇名錶",
		FetchDetails:   s.FetchDetails,
		Boilerplate:    []string{"誠摯邀請", "歡迎洽詢", "歡迎來店", "歡迎預約"},
		SkipCategories: []string{"配件", "錶帶"}, // 手錶配件及錶帶
	}.Crawl(ctx, env, emit)
}
