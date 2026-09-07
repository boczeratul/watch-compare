package crawler

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/hsuanlee/watch-compare/backend/internal/config"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/normalize"
)

// Env is what every Source receives when crawling.
type Env struct {
	Fetcher  *Fetcher
	Cfg      *config.Config
	Log      zerolog.Logger
	MaxPages int
}

// Emit is called once per discovered listing. Returning ErrStop ends the crawl gracefully.
type Emit func(model.Listing) error

// Source is a marketplace adapter.
type Source interface {
	// Key matches sources.key in the database.
	Key() string
	// Crawl walks the marketplace and emits listings. It must be resumable and idempotent.
	Crawl(ctx context.Context, env *Env, emit Emit) error
}

// Preflighter is implemented by sources that need external configuration (API keys, a proxy).
// A non-nil error means "not configured": the Runner skips the source with a warning and records
// a "skipped" crawl run instead of failing the whole job.
type Preflighter interface {
	Preflight(cfg *config.Config) error
}

// EnrichFromTitle fills brand/reference/diameter/year/box-papers heuristically when the
// marketplace did not provide structured data.
func EnrichFromTitle(l *model.Listing, extraBrandHints ...string) {
	if l.BrandSlug == "" {
		if b := normalize.DetectBrand(append([]string{l.Title}, extraBrandHints...)...); b != nil {
			l.BrandSlug, l.BrandName = b.Slug, b.Name
		}
	}
	if l.ReferenceNumber == "" {
		l.ReferenceNumber = normalize.ExtractReference(l.Title)
	}
	// Canonical English model so "Submariner" matches "サブマリーナ" and "水鬼". The dealer's own
	// wording stays in the title. Only applied when the brand is known, so a DOXA "Sub" is not
	// labelled as a Rolex Submariner.
	if l.BrandSlug != "" {
		if m := normalize.DetectModel(l.Model, l.Title); m != "" {
			l.Model = m
		}
	}
	if l.CaseDiameterMM == nil {
		l.CaseDiameterMM = normalize.ParseDiameter(l.Title)
	}
	if l.Year == nil {
		l.Year = normalize.ParseYear(l.Title)
	}
	if l.HasBox == nil && l.HasPapers == nil {
		l.HasBox, l.HasPapers = normalize.DetectBoxPapers(l.Title + " " + l.Description)
	}
	if l.Movement == "" {
		l.Movement = normalize.MovementType(l.Title)
	}
	if l.Gender == "" {
		l.Gender = normalize.GenderType(l.Title)
	}
	if l.Condition == "" {
		l.Condition = model.ConditionUnknown
	}
}
