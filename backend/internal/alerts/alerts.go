// Package alerts matches saved searches against newly crawled listings and sends one push per
// alert with its new finds. It runs at the end of the crawler job.
package alerts

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/hsuanlee/watch-compare/backend/internal/fx"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/push"
	"github.com/hsuanlee/watch-compare/backend/internal/search"
)

// MaxPerSubscriber caps how many alerts one device can save.
const MaxPerSubscriber = 20

var subscriberRE = regexp.MustCompile(`^[A-Za-z0-9-]{32,64}$`)

// ValidSubscriberID reports whether id looks like a client-generated random id (a UUID).
func ValidSubscriberID(id string) bool { return subscriberRE.MatchString(id) }

// ErrNoCriteria rejects an alert that would match every new listing.
var ErrNoCriteria = errors.New("an alert needs a search term or at least one filter")

// Normalize validates a raw query string for an alert and returns its canonical encoding
// (criteria only, stable key order, currency included). It rejects a query with no criteria,
// which would match every new listing.
func Normalize(raw string, conv *fx.Converter) (string, error) {
	qv, err := url.ParseQuery(strings.TrimPrefix(raw, "?"))
	if err != nil {
		return "", errors.New("invalid query")
	}
	crit := search.Criteria(qv)
	if len(crit) == 0 {
		return "", ErrNoCriteria
	}
	if _, err := search.Parse(crit, conv); err != nil {
		return "", err
	}
	return crit.Encode(), nil
}

// MaxNameLen caps an alert name, in characters.
const MaxNameLen = 80

// Name returns the display name for an alert on query: the user's name with whitespace collapsed
// and cut to MaxNameLen characters, or DefaultName(query) when the user gave none.
func Name(userName, query string) string {
	name := strings.Join(strings.Fields(userName), " ")
	if name == "" {
		return DefaultName(query)
	}
	return truncate(name)
}

func truncate(name string) string {
	if r := []rune(name); len(r) > MaxNameLen {
		return string(r[:MaxNameLen-1]) + "…"
	}
	return name
}

// DefaultName summarizes a normalized query for display: search text, brands, model, reference
// and price range, e.g. "rolex · 116610LN · ≤ 300,000 TWD".
func DefaultName(query string) string {
	qv, _ := url.ParseQuery(query)
	var parts []string
	for _, k := range []string{"q", "brand", "model", "ref"} {
		if v := strings.TrimSpace(qv.Get(k)); v != "" {
			parts = append(parts, strings.ReplaceAll(v, ",", ", "))
		}
	}
	cur := search.Currency(qv)
	lo, hi := qv.Get("price_min"), qv.Get("price_max")
	switch {
	case lo != "" && hi != "":
		parts = append(parts, fmt.Sprintf("%s–%s %s", groupDigits(lo), groupDigits(hi), cur))
	case hi != "":
		parts = append(parts, fmt.Sprintf("≤ %s %s", groupDigits(hi), cur))
	case lo != "":
		parts = append(parts, fmt.Sprintf("≥ %s %s", groupDigits(lo), cur))
	}
	if len(parts) == 0 {
		return "Saved search"
	}
	return truncate(strings.Join(parts, " · "))
}

// Store is the slice of the repository the notifier needs.
type Store interface {
	AllAlerts(ctx context.Context) ([]model.Alert, error)
	NewMatches(ctx context.Context, q model.ListingQuery, limit int) (int, []model.Listing, error)
	MarkAlertChecked(ctx context.Context, id int64, at time.Time, notified bool) error
}

// Notifier sends new-find notifications.
type Notifier struct {
	Store   Store
	Push    push.Sender
	Conv    *fx.Converter
	SiteURL string // public web origin for notification links, e.g. https://watch-compare.vercel.app
	Log     zerolog.Logger
	Now     func() time.Time
}

// Result summarizes one run.
type Result struct {
	Alerts, Notified, Failed int
}

// Run checks every alert once. For each, listings first seen after its watermark that match its
// criteria are the new finds; one notification carries them all. The watermark only advances
// after a successful send (or when there was nothing to send), so a OneSignal outage retries the
// same finds on the next run instead of dropping them.
func (n *Notifier) Run(ctx context.Context) (Result, error) {
	var res Result
	now := time.Now
	if n.Now != nil {
		now = n.Now
	}
	// Listings upserted from here on belong to the next run.
	runAt := now()
	all, err := n.Store.AllAlerts(ctx)
	if err != nil {
		return res, err
	}
	res.Alerts = len(all)
	for _, a := range all {
		if ctx.Err() != nil {
			return res, ctx.Err()
		}
		sent, err := n.check(ctx, a, runAt)
		if err != nil {
			res.Failed++
			n.Log.Warn().Err(err).Int64("alert", a.ID).Msg("alert check failed")
			continue
		}
		if sent {
			res.Notified++
		}
	}
	return res, nil
}

