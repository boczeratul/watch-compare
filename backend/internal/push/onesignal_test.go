package push

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOneSignalSend(t *testing.T) {
	var got map[string]any
	var auth, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, path = r.Header.Get("Authorization"), r.URL.RequestURI()
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &got)
		_, _ = w.Write([]byte(`{"id":"abc"}`))
	}))
	defer srv.Close()

	o := NewOneSignal("app-1", "secret")
	o.BaseURL = srv.URL
	err := o.Send(context.Background(), Notification{
		ExternalIDs: []string{"sub-1"},
		Headings:    map[string]string{"en": "New match"},
		Contents:    map[string]string{"en": "Rolex"},
		WebURL:      "https://example.com/listing/1",
		Data:        map[string]any{"listingId": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if auth != "Key secret" || path != "/notifications?c=push" {
		t.Errorf("auth=%q path=%q", auth, path)
	}
	if got["app_id"] != "app-1" || got["target_channel"] != "push" || got["web_url"] != "https://example.com/listing/1" {
		t.Errorf("body %v", got)
	}
	ids := got["include_aliases"].(map[string]any)["external_id"].([]any)
	if len(ids) != 1 || ids[0] != "sub-1" {
		t.Errorf("aliases %v", got["include_aliases"])
	}
}

func TestOneSignalSendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":["app_id not found"]}`))
	}))
	defer srv.Close()
	o := NewOneSignal("bad", "secret")
	o.BaseURL = srv.URL
	if err := o.Send(context.Background(), Notification{ExternalIDs: []string{"x"}, Contents: map[string]string{"en": "x"}}); err == nil {
		t.Error("expected an error for HTTP 400")
	}
}

func TestOneSignalCreateNotSubscribed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"","errors":["All included players are not subscribed"]}`))
	}))
	defer srv.Close()
	o := NewOneSignal("app-1", "secret")
	o.BaseURL = srv.URL
	res, err := o.Create(context.Background(), Notification{ExternalIDs: []string{"x"}, Contents: map[string]string{"en": "x"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.ID != "" || string(res.Errors) != `["All included players are not subscribed"]` {
		t.Errorf("result %+v", res)
	}
}
