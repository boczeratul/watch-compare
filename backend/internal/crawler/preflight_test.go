package crawler_test

import (
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/config"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler/chrono24"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler/ebay"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler/jackroad"
)

func TestChrono24Brands(t *testing.T) {
	got := chrono24.Brands(&config.Config{})
	if len(got) != 5 || got[0] != "rolex" || got[3] != "audemarspiguet" || got[4] != "patekphilippe" {
		t.Errorf("default brands: %v", got)
	}
	got = chrono24.Brands(&config.Config{Chrono24Brands: []string{"Rolex", " patek-philippe ", "tudor"}})
	if len(got) != 3 || got[1] != "patekphilippe" || got[2] != "tudor" {
		t.Errorf("configured brands: %v", got)
	}
}

func TestPreflight(t *testing.T) {
	var _ crawler.Preflighter = chrono24.Source{}
	var _ crawler.Preflighter = &ebay.Source{}
	if _, ok := any(jackroad.Source{}).(crawler.Preflighter); ok {
		t.Error("jackroad needs no configuration and must not implement Preflighter")
	}
	if err := (chrono24.Source{}).Preflight(&config.Config{}); err == nil {
		t.Error("chrono24 without proxy/render service must fail preflight")
	}
	if err := (chrono24.Source{}).Preflight(&config.Config{ProxyURL: "http://proxy:8080"}); err != nil {
		t.Errorf("chrono24 with proxy: %v", err)
	}
	if err := (&ebay.Source{}).Preflight(&config.Config{}); err == nil {
		t.Error("ebay without credentials must fail preflight")
	}
	if err := (&ebay.Source{}).Preflight(&config.Config{EbayClientID: "id", EbayClientSecret: "s"}); err != nil {
		t.Errorf("ebay with credentials: %v", err)
	}
}