func (n *Notifier) check(ctx context.Context, a model.Alert, runAt time.Time) (bool, error) {
	qv, err := url.ParseQuery(a.Query)
	if err != nil {
		return false, err
	}
	q, err := search.Parse(qv, n.Conv)
	if err != nil {
		return false, err
	}
	since := a.CheckedAt
	q.FirstSeenAfter = &since
	total, items, err := n.Store.NewMatches(ctx, q, 1)
	if err != nil {
		return false, err
	}
	if total == 0 {
		return false, n.Store.MarkAlertChecked(ctx, a.ID, runAt, false)
	}
	if err := n.Push.Send(ctx, n.message(a, qv, total, items[0])); err != nil {
		return false, err
	}
	return true, n.Store.MarkAlertChecked(ctx, a.ID, runAt, true)
}

// message builds the notification: the single new listing, or the count plus the cheapest one.
func (n *Notifier) message(a model.Alert, qv url.Values, total int, cheapest model.Listing) push.Notification {
	price := n.formatPrice(cheapest, search.Currency(qv))
	site := strings.TrimRight(n.SiteURL, "/")
	msg := push.Notification{
		ExternalIDs: []string{a.SubscriberID},
		Headings:    map[string]string{},
		Contents:    map[string]string{},
		Data:        map[string]any{"alertId": a.ID, "query": a.Query},
		CollapseID:  "alert-" + strconv.FormatInt(a.ID, 10),
	}
	if total == 1 {
		line := cheapest.Title
		if price != "" {
			line += " · " + price
		}
		line += " · " + cheapest.SourceName
		for lang, t := range texts {
			msg.Headings[lang] = fmt.Sprintf(t.one, a.Name)
			msg.Contents[lang] = line
		}
		msg.Data["listingId"] = cheapest.ID
		if site != "" {
			msg.WebURL = site + "/listing/" + strconv.FormatInt(cheapest.ID, 10)
		}
		return msg
	}
	for lang, t := range texts {
		msg.Headings[lang] = fmt.Sprintf(t.many, total, a.Name)
		if price != "" {
			msg.Contents[lang] = fmt.Sprintf(t.from, price, cheapest.SourceName)
		} else {
			msg.Contents[lang] = t.tap
		}
	}
	if site != "" {
		msg.WebURL = site + "/search?" + a.Query + "&sort=newest"
	}
	return msg
}

// Notification copy per OneSignal language code. "en" is the fallback for other languages.
var texts = map[string]struct{ one, many, from, tap string }{
	"en":      {one: "New match: %s", many: "%d new matches: %s", from: "From %s at %s. Tap to see them all.", tap: "Tap to see them all."},
	"zh-Hant": {one: "新符合物件：%s", many: "%d 件新符合物件：%s", from: "最低 %s（%s）。點擊查看全部。", tap: "點擊查看全部。"},
	"zh-Hans": {one: "新匹配商品：%s", many: "%d 件新匹配商品：%s", from: "最低 %s（%s）。点击查看全部。", tap: "点击查看全部。"},
	"ja":      {one: "新着一致：%s", many: "新着 %d 件：%s", from: "最安 %s（%s）。タップしてすべて表示。", tap: "タップしてすべて表示。"},
	"de":      {one: "Neuer Treffer: %s", many: "%d neue Treffer: %s", from: "Ab %s bei %s. Tippen, um alle zu sehen.", tap: "Tippen, um alle zu sehen."},
}

// formatPrice shows the listing's price in the alert's currency when it can be converted, else as
// listed. Amounts are whole units with thousands separators ("300,000 TWD").
func (n *Notifier) formatPrice(l model.Listing, currency string) string {
	if l.PriceUSD != nil && n.Conv != nil {
		if v, ok := n.Conv.FromUSD(*l.PriceUSD, currency); ok {
			return groupDigits(strconv.FormatFloat(math.Round(v), 'f', 0, 64)) + " " + currency
		}
	}
	if l.Price != nil && l.Currency != "" {
		return groupDigits(strconv.FormatFloat(math.Round(*l.Price), 'f', 0, 64)) + " " + l.Currency
	}
	return ""
}

// groupDigits inserts thousands separators into the integer part of a decimal string.
func groupDigits(s string) string {
	intPart, frac, hasFrac := strings.Cut(s, ".")
	neg := strings.HasPrefix(intPart, "-")
	intPart = strings.TrimPrefix(intPart, "-")
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	out := b.String()
	if neg {
		out = "-" + out
	}
	if hasFrac {
		out += "." + frac
	}
	return out
}
