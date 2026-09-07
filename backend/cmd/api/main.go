// Command api serves the read-only JSON API on Cloud Run.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/hsuanlee/watch-compare/backend/internal/api"
	"github.com/hsuanlee/watch-compare/backend/internal/config"
	"github.com/hsuanlee/watch-compare/backend/internal/db"
	"github.com/hsuanlee/watch-compare/backend/internal/repository"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(2)
	}
	logger := newLogger(cfg)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("database")
	}
	defer pool.Close()
	if os.Getenv("AUTO_MIGRATE") != "false" {
		if err := db.Migrate(ctx, pool); err != nil {
			logger.Fatal().Err(err).Msg("migrate")
		}
	}

	handler := api.NewRouter(repository.New(pool), cfg, logger)
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		logger.Info().Int("port", cfg.Port).Str("env", cfg.Env).Msg("api listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("listen")
		}
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("shutdown")
	}
	logger.Info().Msg("api stopped")
}

func newLogger(cfg *config.Config) zerolog.Logger {
	lvl, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.TimeFieldFormat = time.RFC3339Nano
	// Cloud Logging severity mapping
	zerolog.LevelFieldName = "severity"
	zerolog.LevelFieldMarshalFunc = func(l zerolog.Level) string {
		switch l {
		case zerolog.DebugLevel:
			return "DEBUG"
		case zerolog.WarnLevel:
			return "WARNING"
		case zerolog.ErrorLevel:
			return "ERROR"
		case zerolog.FatalLevel, zerolog.PanicLevel:
			return "CRITICAL"
		}
		return "INFO"
	}
	zerolog.MessageFieldName = "message"
	var l zerolog.Logger
	if cfg.Env == "local" {
		l = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Kitchen})
	} else {
		l = zerolog.New(os.Stdout)
	}
	l = l.Level(lvl).With().Timestamp().Logger()
	log.Logger = l
	return l
}
