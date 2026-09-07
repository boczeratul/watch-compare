package crawler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/hsuanlee/watch-compare/backend/internal/config"
	"github.com/hsuanlee/watch-compare/backend/internal/fx"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
	"github.com/hsuanlee/watch-compare/backend/internal/repository"
)

// Runner executes sources and persists what they emit.
type Runner struct {
	repo    *repository.Repo
	fetcher *Fetcher
	cfg     *config.Config
	conv    *fx.Converter
	sources map[string]Source
	log     zerolog.Logger

	brandMu    sync.Mutex
	brandCache map[string]int
}

// NewRunner wires the runner.
func NewRunner(repo *repository.Repo, fetcher *Fetcher, cfg *config.Config, conv *fx.Converter, logger zerolog.Logger, sources ...Source) *Runner {
	r := &Runner{repo: repo, fetcher: fetcher, cfg: cfg, conv: conv, sources: map[string]Source{}, log: logger, brandCache: map[string]int{}}
	for _, s := range sources {
		r.sources[s.Key()] = s
	}
	return r
}

// Stats summarizes a run.
type Stats struct {
	Seen, New, Updated, PriceChanged, Deactivated, Errors int
}

// Run crawls the requested sources (all enabled when keys is empty) with bounded concurrency.
func (r *Runner) Run(ctx context.Context, keys []string) error {
	dbSources, err := r.repo.Sources(ctx)
	if err != nil {
		return err
	}
	var todo []model.Source
	for _, s := range dbSources {
		if !s.Enabled && len(keys) == 0 {
			continue
		}
		if len(keys) > 0 && !contains(keys, s.Key) {
			continue
		}
		if _, ok := r.sources[s.Key]; !ok {
			r.log.Warn().Str("source", s.Key).Msg("no adapter registered, skipping")
			continue
		}
		todo = append(todo, s)
	}
	if len(todo) == 0 {
		return errors.New("no sources to crawl")
	}

	sem := make(chan struct{}, max(1, r.cfg.CrawlConcurrency))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	for _, s := range todo {
		wg.Add(1)
		go func(s model.Source) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if err := r.runOne(ctx, s); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", s.Key, err))
				mu.Unlock()
			}
		}(s)
	}
	wg.Wait()
	if err := r.repo.RefreshBrandCounts(ctx); err != nil {
		r.log.Error().Err(err).Msg("refresh brand counts")
	}
	return errors.Join(errs...)
}

func (r *Runner) runOne(ctx context.Context, src model.Source) (err error) {
	logger := r.log.With().Str("source", src.Key).Logger()
	adapter := r.sources[src.Key]
	env := &Env{Fetcher: r.fetcher, Cfg: r.cfg, Log: logger, MaxPages: r.cfg.CrawlMaxPages}
	started := time.Now()

	var runID int64
	if !r.cfg.CrawlDryRun {
		runID, err = r.repo.StartCrawlRun(ctx, src.ID)
		if err != nil {
			return err
		}
	}
	var st Stats
	emit := func(l model.Listing) error {
		st.Seen++
		l.SourceID = src.ID
		l.SourceKey = src.Key
		if l.Currency == "" {
			l.Currency = src.Currency
		}
		if l.LocationCountry == "" {
			l.LocationCountry = src.Country
		}
		EnrichFromTitle(&l)
		if l.Price != nil {
			if usd, ok := r.conv.ToUSD(*l.Price, l.Currency); ok {
				l.PriceUSD = &usd
			}
		}
		if r.cfg.CrawlDryRun {
			logger.Info().Str("id", l.ExternalID).Str("brand", l.BrandSlug).Str("ref", l.ReferenceNumber).
				Interface("price", l.Price).Str("cur", l.Currency).Interface("usd", l.PriceUSD).Str("title", truncate(l.Title, 80)).Msg("listing")
			if st.Seen >= 25 {
				return ErrStop
			}
			return nil
		}
		if l.BrandSlug != "" {
			id, err := r.ensureBrand(ctx, l.BrandSlug, l.BrandName)
			if err != nil {
				return err
			}
			l.BrandID = &id
		}
		res, err := r.repo.UpsertListing(ctx, &l)
		if err != nil {
			st.Errors++
			logger.Error().Err(err).Str("id", l.ExternalID).Msg("upsert failed")
			return nil // keep going
		}
		switch {
		case res.IsNew:
			st.New++
		default:
			st.Updated++
		}
		if res.PriceChanged {
			st.PriceChanged++
		}
		return nil
	}

	crawlErr := adapter.Crawl(ctx, env, emit)
	if errors.Is(crawlErr, ErrStop) {
		crawlErr = nil
	}
	status := "ok"
	if crawlErr != nil {
		status = "failed"
		if st.Seen > 0 {
			status = "partial"
		}
	}
	// Only deactivate when the crawl completed, so a blocked day does not empty the index.
	if !r.cfg.CrawlDryRun && crawlErr == nil && st.Seen > 0 {
		n, derr := r.repo.DeactivateStale(ctx, src.ID, time.Now().Add(-r.cfg.CrawlDeactivateStaleAfter))
		if derr != nil {
			logger.Error().Err(derr).Msg("deactivate stale")
		}
		st.Deactivated = n
	}
	logger.Info().Str("status", status).Int("seen", st.Seen).Int("new", st.New).Int("updated", st.Updated).
		Int("priceChanged", st.PriceChanged).Int("deactivated", st.Deactivated).Int("errors", st.Errors).
		Dur("took", time.Since(started)).Msg("crawl finished")
	if !r.cfg.CrawlDryRun {
		if ferr := r.repo.FinishCrawlRun(ctx, runID, status, st.Seen, st.New, st.Updated, st.Deactivated, crawlErr); ferr != nil {
			logger.Error().Err(ferr).Msg("finish crawl run")
		}
	}
	return crawlErr
}

func (r *Runner) ensureBrand(ctx context.Context, slug, name string) (int, error) {
	r.brandMu.Lock()
	defer r.brandMu.Unlock()
	if id, ok := r.brandCache[slug]; ok {
		return id, nil
	}
	if name == "" {
		if b, ok := normalize.BrandBySlug(slug); ok {
			name = b.Name
		} else {
			name = strings.Title(strings.ReplaceAll(slug, "-", " ")) //nolint:staticcheck
		}
	}
	id, err := r.repo.EnsureBrand(ctx, slug, name)
	if err != nil {
		return 0, err
	}
	r.brandCache[slug] = id
	return id, nil
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
