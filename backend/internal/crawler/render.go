package crawler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/hsuanlee/watch-compare/backend/internal/config"
)

// Renderer fetches a page through a headless browser service. Used for sites that block plain
// HTTP clients (Chrono24). Two protocols are supported, selected by CRAWL_RENDER_MODE:
//
//   - "browserless" (default): Browserless v2 REST API — POST <base>/content with a JSON body
//     {"url": ...}. Works with the official image (ghcr.io/browserless/chromium) self-hosted on
//     Cloud Run or with browserless.io. Optional ?token= (TOKEN env of the container) and optional
//     launch options (stealth, proxy) via CRAWL_RENDER_LAUNCH_JSON.
//   - "get": a generic endpoint that answers GET <base>?url=<page> with rendered HTML.
//
// When the service is a private Cloud Run service (*.run.app) the client automatically attaches a
// Google-signed identity token from the metadata server, so the service can stay
// --no-allow-unauthenticated and be limited to the crawler's service account.
type Renderer struct {
	client   *http.Client
	baseURL  string
	mode     string
	token    string
	launch   string
	extra    url.Values
	audience string
	timeout  time.Duration

	idMu     sync.Mutex
	idToken  string
	idExpiry time.Time
}

// NewRenderer returns nil when no render service is configured.
func NewRenderer(cfg *config.Config, client *http.Client) *Renderer {
	if cfg.RenderServiceURL == "" {
		return nil
	}
	mode := strings.ToLower(cfg.RenderMode)
	if mode == "" {
		mode = "browserless"
	}
	r := &Renderer{
		client:  client,
		baseURL: strings.TrimRight(cfg.RenderServiceURL, "/"),
		mode:    mode,
		token:   cfg.RenderServiceToken,
		launch:  cfg.RenderLaunchJSON,
		timeout: 60 * time.Second,
	}
	if cfg.RenderExtraQuery != "" {
		if v, err := url.ParseQuery(cfg.RenderExtraQuery); err == nil {
			r.extra = v
		} else {
			log.Warn().Err(err).Msg("CRAWL_RENDER_EXTRA_QUERY is not a valid query string; ignored")
		}
	}
	if cfg.RenderUseIDToken || strings.HasSuffix(mustHost(r.baseURL), ".run.app") {
		// Cloud Run validates the token audience against the service URL.
		r.audience = r.baseURL
	}
	return r
}

func mustHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Host
}

// Render returns the rendered HTML of pageURL.
func (r *Renderer) Render(ctx context.Context, pageURL string) ([]byte, error) {
	var req *http.Request
	var err error
	switch r.mode {
	case "get":
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL+"?url="+url.QueryEscape(pageURL), nil)
	default:
		req, err = r.browserlessRequest(ctx, pageURL)
	}
	if err != nil {
		return nil, err
	}
	if r.audience != "" {
		tok, err := r.identityToken(ctx)
		if err != nil {
			return nil, fmt.Errorf("render service identity token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		resp, err := r.client.Do(req)
		if err != nil {
			lastErr = err
		} else {
			body, rerr := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
			resp.Body.Close()
			switch {
			case rerr != nil:
				lastErr = rerr
			case resp.StatusCode == http.StatusOK:
				return body, nil
			case resp.StatusCode == 429 || resp.StatusCode >= 500:
				lastErr = fmt.Errorf("render service: http %d: %s", resp.StatusCode, truncate(string(body), 200))
			default:
				return nil, fmt.Errorf("render service: http %d: %s", resp.StatusCode, truncate(string(body), 200))
			}
		}
		backoff := time.Duration(1<<attempt) * 3 * time.Second
		log.Warn().Err(lastErr).Str("page", pageURL).Dur("backoff", backoff).Msg("render retry")
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		// the body of a POST can only be sent once: rebuild the request
		if req, err = r.rebuild(ctx, pageURL, req.Header.Get("Authorization")); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func (r *Renderer) rebuild(ctx context.Context, pageURL, auth string) (*http.Request, error) {
	var req *http.Request
	var err error
	if r.mode == "get" {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL+"?url="+url.QueryEscape(pageURL), nil)
	} else {
		req, err = r.browserlessRequest(ctx, pageURL)
	}
	if err == nil && auth != "" {
		req.Header.Set("Authorization", auth)
	}
	return req, err
}

// browserlessRequest builds POST /content for Browserless v2.
func (r *Renderer) browserlessRequest(ctx context.Context, pageURL string) (*http.Request, error) {
	q := url.Values{}
	for k, vs := range r.extra {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	if r.token != "" {
		q.Set("token", r.token)
	}
	if r.launch != "" {
		q.Set("launch", r.launch)
	}
	endpoint := r.baseURL + "/content"
	if enc := q.Encode(); enc != "" {
		endpoint += "?" + enc
	}
	payload := map[string]any{
		"url": pageURL,
		"gotoOptions": map[string]any{
			"waitUntil": "networkidle2",
			"timeout":   int(r.timeout / time.Millisecond),
		},
		"rejectResourceTypes": []string{"image", "media", "font"},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/html")
	return req, nil
}

// identityToken fetches (and caches) a Google-signed OIDC token for the service audience from the
// GCE/Cloud Run metadata server. Outside Google Cloud set CRAWL_RENDER_USE_IDTOKEN=false and rely
// on the Browserless token instead.
func (r *Renderer) identityToken(ctx context.Context) (string, error) {
	r.idMu.Lock()
	defer r.idMu.Unlock()
	if r.idToken != "" && time.Now().Before(r.idExpiry) {
		return r.idToken, nil
	}
	u := "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?audience=" + url.QueryEscape(r.audience)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Metadata-Flavor", "Google")
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("metadata server unreachable (not on Google Cloud? set CRAWL_RENDER_USE_IDTOKEN=false): %w", err)
	}
	defer resp.Body.Close()
	tok, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("metadata server: http %d", resp.StatusCode)
	}
	r.idToken = strings.TrimSpace(string(tok))
	r.idExpiry = time.Now().Add(50 * time.Minute) // tokens last 1h
	return r.idToken, nil
}
