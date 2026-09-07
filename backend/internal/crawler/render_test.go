package crawler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/config"
)

func TestRendererBrowserless(t *testing.T) {
	var gotPath, gotToken, gotLaunch, gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotToken = r.URL.Query().Get("token")
		gotLaunch = r.URL.Query().Get("launch")
		if r.URL.Query().Get("proxy") != "residential" || r.URL.Query().Get("proxyCountry") != "de" {
			t.Errorf("extra query not forwarded: %s", r.URL.RawQuery)
		}
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
