// Package jinchen crawls https://www.369rolexwatch.com (金宸名錶, Taichung; TWD) — an
// alapower shop whose category pager uses index=<p> and whose list cards show "售價 : 368,000元"
// as plain text.
package jinchen

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
func (Source) Key() string { return "369rolexwatch" }

// Crawl delegates to the shared alapower parser.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	return alapower.Site{
		FetchDetails:   s.FetchDetails,
		BaseURL:        "https://www.369rolexwatch.com/",
		SellerName:     "金宸名錶",
		SessionPager:   true,
		SkipCategories: []string{"配件", "錶帶"},
	}.Crawl(ctx, env, emit)
}
