// Package fx fetches and applies USD-based exchange rates.
package fx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// SupportedCurrencies are exposed to the frontend currency switcher.
var SupportedCurrencies = []string{"USD", "EUR", "GBP", "CHF", "JPY", "TWD", "HKD", "SGD", "CNY", "KRW", "AUD", "CAD"}

// Converter converts amounts to USD using a rate table (1 USD = rate QUOTE).
type Converter struct {
	Rates map[string]float64
}

// ToUSD converts amount in currency to USD.
func (c *Converter) ToUSD(amount float64, currency string) (float64, bool) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "USD" {
		return amount, true
	}
	r, ok := c.Rates[currency]
	if !ok || r == 0 {
		return 0, false
	}
	return round2(amount / r), true
}

// FromUSD converts a USD amount into currency.
func (c *Converter) FromUSD(usd float64, currency string) (float64, bool) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "USD" {
		return usd, true
	}
	r, ok := c.Rates[currency]
	if !ok {
		return 0, false
	}
	return round2(usd * r), true
}

func round2(v float64) float64 { return float64(int64(v*100+0.5)) / 100 }

// Fetch downloads the latest USD rate table. The default provider (open.er-api.com) needs
// no API key; the response shape {"rates": {"EUR": 0.92, ...}} is shared by several providers.
func Fetch(ctx context.Context, client *http.Client, providerURL string) (map[string]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, providerURL, nil)
	if err != nil {
		return nil, err
	}
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fx provider: http %d", resp.StatusCode)
	}
	var payload struct {
		Rates map[string]float64 `json:"rates"`
		Base  string             `json:"base_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.Base != "" && payload.Base != "USD" {
		return nil, fmt.Errorf("fx provider base is %s, expected USD", payload.Base)
	}
	if len(payload.Rates) == 0 {
		return nil, fmt.Errorf("fx provider returned no rates")
	}
	out := map[string]float64{"USD": 1}
	for _, c := range SupportedCurrencies {
		if r, ok := payload.Rates[c]; ok {
			out[c] = r
		}
	}
	return out, nil
}
