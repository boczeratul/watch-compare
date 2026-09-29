// Package search turns API query parameters into a model.ListingQuery. The listings endpoint and
// the alert notifier share it, so a saved alert matches exactly what the same search shows.
package search

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/hsuanlee/watch-compare/backend/internal/fx"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

// CriteriaParams are the parameters that narrow a search (everything except currency, sort and
// paging). An alert stores only these plus currency.
var CriteriaParams = []string{"q", "brand", "model", "ref", "source", "condition", "movement", "gender", "country", "dial",
	"price_min", "price_max", "year_min", "year_max", "diameter_min", "diameter_max", "box", "papers"}

// Parse validates query parameters. price_min/price_max are read in `currency` (default USD) and
// converted to USD with conv.
func Parse(qv url.Values, conv *fx.Converter) (model.ListingQuery, error) {
	q := model.ListingQuery{
		Text:       strings.TrimSpace(qv.Get("q")),
		Brands:     CSV(qv.Get("brand")),
		Model:      strings.TrimSpace(qv.Get("model")),
		Reference:  strings.TrimSpace(qv.Get("ref")),
		Sources:    CSV(qv.Get("source")),
		Countries:  upperAll(CSV(qv.Get("country"))),
		DialColors: CSV(qv.Get("dial")),
		Sort:       model.SortKey(qv.Get("sort")),
		Page:       atoiDefault(qv.Get("page"), 1),
		PerPage:    atoiDefault(qv.Get("per_page"), 30),
	}
	if q.Text != "" {
		q.TextAlt = normalize.CanonicalizeQuery(q.Text)
	}
	for _, c := range CSV(qv.Get("condition")) {
		q.Conditions = append(q.Conditions, model.Condition(c))
	}
	for _, m := range CSV(qv.Get("movement")) {
		q.Movements = append(q.Movements, model.Movement(m))
	}
	for _, g := range CSV(qv.Get("gender")) {
		q.Genders = append(q.Genders, model.Gender(g))
	}
	currency := Currency(qv)
	toUSD := func(key string) (*float64, error) {
		raw := strings.TrimSpace(qv.Get(key))
		if raw == "" {
			return nil, nil
		}
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil || v < 0 {
			return nil, errors.New("invalid " + key)
		}
		usd, ok := conv.ToUSD(v, currency)
		if !ok {
			return nil, errors.New("unsupported currency " + currency)
		}
		return &usd, nil
	}
	var err error
	if q.PriceMinUSD, err = toUSD("price_min"); err != nil {
		return q, err
	}
	if q.PriceMaxUSD, err = toUSD("price_max"); err != nil {
		return q, err
	}
	q.YearMin = atoiPtr(qv.Get("year_min"))
	q.YearMax = atoiPtr(qv.Get("year_max"))
	q.DiameterMin = atofPtr(qv.Get("diameter_min"))
	q.DiameterMax = atofPtr(qv.Get("diameter_max"))
	q.HasBox = boolPtr(qv.Get("box"))
	q.HasPapers = boolPtr(qv.Get("papers"))
	if q.Sort == "" {
		if q.Text != "" {
			q.Sort = model.SortRelevance
		} else {
			q.Sort = model.SortNewest
		}
	}
	return q, nil
}

// Currency is the upper-cased `currency` parameter, USD when absent.
func Currency(qv url.Values) string {
	c := strings.ToUpper(strings.TrimSpace(qv.Get("currency")))
	if c == "" {
		return "USD"
	}
	return c
}

// Criteria keeps only the non-empty CriteriaParams plus currency, in a stable encoding.
func Criteria(qv url.Values) url.Values {
	out := url.Values{}
	for _, k := range CriteriaParams {
		if v := strings.TrimSpace(qv.Get(k)); v != "" {
			out.Set(k, v)
		}
	}
	if len(out) > 0 {
		out.Set("currency", Currency(qv))
	}
	return out
}

// CSV splits a comma-separated parameter, dropping blanks.
func CSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func upperAll(xs []string) []string {
	for i := range xs {
		xs[i] = strings.ToUpper(xs[i])
	}
	return xs
}

func atoiDefault(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil && n > 0 {
		return n
	}
	return def
}

func atoiPtr(s string) *int {
	if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return &n
	}
	return nil
}

func atofPtr(s string) *float64 {
	if f, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
		return &f
	}
	return nil
}

func boolPtr(s string) *bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes":
		t := true
		return &t
	case "0", "false", "no":
		f := false
		return &f
	}
	return nil
}
