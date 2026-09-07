// Package crawler contains the polite HTTP fetcher, the Source interface every
// marketplace adapter implements, and the Runner that persists results.
package crawler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/html/charset"
	"golang.org/x/time/rate"

	"github.com/hsuanlee/watch-compare/backend/internal/config"
)

// Fetcher is an HTTP client with per-host rate limiting, retries and charset decoding.
type Fetcher struct {
	client    *http.Client
	userAgent string
	every     time.Duration
	renderURL string
	mu        sync.Mutex
	limiters  map[string]*rate.Limiter
}

// NewFetcher builds a Fetcher from config (proxy, UA, rate limit, render service).
func NewFetcher(cfg *config.Config) *Fetcher {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = 4
	if cfg.ProxyURL != "" {
		if pu, err := url.Parse(cfg.ProxyURL); err == nil {
			transport.Proxy = http.ProxyURL(pu)
		}
	}
	return &Fetcher{
		client:    &http.Client{Transport: transport, Timeout: 45 * time.Second},
		userAgent: cfg.UserAgent,
		every:     time.Duration(cfg.CrawlRateLimitMS) * time.Millisecond,
		renderURL: cfg.RenderServiceURL,
		limiters:  map[string]*rate.Limiter{},
	}
}

// Client exposes the underlying client for API-based sources (eBay).
func (f *Fetcher) Client() *http.Client { return f.client }

func (f *Fetcher) limiter(host string) *rate.Limiter {
	f.mu.Lock()
	defer f.mu.Unlock()
	l, ok := f.limiters[host]
	if !ok {
		l = rate.NewLimiter(rate.Every(f.every), 1)
		f.limiters[host] = l
	}
	return l
}

// RetryableError marks a transient failure.
type RetryableError struct{ Err error }

func (e *RetryableError) Error() string { return e.Err.Error() }
func (e *RetryableError) Unwrap() error { return e.Err }

// Do executes a request with rate limiting and up to 3 retries on transient errors.
func (f *Fetcher) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", f.userAgent)
	}
	if req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", "en-US,en;q=0.9,ja;q=0.8,zh-TW;q=0.7")
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/json;q=0.9,*/*;q=0.8")
	}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if err := f.limiter(req.URL.Host).Wait(ctx); err != nil {
			return nil, err
		}
		resp, err := f.client.Do(req.WithContext(ctx))
		if err != nil {
			lastErr = &RetryableError{err}
		} else if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = &RetryableError{fmt.Errorf("%s: http %d", req.URL, resp.StatusCode)}
		} else {
			return resp, nil
		}
		backoff := time.Duration(1<<attempt) * 2 * time.Second
		log.Warn().Err(lastErr).Str("url", req.URL.String()).Dur("backoff", backoff).Msg("fetch retry")
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}
	return nil, lastErr
}

// Get downloads a URL and returns the body decoded to UTF-8.
func (f *Fetcher) Get(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := f.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: http %d", rawURL, resp.StatusCode)
	}
	return decodeBody(resp)
}

// GetRendered fetches through the optional headless render service (for JS/anti-bot sites).
// Falls back to a plain Get when no render service is configured.
func (f *Fetcher) GetRendered(ctx context.Context, rawURL string) ([]byte, error) {
	if f.renderURL == "" {
		return f.Get(ctx, rawURL)
	}
	u := f.renderURL
	if strings.Contains(u, "?") {
		u += "&url=" + url.QueryEscape(rawURL)
	} else {
		u += "?url=" + url.QueryEscape(rawURL)
	}
	return f.Get(ctx, u)
}

// Doc fetches a URL and parses it as HTML.
func (f *Fetcher) Doc(ctx context.Context, rawURL string) (*goquery.Document, error) {
	body, err := f.Get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	return ParseHTML(body, rawURL)
}

// DocRendered is Doc via the render service.
func (f *Fetcher) DocRendered(ctx context.Context, rawURL string) (*goquery.Document, error) {
	body, err := f.GetRendered(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	return ParseHTML(body, rawURL)
}

// ParseHTML builds a goquery document and records the page URL for AbsURL.
func ParseHTML(body []byte, pageURL string) (*goquery.Document, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if u, err := url.Parse(pageURL); err == nil {
		doc.Url = u
	}
	return doc, nil
}

func decodeBody(resp *http.Response) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "json") {
		return raw, nil
	}
	r, err := charset.NewReader(bytes.NewReader(raw), ct)
	if err != nil {
		return raw, nil //nolint:nilerr // undecodable charset: return raw bytes
	}
	out, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AbsURL resolves href against the document URL.
func AbsURL(doc *goquery.Document, href string) string {
	href = strings.TrimSpace(href)
	if href == "" || doc.Url == nil {
		return href
	}
	u, err := url.Parse(href)
	if err != nil {
		return href
	}
	return doc.Url.ResolveReference(u).String()
}

// Text returns trimmed, whitespace-collapsed text of a selection.
func Text(s *goquery.Selection) string {
	return strings.Join(strings.Fields(s.Text()), " ")
}

// IsNotFound reports whether err is a 404.
func IsNotFound(err error) bool {
	return err != nil && strings.HasSuffix(err.Error(), "http 404")
}

var errStop = errors.New("stop")

// ErrStop can be returned by an emit callback to end a crawl early (e.g. dry-run limits).
var ErrStop = errStop
