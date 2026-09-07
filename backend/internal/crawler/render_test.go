package crawler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/config"
)

func TestRendererSmartScrape(t *testing.T) {
	var gotPath, gotToken, gotTimeout, gotURL, gotFormats string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.URL.Query().Get("token")
		gotTimeout = r.URL.Query().Get("timeout")
		if r.URL.Query().Get("proxy") != "residential" || r.URL.Query().Get("proxyCountry") != "de" {
			t.Errorf("extra query not forwarded: %s", r.URL.RawQuery)
		}
		if r.URL.Query().Has("launch") {
			t.Errorf("launch options must not be sent to /smart-scrape: %s", r.URL.RawQuery)
		}
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(405)
			return
		}
		var body struct {
			URL     string   `json:"url"`
			Formats []string `json:"formats"`
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		gotURL, gotFormats = body.URL, strings.Join(body.Formats, ",")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"statusCode":200,"content":"<html><body>rendered</body></html>","contentType":"text/html","strategy":"http-fetch","attempted":["http-fetch"],"message":null}`))
	}))
	defer srv.Close()

	cfg := &config.Config{RenderServiceURL: srv.URL + "/", RenderServiceToken: "secret", RenderLaunchJSON: `{"stealth":true}`, RenderExtraQuery: "proxy=residential&proxyCountry=de"}
	r := NewRenderer(cfg, srv.Client())
	if r == nil || r.audience != "" {
		t.Fatalf("expected renderer without identity token for non-run.app URL, got %+v", r)
	}
	html, err := r.Render(context.Background(), "https://www.chrono24.com/rolex/index.htm")
	if err != nil {
		t.Fatal(err)
	}
	if string(html) != "<html><body>rendered</body></html>" {
		t.Errorf("unexpected body %q", html)
	}
	if gotPath != "/smart-scrape" || gotToken != "secret" || gotTimeout != "60000" || gotURL != "https://www.chrono24.com/rolex/index.htm" || gotFormats != "html" {
		t.Errorf("request mismatch: path=%s token=%s timeout=%s url=%s formats=%s", gotPath, gotToken, gotTimeout, gotURL, gotFormats)
	}
}

func TestRendererSmartScrapeBlocked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"statusCode":403,"content":"<html>Just a moment...</html>","strategy":"browser","attempted":["http-fetch","browser"],"message":"all strategies failed"}`))
	}))
	defer srv.Close()
	r := NewRenderer(&config.Config{RenderServiceURL: srv.URL}, srv.Client())
	body, err := r.Render(context.Background(), "https://www.chrono24.com/rolex/index.htm")
	if err == nil || !strings.Contains(err.Error(), "403") || !strings.Contains(err.Error(), "all strategies failed") {
		t.Fatalf("expected a blocked-page error, got body=%q err=%v", body, err)
	}
}

func TestRendererContentMode(t *testing.T) {
	var gotPath, gotToken, gotLaunch, gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.URL.Query().Get("token")
		gotLaunch = r.URL.Query().Get("launch")
		var body struct {
			URL string `json:"url"`
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		gotURL = body.URL
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body>rendered</body></html>"))
	}))
	defer srv.Close()

	cfg := &config.Config{RenderServiceURL: srv.URL, RenderMode: "content", RenderServiceToken: "secret", RenderLaunchJSON: `{"stealth":true}`}
	html, err := NewRenderer(cfg, srv.Client()).Render(context.Background(), "https://www.chrono24.com/rolex/index.htm")
	if err != nil {
		t.Fatal(err)
	}
	if string(html) != "<html><body>rendered</body></html>" {
		t.Errorf("unexpected body %q", html)
	}
	if gotPath != "/content" || gotToken != "secret" || gotLaunch != `{"stealth":true}` || gotURL != "https://www.chrono24.com/rolex/index.htm" {
		t.Errorf("request mismatch: path=%s token=%s launch=%s url=%s", gotPath, gotToken, gotLaunch, gotURL)
	}
}

func TestRendererGetMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Query().Get("url") != "https://example.com/x" {
			w.WriteHeader(400)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	r := NewRenderer(&config.Config{RenderServiceURL: srv.URL, RenderMode: "get"}, srv.Client())
	body, err := r.Render(context.Background(), "https://example.com/x")
	if err != nil || string(body) != "ok" {
		t.Fatalf("get mode: %q %v", body, err)
	}
}

func TestRendererCloudRunAudience(t *testing.T) {
	r := NewRenderer(&config.Config{RenderServiceURL: "https://browserless-abc-de.a.run.app"}, http.DefaultClient)
	if r.audience != "https://browserless-abc-de.a.run.app" {
		t.Errorf("expected Cloud Run audience, got %q", r.audience)
	}
	if NewRenderer(&config.Config{}, http.DefaultClient) != nil {
		t.Error("expected nil renderer when unconfigured")
	}
}
