// Package rdwatch crawls https://www.rdwatch.com.tw (RD Watch, Taipei; TWD) — an alapower shop
// with the same markup as Hourstack (cards are 230px columns, prices without thousands separators).
package rdwatch

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
func (Source) Key() string { return "rdwatch" }

// Crawl delegates to the shared alapower parser.
func (s Source) Crawl(ctx context.Context, env *crawler.Env, emit crawler.Emit) error {
	return alapower.Site{
		BaseURL:      "https://www.rdwatch.com.tw/",
		SellerName:   "RD Watch",
		FetchDetails: s.FetchDetails,
		Boilerplate:  []string{"誠摯邀請", "歡迎洽詢", "歡迎來店", "歡迎預約"},
	}.Crawl(ctx, env, emit)
}
