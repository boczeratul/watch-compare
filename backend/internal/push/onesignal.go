// Package push sends notifications through OneSignal's REST API, which fans them out to APNs
// (iOS app) and Web Push (browsers). Devices are addressed by external_id: each client generates
// a random subscriber id, logs in to OneSignal with it and saves alerts under the same id.
package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultBaseURL is OneSignal's REST API root.
const DefaultBaseURL = "https://api.onesignal.com"

// Notification is one push to one or more subscribers. Headings and Contents are keyed by
// OneSignal language code ("en", "zh-Hant", "zh-Hans", "ja", "de"); "en" is required and is the
// fallback for every other device language.
type Notification struct {
	ExternalIDs []string
	Headings    map[string]string
	Contents    map[string]string
	WebURL      string         // opened when a browser notification is clicked
	Data        map[string]any // delivered to the app; the iOS client routes on it
	CollapseID  string         // a newer notification with the same id replaces an unread one
}

// Sender delivers notifications.
type Sender interface {
	Send(ctx context.Context, n Notification) error
}

// OneSignal is a Sender backed by the OneSignal REST API.
type OneSignal struct {
	AppID   string
	APIKey  string // REST API key (Secret Manager: ONESIGNAL_REST_API_KEY)
	BaseURL string
	HTTP    *http.Client
}

// NewOneSignal returns a client with sane defaults.
func NewOneSignal(appID, apiKey string) *OneSignal {
	return &OneSignal{AppID: appID, APIKey: apiKey, BaseURL: DefaultBaseURL, HTTP: &http.Client{Timeout: 20 * time.Second}}
}

type createRequest struct {
	AppID          string              `json:"app_id"`
	IncludeAliases map[string][]string `json:"include_aliases"`
	TargetChannel  string              `json:"target_channel"`
	Headings       map[string]string   `json:"headings"`
	Contents       map[string]string   `json:"contents"`
	WebURL         string              `json:"web_url,omitempty"`
	Data           map[string]any      `json:"data,omitempty"`
	CollapseID     string              `json:"collapse_id,omitempty"`
}

// Result is OneSignal's answer to a create call. ID is empty when nothing was queued; Errors then
// says why (e.g. {"invalid_aliases": ...} or ["All included players are not subscribed"]).
type Result struct {
	ID     string          `json:"id"`
	Errors json.RawMessage `json:"errors,omitempty"`
}

// Send creates a push notification. A request OneSignal accepts but cannot deliver to anyone
// (every targeted device unsubscribed) is not an error: there is nobody to retry for.
func (o *OneSignal) Send(ctx context.Context, n Notification) error {
	if len(n.ExternalIDs) == 0 {
		return nil
	}
	_, err := o.Create(ctx, n)
	return err
}

// Create is Send that also returns OneSignal's response, for diagnostics.
func (o *OneSignal) Create(ctx context.Context, n Notification) (Result, error) {
	var out Result
	body, err := json.Marshal(createRequest{
		AppID:          o.AppID,
		IncludeAliases: map[string][]string{"external_id": n.ExternalIDs},
		TargetChannel:  "push",
		Headings:       n.Headings,
		Contents:       n.Contents,
		WebURL:         n.WebURL,
		Data:           n.Data,
		CollapseID:     n.CollapseID,
	})
	if err != nil {
		return out, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.BaseURL+"/notifications?c=push", bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Key "+o.APIKey)
	res, err := o.HTTP.Do(req)
	if err != nil {
		return out, fmt.Errorf("onesignal: %w", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	if res.StatusCode/100 != 2 {
		return out, fmt.Errorf("onesignal: HTTP %d: %s", res.StatusCode, bytes.TrimSpace(raw))
	}
	_ = json.Unmarshal(raw, &out)
	return out, nil
}
