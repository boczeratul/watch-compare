package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hsuanlee/watch-compare/backend/internal/fx"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
	"github.com/hsuanlee/watch-compare/backend/internal/repository"
)

// ---------- helpers ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

func itoa(i int) string { return strconv.Itoa(i) }

// ratesCache memoizes the FX table so search requests do not hit the DB for it.
type ratesCache struct {
	repo *repository.Repo
	ttl  time.Duration
	mu   sync.Mutex
	at   time.Time
	conv *fx.Converter
	list []model.ExchangeRate
}

func newRatesCache(repo *repository.Repo, ttl time.Duration) *ratesCache {
	return &ratesCache{repo: repo, ttl: ttl}
}

func (c *ratesCache) get(ctx context.Context) (*fx.Converter, []model.ExchangeRate, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conv != nil && time.Since(c.at) < c.ttl {
		return c.conv, c.list, nil
	}
	list, err := c.repo.Rates(ctx)
	if err != nil {
		return nil, nil, err
	}
	m := make(map[string]float64, len(list))
	for _, e := range list {
		m[e.Quote] = e.Rate
	}
	c.conv, c.list, c.at = &fx.Converter{Rates: m}, list, time.Now()
	return c.conv, c.list, nil
}

// ---------- health ----------

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := s.repo.Pool().Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unreachable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// ---------- listings ----------

func parseQuery(r *http.Request, conv *fx.Converter) (model.ListingQuery, error) {
	qv := r.URL.Query()
	q := model.ListingQuery{
		Text:       strings.TrimSpace(qv.Get("q")),
		Brands:     csv(qv.Get("brand")),
		Model:      strings.TrimSpace(qv.Get("model")),
		Reference:  strings.TrimSpace(qv.Get("ref")),
		Sources:    csv(qv.Get("source")),
		Countries:  upperAll(csv(qv.Get("country"))),
		DialColors: csv(qv.Get("dial")),
		Sort:       model.SortKey(qv.Get("sort")),
		Page:       atoiDefault(qv.Get("page"), 1),
		PerPage:    atoiDefault(qv.Get("per_page"), 30),
	}
	if q.Text != "" {
		q.TextAlt = normalize.CanonicalizeQuery(q.Text)
	}
	for _, c := range csv(qv.Get("condition")) {
		q.Conditions = append(q.Conditions, model.Condition(c))
	}
	for _, m := range csv(qv.Get("movement")) {
		q.Movements = append(q.Movements, model.Movement(m))
	}
	for _, g := range csv(qv.Get("gender")) {
		q.Genders = append(q.Genders, model.Gender(g))
	}
	currency := strings.ToUpper(strings.TrimSpace(qv.Get("currency")))
	if currency == "" {
		currency = "USD"
	}
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

func (s *Server) searchListings(w http.ResponseWriter, r *http.Request) {
	conv, _, err := s.rates.get(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	q, err := parseQuery(r, conv)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	res, err := s.repo.SearchListings(r.Context(), q)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) getListing(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	l, err := s.repo.GetListing(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "listing not found")
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, l)
}

func (s *Server) similarListings(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	l, err := s.repo.GetListing(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusNotFound, "listing not found")
		return
	}
	if err != nil {
		s.fail(w, err)
		return
	}
	items, err := s.repo.SimilarListings(r.Context(), l, 20)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) priceHistory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	pts, err := s.repo.PriceHistory(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": pts})
}

// ---------- reference data ----------

func (s *Server) brands(w http.ResponseWriter, r *http.Request) {
	b, err := s.repo.Brands(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	if b == nil {
		b = []model.Brand{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": b})
}

func (s *Server) sources(w http.ResponseWriter, r *http.Request) {
	src, err := s.repo.Sources(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": src})
}

func (s *Server) getRates(w http.ResponseWriter, r *http.Request) {
	_, list, err := s.rates.get(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"base": "USD", "supported": fx.SupportedCurrencies, "items": list})
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	st, err := s.repo.Stats(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) crawls(w http.ResponseWriter, r *http.Request) {
	runs, err := s.repo.RecentCrawlRuns(r.Context(), 50)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": runs})
}

func (s *Server) fail(w http.ResponseWriter, err error) {
	s.log.Error().Err(err).Msg("handler error")
	writeError(w, http.StatusInternalServerError, "internal error")
}

// ---------- parsing ----------

func csv(s string) []string {
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
