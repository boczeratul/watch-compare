package crawler_test

import (
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/config"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler/ebay"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler/jackroad"
)

func TestPreflight(t *testing.T) {
	var _ crawler.Preflighter = &ebay.Source{}
	if _, ok := any(jackroad.Source{}).(crawler.Preflighter); ok {
		t.Error("jackroad needs no configuration and must not implement Preflighter")
	}
	if err := (&ebay.Source{}).Preflight(&config.Config{}); err == nil {
		t.Error("ebay without credentials must fail preflight")
	}
	if err := (&ebay.Source{}).Preflight(&config.Config{EbayClientID: "id", EbayClientSecret: "s"}); err != nil {
		t.Errorf("ebay with credentials: %v", err)
	}
}
