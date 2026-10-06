package alerts

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/hsuanlee/watch-compare/backend/internal/fx"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/push"
)

var conv = &fx.Converter{Rates: map[string]float64{"USD": 1, "TWD": 32, "JPY": 150}}

func TestNormalize(t *testing.T) {
	got, err := Normalize("?sort=price_asc&page=3&brand=rolex&price_max=300000&currency=twd&q=+sub+", conv)
	if err != nil {
		t.Fatal(err)
	}
	if want := "brand=rolex&currency=TWD&price_max=300000&q=sub"; got != want {
		t.Errorf("Normalize = %q, want %q", got, want)
	}
	for _, bad := range []string{"", "sort=newest&currency=USD", "price_max=abc", "brand=rolex&price_max=10&currency=XYZ"} {
		if _, err := Normalize(bad, conv); err == nil {
			t.Errorf("Normalize(%q) accepted", bad)
		}
	}
}

func TestDefaultName(t *testing.T) {
	cases := map[string]string{
		"brand=rolex&currency=TWD&price_max=300000&ref=116610LN":    "rolex · 116610LN · ≤ 300,000 TWD",
		"currency=USD&price_min=5000&price_max=12000&q=speedmaster": "speedmaster · 5,000–12,000 USD",
		"brand=rolex,omega&currency=USD":                            "rolex, omega",
		"condition=new&currency=USD":                                "Saved search",
	}
	for q, want := range cases {
		if got := DefaultName(q); got != want {
			t.Errorf("DefaultName(%q) = %q, want %q", q, got, want)
		}
	}
}

