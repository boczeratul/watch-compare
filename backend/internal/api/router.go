// Package api exposes the read-only HTTP API consumed by the Next.js frontend.
package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"

	"github.com/hsuanlee/watch-compare/backend/internal/config"
	"github.com/hsuanlee/watch-compare/backend/internal/repository"
)

// Server holds handler dependencies.
type Server struct {
	repo  *repository.Repo
	cfg   *config.Config
	log   zerolog.Logger
	rates *ratesCache
}

// NewRouter builds the chi router with middleware and routes.
func NewRouter(repo *repository.Repo, cfg *config.Config, logger zerolog.Logger) http.Handler {
	s := &Server{repo: repo, cfg: cfg, log: logger, rates: newRatesCache(repo, 10*time.Minute)}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(requestLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(middleware.Compress(5))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           600,
	}))

	r.Get("/healthz", s.health)
	r.Get("/readyz", s.ready)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(cacheControl(cfg.CacheTTL))
		r.Get("/listings", s.searchListings)
		r.Get("/listings/{id}", s.getListing)
		r.Get("/listings/{id}/similar", s.similarListings)
		r.Get("/listings/{id}/price-history", s.priceHistory)
		r.Get("/brands", s.brands)
		r.Get("/sources", s.sources)
		r.Get("/rates", s.getRates)
		r.Get("/stats", s.stats)
		r.Get("/crawls", s.crawls)
	})
	return r
}

func cacheControl(ttl time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "public, max-age="+itoa(int(ttl.Seconds()))+", stale-while-revalidate=300")
			next.ServeHTTP(w, r)
		})
	}
}

func requestLogger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			logger.Info().
				Str("method", r.Method).Str("path", r.URL.Path).Str("query", r.URL.RawQuery).
				Int("status", ww.Status()).Int("bytes", ww.BytesWritten()).
				Dur("latency", time.Since(start)).Str("reqId", middleware.GetReqID(r.Context())).
				Msg("request")
		})
	}
}
