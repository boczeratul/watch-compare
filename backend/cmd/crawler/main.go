// Command crawler runs one crawl cycle. Deployed as a Cloud Run Job triggered by
// Cloud Scheduler at 02:00 Asia/Taipei daily; also usable locally.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/hsuanlee/watch-compare/backend/internal/config"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler/allu"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler/chrono24"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler/commitwatch"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler/ebay"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler/hourstack"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler/housekihiroba"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler/ishida"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler/jackroad"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler/lips"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler/rdwatch"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler/sevenhours"
	"github.com/hsuanlee/watch-compare/backend/internal/crawler/watchnian"
	"github.com/hsuanlee/watch-compare/backend/internal/db"
	"github.com/hsuanlee/watch-compare/backend/internal/fx"
	"github.com/hsuanlee/watch-compare/backend/internal/repository"
)

func main() {
	var (
		sourcesFlag = flag.String("sources", "", "comma-separated source keys (default: all enabled / CRAWL_SOURCES)")
		dryRun      = flag.Bool("dry-run", false, "print the first listings per source without writing")
		maxPages    = flag.Int("max-pages", 0, "override CRAWL_MAX_PAGES")
		skipFX      = flag.Bool("skip-fx", false, "do not refresh exchange rates")
		timeout     = flag.Duration("timeout", 5*time.Hour, "overall run timeout")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(2)
	}
	if *sourcesFlag != "" {
		cfg.CrawlSources = strings.Split(*sourcesFlag, ",")
	}
	if *dryRun {
		cfg.CrawlDryRun = true
	}
	if *maxPages > 0 {
		cfg.CrawlMaxPages = *maxPages
	}
	logger := newLogger(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("database")
	}
	defer pool.Close()
	// Production runs migrations once in a controlled Cloud Build step, so the job is deployed with
	// AUTO_MIGRATE=false. Locally (and in tests) the default keeps the schema current on start.
	if os.Getenv("AUTO_MIGRATE") != "false" {
		if err := db.Migrate(ctx, pool); err != nil {
			logger.Fatal().Err(err).Msg("migrate")
		}
	}
	repo := repository.New(pool)
	fetcher := crawler.NewFetcher(cfg)

	// 1. Refresh FX so price_usd is computed with today's rates.
	if !*skipFX {
		if rates, err := fx.Fetch(ctx, fetcher.Client(), cfg.FXProviderURL); err != nil {
			logger.Warn().Err(err).Msg("fx refresh failed; using stored rates")
		} else if !cfg.CrawlDryRun {
			if err := repo.UpsertRates(ctx, rates); err != nil {
				logger.Error().Err(err).Msg("store fx rates")
			} else {
				logger.Info().Int("currencies", len(rates)).Msg("fx rates refreshed")
			}
		}
	}
	rates, err := repo.RatesMap(ctx)
	if err != nil {
		logger.Fatal().Err(err).Msg("load fx rates")
	}
	conv := &fx.Converter{Rates: rates}

	// 2. Crawl.
	runner := crawler.NewRunner(repo, fetcher, cfg, conv, logger,
		hourstack.Source{FetchDetails: os.Getenv("ALAPOWER_FETCH_DETAILS") == "true" || os.Getenv("HOURSTACK_FETCH_DETAILS") == "true"},
		rdwatch.Source{FetchDetails: os.Getenv("ALAPOWER_FETCH_DETAILS") == "true"},
		jackroad.Source{},
		watchnian.Source{},
		commitwatch.Source{},
		sevenhours.Source{},
		lips.Source{},
		allu.Source{},
		housekihiroba.Source{},
		ishida.Source{},
		&ebay.Source{},
		chrono24.Source{},
	)
	if err := runner.Run(ctx, cfg.CrawlSources); err != nil {
		logger.Error().Err(err).Msg("crawl finished with errors")
		os.Exit(1)
	}
	logger.Info().Msg("crawl finished")
}

func newLogger(cfg *config.Config) zerolog.Logger {
	lvl, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.LevelFieldName = "severity"
	zerolog.LevelFieldMarshalFunc = func(l zerolog.Level) string { return strings.ToUpper(l.String()) }
	zerolog.MessageFieldName = "message"
	var l zerolog.Logger
	if cfg.Env == "local" {
		l = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Kitchen})
	} else {
		l = zerolog.New(os.Stdout)
	}
	l = l.Level(lvl).With().Timestamp().Str("job", "crawler").Logger()
	log.Logger = l
	return l
}