func TestName(t *testing.T) {
	q := "brand=rolex&currency=USD"
	long := strings.Repeat("錶", 100)
	cases := map[string]string{
		"":                  "rolex",
		"   ":               "rolex",
		"  My  daily\tGMT ": "My daily GMT",
		long:                strings.Repeat("錶", MaxNameLen-1) + "…",
	}
	for in, want := range cases {
		if got := Name(in, q); got != want {
			t.Errorf("Name(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidSubscriberID(t *testing.T) {
	if !ValidSubscriberID("0B6F1A2E-5C1D-4B8E-9F3A-2D7C6E1F0A9B") {
		t.Error("UUID rejected")
	}
	for _, bad := range []string{"", "short", "0b6f1a2e 5c1d 4b8e 9f3a 2d7c6e1f0a9b", strings.Repeat("a", 65)} {
		if ValidSubscriberID(bad) {
			t.Errorf("accepted %q", bad)
		}
	}
}

type fakeStore struct {
	alerts  []model.Alert
	matches map[int64][]model.Listing // by alert id, keyed via the brand filter below
	checked map[int64]bool            // id -> notified
	since   map[int64]time.Time
}

func (f *fakeStore) AllAlerts(context.Context) ([]model.Alert, error) { return f.alerts, nil }

func (f *fakeStore) NewMatches(_ context.Context, q model.ListingQuery, limit int) (int, []model.Listing, error) {
	for _, a := range f.alerts {
		if strings.Contains(a.Query, "brand="+q.Brands[0]) {
			f.since[a.ID] = *q.FirstSeenAfter
			m := f.matches[a.ID]
			if len(m) > limit {
				return len(m), m[:limit], nil
			}
			return len(m), m, nil
		}
	}
	return 0, nil, nil
}

func (f *fakeStore) MarkAlertChecked(_ context.Context, id int64, _ time.Time, notified bool) error {
	f.checked[id] = notified
	return nil
}

type fakeSender struct {
	sent []push.Notification
	fail bool
}

func (s *fakeSender) Send(_ context.Context, n push.Notification) error {
	if s.fail {
		return errors.New("boom")
	}
	s.sent = append(s.sent, n)
	return nil
}

func TestNotifierRun(t *testing.T) {
	created := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	usd1, usd2 := 9375.0, 8000.0
	store := &fakeStore{
		alerts: []model.Alert{
			{ID: 1, SubscriberID: "sub-a", Name: "rolex", Query: "brand=rolex&currency=TWD", CheckedAt: created},
			{ID: 2, SubscriberID: "sub-b", Name: "omega", Query: "brand=omega&currency=USD", CheckedAt: created},
			{ID: 3, SubscriberID: "sub-c", Name: "tudor", Query: "brand=tudor&currency=USD", CheckedAt: created},
		},
		matches: map[int64][]model.Listing{
			1: {{ID: 42, Title: "Rolex Submariner 116610LN", SourceName: "Hourstack", PriceUSD: &usd1}},
			2: {{ID: 7, Title: "Omega Speedmaster", SourceName: "Jackroad", PriceUSD: &usd2}, {ID: 8}, {ID: 9}},
		},
		checked: map[int64]bool{},
		since:   map[int64]time.Time{},
	}
	sender := &fakeSender{}
	n := &Notifier{Store: store, Push: sender, Conv: conv, SiteURL: "https://example.com/", Log: zerolog.Nop(),
		Now: func() time.Time { return created.Add(24 * time.Hour) }}
	res, err := n.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Alerts != 3 || res.Notified != 2 || res.Failed != 0 {
		t.Errorf("result %+v", res)
	}
	if !store.since[1].Equal(created) {
		t.Errorf("matched from %v, want the alert's watermark", store.since[1])
	}
	if len(sender.sent) != 2 {
		t.Fatalf("sent %d", len(sender.sent))
	}
	one := sender.sent[0]
	if one.ExternalIDs[0] != "sub-a" || one.WebURL != "https://example.com/listing/42" || one.Data["listingId"] != int64(42) {
		t.Errorf("single: %+v", one)
	}
	if one.Headings["en"] != "New match: rolex" || one.Contents["en"] != "Rolex Submariner 116610LN · 300,000 TWD · Hourstack" {
		t.Errorf("single copy: %q / %q", one.Headings["en"], one.Contents["en"])
	}
	for _, lang := range []string{"en", "zh-Hant", "zh-Hans", "ja", "de"} {
		if one.Headings[lang] == "" || one.Contents[lang] == "" {
			t.Errorf("missing %s copy", lang)
		}
	}
	many := sender.sent[1]
	if many.Headings["en"] != "3 new matches: omega" || many.Contents["en"] != "From 8,000 USD at Jackroad. Tap to see them all." {
		t.Errorf("multi copy: %q / %q", many.Headings["en"], many.Contents["en"])
	}
	if many.WebURL != "https://example.com/search?brand=omega&currency=USD&sort=newest" || many.Data["listingId"] != nil {
		t.Errorf("multi: %+v", many)
	}
	if !store.checked[1] || !store.checked[2] {
		t.Errorf("notified alerts not stamped: %v", store.checked)
	}
	if notified, ok := store.checked[3]; !ok || notified {
		t.Errorf("alert without finds should advance its watermark without a notification: %v", store.checked)
	}
}

func TestNotifierKeepsWatermarkWhenSendFails(t *testing.T) {
	usd := 1000.0
	store := &fakeStore{
		alerts:  []model.Alert{{ID: 1, SubscriberID: "sub-a", Name: "rolex", Query: "brand=rolex&currency=USD"}},
		matches: map[int64][]model.Listing{1: {{ID: 1, Title: "x", PriceUSD: &usd}}},
		checked: map[int64]bool{},
		since:   map[int64]time.Time{},
	}
	n := &Notifier{Store: store, Push: &fakeSender{fail: true}, Conv: conv, Log: zerolog.Nop()}
	res, err := n.Run(context.Background())
	if err != nil || res.Failed != 1 || res.Notified != 0 {
		t.Errorf("result %+v %v", res, err)
	}
	if _, ok := store.checked[1]; ok {
		t.Error("watermark advanced although the push failed; the finds would be lost")
	}
}
