// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds every setting shared by the API server and the crawler job.
type Config struct {
	Env         string // local | staging | production
	Port        int
	DatabaseURL string
	LogLevel    string

	// API
	CORSAllowedOrigins []string
	CacheTTL           time.Duration

	// Crawler
	CrawlSources              []string // empty = all enabled sources
	CrawlConcurrency          int
	CrawlMaxPages             int
	CrawlRateLimitMS          int // minimum ms between requests to the same host
	CrawlDryRun               bool
	CrawlDeactivateStaleAfter time.Duration
	UserAgent                 string
	ProxyURL                  string // optional forward proxy (e.g. for Chrono24)
	RenderServiceURL          string // optional headless-render service base URL (Browserless on Cloud Run)
	RenderServiceToken        string // Browserless TOKEN (sent as ?token=)
	RenderMode                string // browserless (default, /smart-scrape) | content (Browserless v2 /content) | get
	RenderLaunchJSON          string // Browserless launch options for content mode, e.g. {"stealth":true,"args":["--proxy-server=..."]}
	RenderExtraQuery          string // extra query string for the render endpoint, e.g. proxy=residential&proxyCountry=de
	RenderUseIDToken          bool   // attach a Google identity token (auto for *.run.app URLs)

	// Chrono24 (rendered pages are the expensive part: keep the nightly budget small)
	Chrono24Brands   []string // Chrono24 brand slugs to crawl
	Chrono24MaxPages int      // list pages per brand per night

	// Integrations
	EbayClientID     string
	EbayClientSecret string
	EbayMarketplaces []string
	FXProviderURL    string
}

// Load reads the environment and returns a validated Config.
func Load() (*Config, error) {
	c := &Config{
		Env:                       getenv("APP_ENV", "local"),
		Port:                      getenvInt("PORT", 8080),
		DatabaseURL:               os.Getenv("DATABASE_URL"),
		LogLevel:                  getenv("LOG_LEVEL", "info"),
		CORSAllowedOrigins:        splitList(getenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		CacheTTL:                  getenvDuration("CACHE_TTL", 60*time.Second),
		CrawlSources:              splitList(os.Getenv("CRAWL_SOURCES")),
		CrawlConcurrency:          getenvInt("CRAWL_CONCURRENCY", 3),
		CrawlMaxPages:             getenvInt("CRAWL_MAX_PAGES", 200),
		CrawlRateLimitMS:          getenvInt("CRAWL_RATE_LIMIT_MS", 1500),
		CrawlDryRun:               getenvBool("CRAWL_DRY_RUN", false),
		CrawlDeactivateStaleAfter: getenvDuration("CRAWL_DEACTIVATE_STALE_AFTER", 72*time.Hour),
		UserAgent:                 getenv("CRAWL_USER_AGENT", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0 Safari/537.36 WatchCompareBot/1.0 (+https://github.com/hsuanlee/watch-compare)"),
		ProxyURL:                  os.Getenv("CRAWL_PROXY_URL"),
		RenderServiceURL:          os.Getenv("CRAWL_RENDER_SERVICE_URL"),
		RenderServiceToken:        os.Getenv("CRAWL_RENDER_SERVICE_TOKEN"),
		RenderMode:                getenv("CRAWL_RENDER_MODE", "browserless"),
		RenderLaunchJSON:          os.Getenv("CRAWL_RENDER_LAUNCH_JSON"),
		RenderExtraQuery:          strings.TrimPrefix(os.Getenv("CRAWL_RENDER_EXTRA_QUERY"), "?"),
		RenderUseIDToken:          getenvBool("CRAWL_RENDER_USE_IDTOKEN", strings.HasSuffix(os.Getenv("CRAWL_RENDER_SERVICE_URL"), ".run.app")),
		Chrono24Brands:            splitList(getenv("CHRONO24_BRANDS", "rolex,omega,iwc,audemarspiguet,patekphilippe")),
		Chrono24MaxPages:          getenvInt("CHRONO24_MAX_PAGES", 3),
		EbayClientID:              os.Getenv("EBAY_CLIENT_ID"),
		EbayClientSecret:          os.Getenv("EBAY_CLIENT_SECRET"),
		EbayMarketplaces:          splitList(getenv("EBAY_MARKETPLACES", "EBAY_US")),
		FXProviderURL:             getenv("FX_PROVIDER_URL", "https://open.er-api.com/v6/latest/USD"),
	}
	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	return c, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenvBool(k string, def bool) bool {
	if v := os.Getenv(k); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getenvDuration(k string, def time.Duration) time.Duration {
	if v := os.Getenv(k); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func splitList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
