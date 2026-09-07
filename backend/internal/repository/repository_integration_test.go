package repository_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"

	"github.com/hsuanlee/watch-compare/backend/internal/db"
	"github.com/hsuanlee/watch-compare/backend/internal/model"
	"github.com/hsuanlee/watch-compare/backend/internal/repository"
)

// Runs a real PostgreSQL via embedded-postgres. Enable with TEST_EMBEDDED_PG=1
// (downloads a ~30MB binary on first run). Also honours TEST_DATABASE_URL to reuse a DB.
func TestRepositoryEndToEnd(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		if os.Getenv("TEST_EMBEDDED_PG") == "" {
			t.Skip("set TEST_EMBEDDED_PG=1 or TEST_DATABASE_URL to run")
		}
		cacheDir := os.Getenv("TEST_EMBEDDED_PG_CACHE")
		port := uint32(55432)
		if p := os.Getenv("TEST_EMBEDDED_PG_PORT"); p != "" {
			var n uint32
			if _, err := fmt.Sscanf(p, "%d", &n); err == nil {
				port = n
			}
		}
		cfg := embeddedpostgres.DefaultConfig().Version(embeddedpostgres.V16).Port(port).
			Username("watch").Password("watch").Database("watch").StartTimeout(60 * time.Second)
		if cacheDir != "" {
			suffix := fmt.Sprintf("-%d", port)
			cfg = cfg.CachePath(cacheDir).RuntimePath(cacheDir + "/runtime" + suffix).DataPath(cacheDir + "/data" + suffix).BinariesPath(cacheDir + "/bin")
		}
		pg := embeddedpostgres.NewDatabase(cfg)
		if err := pg.Start(); err != nil {
			t.Fatalf("start embedded postgres: %v", err)
		}
		t.Cleanup(func() { _ = pg.Stop() })
		dsn = fmt.Sprintf("postgres://watch:watch@localhost:%d/watch?sslmode=disable", port)
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if os.Getenv("TEST_DATABASE_URL") == "" {
		// embedded data dirs are reused between runs: start from an empty schema
		if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	// migrations must be idempotent and safe to run concurrently (API + job booting together);
	// a leaked advisory lock would make this hang.
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	{
		cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		errs := make(chan error, 4)
		for i := 0; i < 4; i++ {
			go func() {
				p2, err := db.Connect(cctx, dsn)
				if err != nil {
					errs <- err
					return
				}
				defer p2.Close()
				errs <- db.Migrate(cctx, p2)
			}()
		}
		for i := 0; i < 4; i++ {
			if err := <-errs; err != nil {
				t.Fatalf("concurrent migrate: %v", err)
			}
		}
		// rollback + re-apply the latest migration through the same lock path
		if err := db.Rollback(ctx, pool); err != nil {
			t.Fatal(err)
		}
		if err := db.Migrate(ctx, pool); err != nil {
			t.Fatal(err)
		}
	}
	repo := repository.New(pool)

	src, err := repo.SourceByKey(ctx, "jackroad")
	if err != nil {
		t.Fatal(err)
	}
	rolex, err := repo.EnsureBrand(ctx, "rolex", "Rolex")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertRates(ctx, map[string]float64{"USD": 1, "JPY": 150, "TWD": 32, "EUR": 0.9}); err != nil {
		t.Fatal(err)
	}

	price := 1500000.0
	usd := 10000.0
	l := model.Listing{
		SourceID: src.ID, ExternalID: "abc1", URL: "https://example.com/1", Title: "Rolex Submariner 116610LN",
		BrandID: &rolex, BrandName: "Rolex", Model: "Submariner", ReferenceNumber: "116610LN",
		Condition: model.ConditionVeryGood, Movement: model.MovementAutomatic, Gender: model.GenderMen,
		Price: &price, Currency: "JPY", PriceUSD: &usd, ImageURLs: []string{"https://example.com/1.jpg"}, DialColor: "black",
	}
	res, err := repo.UpsertListing(ctx, &l)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if !res.IsNew {
		t.Error("expected new")
	}
	// same price again → no history row, not new
	res, err = repo.UpsertListing(ctx, &l)
	if err != nil || res.IsNew || res.PriceChanged {
		t.Errorf("second upsert: %+v %v", res, err)
	}
	// price drop → history row
	price2, usd2 := 1400000.0, 9333.0
	l.Price, l.PriceUSD = &price2, &usd2
	res, err = repo.UpsertListing(ctx, &l)
	if err != nil || !res.PriceChanged {
		t.Errorf("third upsert: %+v %v", res, err)
	}
	hist, err := repo.PriceHistory(ctx, res.ID)
	if err != nil || len(hist) != 2 {
		t.Errorf("history: %v %v", hist, err)
	}

	// second listing, other source, same reference, cheaper
	hs, _ := repo.SourceByKey(ctx, "hourstack")
	p3, u3 := 280000.0, 8750.0
	l2 := model.Listing{SourceID: hs.ID, ExternalID: "x9", URL: "https://example.com/2", Title: "勞力士 Submariner 116610LN 黑水鬼",
		BrandID: &rolex, BrandName: "Rolex", ReferenceNumber: "116610LN", Condition: model.ConditionGood,
		Movement: model.MovementAutomatic, Gender: model.GenderMen, Price: &p3, Currency: "TWD", PriceUSD: &u3, ImageURLs: []string{}}
	if _, err := repo.UpsertListing(ctx, &l2); err != nil {
		t.Fatal(err)
	}
	if err := repo.RefreshBrandCounts(ctx); err != nil {
		t.Fatal(err)
	}

	// search: text + sort + facets
	sr, err := repo.SearchListings(ctx, model.ListingQuery{Text: "submariner", Sort: model.SortPriceAsc, Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if sr.Total != 2 || len(sr.Items) != 2 || sr.Items[0].SourceKey != "hourstack" {
		t.Errorf("search result: total=%d items=%d first=%s", sr.Total, len(sr.Items), sr.Items[0].SourceKey)
	}
	if len(sr.Facets.Brands) != 1 || sr.Facets.Brands[0].Key != "rolex" || sr.Facets.Brands[0].Count != 2 {
		t.Errorf("brand facet: %+v", sr.Facets.Brands)
	}
	if len(sr.Facets.Sources) != 2 {
		t.Errorf("source facet: %+v", sr.Facets.Sources)
	}
	// relevance sort with a canonicalized alternative (regression: rank arg must not leak into count/facets)
	sr, err = repo.SearchListings(ctx, model.ListingQuery{Text: "水鬼", TextAlt: "Submariner", Sort: model.SortRelevance})
	if err != nil || sr.Total != 2 {
		t.Errorf("relevance search: total=%d err=%v", sr.Total, err)
	}
	// dial colour filter + facets
	sr, err = repo.SearchListings(ctx, model.ListingQuery{DialColors: []string{"black"}})
	if err != nil || sr.Total != 1 || sr.Items[0].DialColor != "black" {
		t.Errorf("dial filter: total=%d err=%v", sr.Total, err)
	}
	sr, _ = repo.SearchListings(ctx, model.ListingQuery{})
	if len(sr.Facets.DialColors) != 1 || sr.Facets.DialColors[0].Key != "black" {
		t.Errorf("dial facet: %+v", sr.Facets.DialColors)
	}
	// price filter in USD
	minP := 9000.0
	sr, err = repo.SearchListings(ctx, model.ListingQuery{PriceMinUSD: &minP, Sort: model.SortNewest})
	if err != nil || sr.Total != 1 {
		t.Errorf("price filter: total=%d err=%v", sr.Total, err)
	}
	// reference prefix + brand filter
	sr, err = repo.SearchListings(ctx, model.ListingQuery{Brands: []string{"rolex"}, Reference: "1166", Conditions: []model.Condition{model.ConditionGood}})
	if err != nil || sr.Total != 1 {
		t.Errorf("ref filter: total=%d err=%v", sr.Total, err)
	}
	// similar
	got, err := repo.GetListing(ctx, res.ID)
	if err != nil {
		t.Fatal(err)
	}
	sim, err := repo.SimilarListings(ctx, got, 10)
	if err != nil || len(sim) != 1 || sim[0].SourceKey != "hourstack" {
		t.Errorf("similar: %v %v", sim, err)
	}
	// deactivate stale
	n, err := repo.DeactivateStale(ctx, src.ID, time.Now().Add(time.Hour))
	if err != nil || n != 1 {
		t.Errorf("deactivate: %d %v", n, err)
	}
	brands, _ := repo.Brands(ctx)
	if len(brands) != 1 || brands[0].ListingCount != 2 {
		t.Errorf("brands: %+v", brands)
	}
	stats, err := repo.Stats(ctx)
	if err != nil || stats["activeListings"].(int) != 1 {
		t.Errorf("stats: %v %v", stats, err)
	}
	runID, _ := repo.StartCrawlRun(ctx, src.ID)
	if err := repo.FinishCrawlRun(ctx, runID, "ok", 1, 1, 0, 0, nil); err != nil {
		t.Fatal(err)
	}
	runs, _ := repo.RecentCrawlRuns(ctx, 5)
	if len(runs) != 1 || runs[0].Status != "ok" {
		t.Errorf("runs: %+v", runs)
	}
}
