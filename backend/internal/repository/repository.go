// Package repository is the only place that speaks SQL.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

// Repo wraps a pgx pool.
type Repo struct {
	pool *pgxpool.Pool
}

// New creates a Repo.
func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// Pool exposes the underlying pool for health checks.
func (r *Repo) Pool() *pgxpool.Pool { return r.pool }

// ---------- sources ----------

// Sources lists all configured sources.
func (r *Repo) Sources(ctx context.Context) ([]model.Source, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, key, name, base_url, country, currency, enabled FROM sources ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Source{} // never nil: a JSON null would break array consumers
	for rows.Next() {
		var s model.Source
		if err := rows.Scan(&s.ID, &s.Key, &s.Name, &s.BaseURL, &s.Country, &s.Currency, &s.Enabled); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// SourceByKey fetches one source.
func (r *Repo) SourceByKey(ctx context.Context, key string) (model.Source, error) {
	var s model.Source
	err := r.pool.QueryRow(ctx, `SELECT id, key, name, base_url, country, currency, enabled FROM sources WHERE key=$1`, key).
		Scan(&s.ID, &s.Key, &s.Name, &s.BaseURL, &s.Country, &s.Currency, &s.Enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return s, ErrNotFound
	}
	return s, err
}

// ---------- brands ----------

// Brands lists brands with active listing counts.
func (r *Repo) Brands(ctx context.Context) ([]model.Brand, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, slug, name, listing_count FROM brands ORDER BY listing_count DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Brand{} // never nil: a JSON null would break array consumers
	for rows.Next() {
		var b model.Brand
		if err := rows.Scan(&b.ID, &b.Slug, &b.Name, &b.ListingCount); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// EnsureBrand inserts the brand if missing and returns its id.
func (r *Repo) EnsureBrand(ctx context.Context, slug, name string) (int, error) {
	var id int
	err := r.pool.QueryRow(ctx, `
		INSERT INTO brands (slug, name) VALUES ($1, $2)
		ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name
		RETURNING id`, slug, name).Scan(&id)
	return id, err
}

// RefreshBrandCounts recomputes denormalized brand counts.
func (r *Repo) RefreshBrandCounts(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE brands b SET listing_count = c.n
		FROM (SELECT brand_id, count(*) n FROM listings WHERE is_active AND brand_id IS NOT NULL GROUP BY brand_id) c
		WHERE c.brand_id = b.id`)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `UPDATE brands SET listing_count = 0 WHERE id NOT IN (SELECT DISTINCT brand_id FROM listings WHERE is_active AND brand_id IS NOT NULL)`)
	return err
}

// ---------- exchange rates ----------

// Rates returns all stored rates (1 USD = rate QUOTE).
func (r *Repo) Rates(ctx context.Context) ([]model.ExchangeRate, error) {
	rows, err := r.pool.Query(ctx, `SELECT quote, rate, fetched_at FROM exchange_rates ORDER BY quote`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.ExchangeRate{} // never nil: a JSON null would break array consumers
	for rows.Next() {
		var e model.ExchangeRate
		if err := rows.Scan(&e.Quote, &e.Rate, &e.FetchedAt); err != nil {
			return nil, err
		}
		e.Quote = strings.TrimSpace(e.Quote)
		out = append(out, e)
	}
	return out, rows.Err()
}

// RatesMap is Rates as a lookup table.
func (r *Repo) RatesMap(ctx context.Context) (map[string]float64, error) {
	rates, err := r.Rates(ctx)
	if err != nil {
		return nil, err
	}
	m := make(map[string]float64, len(rates))
	for _, e := range rates {
		m[e.Quote] = e.Rate
	}
	return m, nil
}

// UpsertRates stores a fresh rate table.
func (r *Repo) UpsertRates(ctx context.Context, rates map[string]float64) error {
	batch := &pgx.Batch{}
	for q, v := range rates {
		batch.Queue(`INSERT INTO exchange_rates (quote, rate, fetched_at) VALUES ($1, $2, now())
			ON CONFLICT (quote) DO UPDATE SET rate = EXCLUDED.rate, fetched_at = now()`, q, v)
	}
	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range rates {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

// ---------- listings ----------

const listingColumns = `
	l.id, s.key, s.name, l.external_id, l.url, l.title, l.brand_id, coalesce(b.slug,''), coalesce(l.brand_name,''),
	coalesce(l.model,''), coalesce(l.reference_number,''), l.condition, l.year, l.case_diameter_mm, coalesce(l.case_material,''), coalesce(l.dial_color,''),
	coalesce(l.movement,'unknown'), coalesce(l.gender,'unknown'), l.has_box, l.has_papers, l.price, coalesce(l.currency,''), l.price_usd, l.shipping_price,
	coalesce(l.location_country,''), coalesce(l.location_city,''), coalesce(l.seller_name,''), coalesce(l.seller_type,''),
	l.image_urls, coalesce(l.description,''), l.attributes, l.is_active, l.first_seen_at, l.last_seen_at`

const listingFrom = ` FROM listings l JOIN sources s ON s.id = l.source_id LEFT JOIN brands b ON b.id = l.brand_id `

func scanListing(row pgx.Row) (*model.Listing, error) {
	var l model.Listing
	var attrs []byte
	var cond, mov, gen string
	err := row.Scan(&l.ID, &l.SourceKey, &l.SourceName, &l.ExternalID, &l.URL, &l.Title, &l.BrandID, &l.BrandSlug, &l.BrandName,
		&l.Model, &l.ReferenceNumber, &cond, &l.Year, &l.CaseDiameterMM, &l.CaseMaterial, &l.DialColor,
		&mov, &gen, &l.HasBox, &l.HasPapers, &l.Price, &l.Currency, &l.PriceUSD, &l.ShippingPrice,
		&l.LocationCountry, &l.LocationCity, &l.SellerName, &l.SellerType,
		&l.ImageURLs, &l.Description, &attrs, &l.IsActive, &l.FirstSeenAt, &l.LastSeenAt)
	if err != nil {
		return nil, err
	}
	l.Condition, l.Movement, l.Gender = model.Condition(cond), model.Movement(mov), model.Gender(gen)
	l.Currency = strings.TrimSpace(l.Currency)
	l.LocationCountry = strings.TrimSpace(l.LocationCountry)
	if len(attrs) > 0 {
		l.Attributes = json.RawMessage(attrs)
	}
	if l.ImageURLs == nil {
		l.ImageURLs = []string{}
	}
	return &l, nil
}

// GetListing fetches one listing by id.
func (r *Repo) GetListing(ctx context.Context, id int64) (*model.Listing, error) {
	l, err := scanListing(r.pool.QueryRow(ctx, `SELECT `+listingColumns+listingFrom+` WHERE l.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return l, err
}

// PriceHistory returns price observations for a listing, newest first.
func (r *Repo) PriceHistory(ctx context.Context, id int64) ([]model.PricePoint, error) {
	rows, err := r.pool.Query(ctx, `SELECT price, coalesce(currency,''), price_usd, observed_at FROM price_history WHERE listing_id=$1 ORDER BY observed_at DESC LIMIT 200`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.PricePoint{}
	for rows.Next() {
		var p model.PricePoint
		if err := rows.Scan(&p.Price, &p.Currency, &p.PriceUSD, &p.ObservedAt); err != nil {
			return nil, err
		}
		p.Currency = strings.TrimSpace(p.Currency)
		out = append(out, p)
	}
	return out, rows.Err()
}

// SimilarListings finds the same reference (preferred) or same brand+model on any source,
// cheapest first. This powers the "best deal across platforms" panel.
func (r *Repo) SimilarListings(ctx context.Context, l *model.Listing, limit int) ([]model.Listing, error) {
	var (
		where string
		args  []any
	)
	switch {
	case l.ReferenceNumber != "":
		where = `l.is_active AND l.id <> $1 AND upper(l.reference_number) = upper($2)`
		args = []any{l.ID, l.ReferenceNumber}
	case l.BrandID != nil && l.Model != "":
		where = `l.is_active AND l.id <> $1 AND l.brand_id = $2 AND l.model ILIKE $3`
		args = []any{l.ID, *l.BrandID, l.Model}
	default:
		return []model.Listing{}, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT `+listingColumns+listingFrom+` WHERE `+where+` ORDER BY l.price_usd ASC NULLS LAST LIMIT `+fmt.Sprint(limit), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collect(rows)
}

func collect(rows pgx.Rows) ([]model.Listing, error) {
	out := []model.Listing{}
	for rows.Next() {
		l, err := scanListing(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *l)
	}
	return out, rows.Err()
}

// SearchListings runs a filtered, sorted, paginated query plus facet counts.
func (r *Repo) SearchListings(ctx context.Context, q model.ListingQuery) (*model.SearchResult, error) {
	b := newBuilder()
	if !q.IncludeInactive {
		b.where("l.is_active")
	}
	if q.Text != "" {
		// The 'simple' parser drops CJK characters, so a Japanese/Chinese query also gets
		// (a) its canonicalized English form and (b) a trigram-indexed substring match on the title.
		conds := []string{"l.search_tsv @@ websearch_to_tsquery('simple', " + b.arg(q.Text) + ")"}
		if q.TextAlt != "" && q.TextAlt != q.Text {
			conds = append(conds, "l.search_tsv @@ websearch_to_tsquery('simple', "+b.arg(q.TextAlt)+")")
		}
		if hasNonASCII(q.Text) {
			var likes []string
			for _, tok := range strings.Fields(q.Text) {
				if hasNonASCII(tok) {
					likes = append(likes, "l.title ILIKE "+b.arg("%"+tok+"%"))
				}
			}
			if len(likes) > 0 {
				conds = append(conds, "("+strings.Join(likes, " AND ")+")")
			}
		}
		b.where("(" + strings.Join(conds, " OR ") + ")")
	}
	if len(q.Brands) > 0 {
		b.where("b.slug = ANY(" + b.arg(q.Brands) + ")")
	}
	if q.Model != "" {
		b.where("(l.model ILIKE " + b.arg("%"+q.Model+"%") + " OR l.title ILIKE " + b.arg("%"+q.Model+"%") + ")")
	}
	if q.Reference != "" {
		b.where("upper(l.reference_number) LIKE upper(" + b.arg(q.Reference+"%") + ")")
	}
	if len(q.Sources) > 0 {
		b.where("s.key = ANY(" + b.arg(q.Sources) + ")")
	}
	if len(q.Conditions) > 0 {
		b.where("l.condition = ANY(" + b.arg(toStrings(q.Conditions)) + ")")
	}
	if len(q.Movements) > 0 {
		b.where("l.movement = ANY(" + b.arg(toStrings(q.Movements)) + ")")
	}
	if len(q.Genders) > 0 {
		b.where("l.gender = ANY(" + b.arg(toStrings(q.Genders)) + ")")
	}
	if len(q.Countries) > 0 {
		b.where("l.location_country = ANY(" + b.arg(q.Countries) + ")")
	}
	if len(q.DialColors) > 0 {
		b.where("l.dial_color = ANY(" + b.arg(q.DialColors) + ")")
	}
	if q.PriceMinUSD != nil {
		b.where("l.price_usd >= " + b.arg(*q.PriceMinUSD))
	}
	if q.PriceMaxUSD != nil {
		b.where("l.price_usd <= " + b.arg(*q.PriceMaxUSD))
	}
	if q.YearMin != nil {
		b.where("l.year >= " + b.arg(*q.YearMin))
	}
	if q.YearMax != nil {
		b.where("l.year <= " + b.arg(*q.YearMax))
	}
	if q.DiameterMin != nil {
		b.where("l.case_diameter_mm >= " + b.arg(*q.DiameterMin))
	}
	if q.DiameterMax != nil {
		b.where("l.case_diameter_mm <= " + b.arg(*q.DiameterMax))
	}
	if q.HasBox != nil {
		b.where("l.has_box = " + b.arg(*q.HasBox))
	}
	if q.HasPapers != nil {
		b.where("l.has_papers = " + b.arg(*q.HasPapers))
	}

	// Everything above is part of the WHERE clause and shared by count, page and facet queries.
	whereSQL := b.whereSQL()
	whereArgs := append([]any(nil), b.args...)

	orderBy := "l.last_seen_at DESC, l.id DESC"
	switch q.Sort {
	case model.SortPriceAsc:
		orderBy = "l.price_usd ASC NULLS LAST, l.id"
	case model.SortPriceDesc:
		orderBy = "l.price_usd DESC NULLS LAST, l.id"
	case model.SortNewest:
		orderBy = "l.first_seen_at DESC, l.id DESC"
	case model.SortOldest:
		orderBy = "l.first_seen_at ASC, l.id"
	case model.SortYearDesc:
		orderBy = "l.year DESC NULLS LAST, l.id"
	case model.SortYearAsc:
		orderBy = "l.year ASC NULLS LAST, l.id"
	case model.SortSizeAsc:
		orderBy = "l.case_diameter_mm ASC NULLS LAST, l.id"
	case model.SortSizeDesc:
		orderBy = "l.case_diameter_mm DESC NULLS LAST, l.id"
	case model.SortRelevance:
		if q.Text != "" {
			rankText := q.Text
			if q.TextAlt != "" {
				rankText = q.Text + " " + q.TextAlt
			}
			orderBy = "ts_rank(l.search_tsv, websearch_to_tsquery('simple', " + b.arg(rankText) + ")) DESC, l.price_usd ASC NULLS LAST"
		}
	}

	perPage := q.PerPage
	if perPage <= 0 || perPage > 120 {
		perPage = 30
	}
	page := q.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * perPage

	res := &model.SearchResult{Page: page, PerPage: perPage, Items: []model.Listing{}, Facets: model.NewFacets()}

	// total + price bounds
	if err := r.pool.QueryRow(ctx, `SELECT count(*), min(l.price_usd), max(l.price_usd)`+listingFrom+whereSQL, whereArgs...).
		Scan(&res.Total, &res.Facets.PriceMin, &res.Facets.PriceMax); err != nil {
		return nil, fmt.Errorf("count: %w", err)
	}
	res.TotalPages = (res.Total + perPage - 1) / perPage

	rows, err := r.pool.Query(ctx, `SELECT `+listingColumns+listingFrom+whereSQL+` ORDER BY `+orderBy+
		fmt.Sprintf(" LIMIT %d OFFSET %d", perPage, offset), b.args...)
	if err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}
	items, err := collect(rows)
	rows.Close()
	if err != nil {
		return nil, err
	}
	res.Items = items

	// facets (single round-trip via batch)
	facetSQL := func(expr, label string) string {
		return `SELECT ` + expr + ` AS k, ` + label + ` AS lbl, count(*) n` + listingFrom + whereSQL + ` AND ` + expr + ` IS NOT NULL GROUP BY 1,2 ORDER BY n DESC LIMIT 40`
	}
	batch := &pgx.Batch{}
	batch.Queue(facetSQL("b.slug", "b.name"), whereArgs...)
	batch.Queue(facetSQL("s.key", "s.name"), whereArgs...)
	batch.Queue(facetSQL("l.condition", "l.condition"), whereArgs...)
	batch.Queue(facetSQL("l.movement", "l.movement"), whereArgs...)
	batch.Queue(facetSQL("nullif(l.location_country,'')", "nullif(l.location_country,'')"), whereArgs...)
	batch.Queue(facetSQL("l.dial_color", "l.dial_color"), whereArgs...)
	batch.Queue(`SELECT l.year::text AS k, l.year::text AS lbl, count(*) n`+listingFrom+whereSQL+` AND l.year IS NOT NULL GROUP BY 1,2 ORDER BY k DESC LIMIT 60`, whereArgs...)
	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()
	targets := []*[]model.FacetValue{&res.Facets.Brands, &res.Facets.Sources, &res.Facets.Conditions, &res.Facets.Movements, &res.Facets.Countries, &res.Facets.DialColors, &res.Facets.Years}
	for _, t := range targets {
		frows, err := br.Query()
		if err != nil {
			return nil, fmt.Errorf("facets: %w", err)
		}
		vals := []model.FacetValue{}
		for frows.Next() {
			var fv model.FacetValue
			if err := frows.Scan(&fv.Key, &fv.Label, &fv.Count); err != nil {
				frows.Close()
				return nil, err
			}
			fv.Key, fv.Label = strings.TrimSpace(fv.Key), strings.TrimSpace(fv.Label)
			vals = append(vals, fv)
		}
		frows.Close()
		*t = vals
	}
	return res, nil
}

// UpsertResult reports what changed during an upsert.
type UpsertResult struct {
	ID           int64
	IsNew        bool
	PriceChanged bool
}

// UpsertListing inserts or refreshes a listing and records price history when it changes.
func (r *Repo) UpsertListing(ctx context.Context, l *model.Listing) (UpsertResult, error) {
	var res UpsertResult
	attrs := l.Attributes
	if len(attrs) == 0 {
		attrs = json.RawMessage(`{}`)
	}
	var inserted bool
	err := r.pool.QueryRow(ctx, `
		INSERT INTO listings (source_id, external_id, url, title, brand_id, brand_name, model, reference_number, condition, year,
			case_diameter_mm, case_material, movement, gender, has_box, has_papers, price, currency, price_usd, shipping_price,
			location_country, location_city, seller_name, seller_type, image_urls, description, attributes, is_active,
			first_seen_at, last_seen_at, updated_at, dial_color)
		VALUES ($1,$2,$3,$4,$5,nullif($6,''),nullif($7,''),nullif($8,''),$9,$10,$11,nullif($12,''),$13,$14,$15,$16,$17,nullif($18,''),$19,$20,
			nullif($21,''),nullif($22,''),nullif($23,''),nullif($24,''),$25,nullif($26,''),$27,TRUE,now(),now(),now(),nullif($28,''))
		ON CONFLICT (source_id, external_id) DO UPDATE SET
			url = EXCLUDED.url, title = EXCLUDED.title, brand_id = EXCLUDED.brand_id, brand_name = EXCLUDED.brand_name,
			model = coalesce(EXCLUDED.model, listings.model), reference_number = coalesce(EXCLUDED.reference_number, listings.reference_number),
			condition = EXCLUDED.condition, year = coalesce(EXCLUDED.year, listings.year),
			case_diameter_mm = coalesce(EXCLUDED.case_diameter_mm, listings.case_diameter_mm),
			case_material = coalesce(EXCLUDED.case_material, listings.case_material),
			dial_color = EXCLUDED.dial_color, -- derived from the current title/description: always refresh
			movement = EXCLUDED.movement, gender = EXCLUDED.gender,
			has_box = coalesce(EXCLUDED.has_box, listings.has_box), has_papers = coalesce(EXCLUDED.has_papers, listings.has_papers),
			price = EXCLUDED.price, currency = EXCLUDED.currency, price_usd = EXCLUDED.price_usd, shipping_price = EXCLUDED.shipping_price,
			location_country = coalesce(EXCLUDED.location_country, listings.location_country),
			location_city = coalesce(EXCLUDED.location_city, listings.location_city),
			seller_name = coalesce(EXCLUDED.seller_name, listings.seller_name), seller_type = coalesce(EXCLUDED.seller_type, listings.seller_type),
			image_urls = CASE WHEN cardinality(EXCLUDED.image_urls) > 0 THEN EXCLUDED.image_urls ELSE listings.image_urls END,
			description = coalesce(EXCLUDED.description, listings.description),
			attributes = listings.attributes || EXCLUDED.attributes,
			is_active = TRUE, last_seen_at = now(), updated_at = now()
		RETURNING id, (xmax = 0) AS inserted`,
		l.SourceID, l.ExternalID, l.URL, l.Title, l.BrandID, l.BrandName, l.Model, l.ReferenceNumber, string(l.Condition), l.Year,
		l.CaseDiameterMM, l.CaseMaterial, string(l.Movement), string(l.Gender), l.HasBox, l.HasPapers, l.Price, l.Currency, l.PriceUSD, l.ShippingPrice,
		l.LocationCountry, l.LocationCity, l.SellerName, l.SellerType, l.ImageURLs, l.Description, attrs, l.DialColor,
	).Scan(&res.ID, &inserted)
	if err != nil {
		return res, err
	}
	res.IsNew = inserted
	// Compare against the latest price_history point to decide whether the price moved.
	var lastHist *float64
	histErr := r.pool.QueryRow(ctx, `SELECT price FROM price_history WHERE listing_id=$1 ORDER BY observed_at DESC LIMIT 1`, res.ID).Scan(&lastHist)
	switch {
	case errors.Is(histErr, pgx.ErrNoRows):
		// first observation
	case histErr != nil:
		return res, histErr
	case floatEq(lastHist, l.Price):
		return res, nil // unchanged
	default:
		res.PriceChanged = true
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO price_history (listing_id, price, currency, price_usd) VALUES ($1,$2,nullif($3,''),$4)`,
		res.ID, l.Price, l.Currency, l.PriceUSD); err != nil {
		return res, err
	}
	return res, nil
}

func floatEq(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	d := *a - *b
	return d < 0.005 && d > -0.005
}

// DeactivateStale marks listings for a source that were not seen since `before` as inactive.
func (r *Repo) DeactivateStale(ctx context.Context, sourceID int16, before time.Time) (int, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE listings SET is_active = FALSE, updated_at = now() WHERE source_id=$1 AND is_active AND last_seen_at < $2`, sourceID, before)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// ---------- crawl runs ----------

// StartCrawlRun opens an audit row.
func (r *Repo) StartCrawlRun(ctx context.Context, sourceID int16) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `INSERT INTO crawl_runs (source_id) VALUES ($1) RETURNING id`, sourceID).Scan(&id)
	return id, err
}

// FinishCrawlRun closes an audit row.
func (r *Repo) FinishCrawlRun(ctx context.Context, id int64, status string, seen, added, updated, deactivated int, runErr error) error {
	var msg *string
	if runErr != nil {
		s := runErr.Error()
		msg = &s
	}
	_, err := r.pool.Exec(ctx, `UPDATE crawl_runs SET finished_at=now(), status=$2, listings_seen=$3, listings_new=$4, listings_updated=$5, listings_deactivated=$6, error=$7 WHERE id=$1`,
		id, status, seen, added, updated, deactivated, msg)
	return err
}

// RecentCrawlRuns lists the latest runs.
func (r *Repo) RecentCrawlRuns(ctx context.Context, limit int) ([]model.CrawlRun, error) {
	rows, err := r.pool.Query(ctx, `SELECT c.id, s.key, c.started_at, c.finished_at, c.status, c.listings_seen, c.listings_new, c.listings_updated, c.listings_deactivated, coalesce(c.error,'')
		FROM crawl_runs c JOIN sources s ON s.id=c.source_id ORDER BY c.started_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.CrawlRun{}
	for rows.Next() {
		var c model.CrawlRun
		if err := rows.Scan(&c.ID, &c.SourceKey, &c.StartedAt, &c.FinishedAt, &c.Status, &c.ListingsSeen, &c.ListingsNew, &c.ListingsUpdated, &c.ListingsDeactivated, &c.Error); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Stats returns headline numbers for the home page.
func (r *Repo) Stats(ctx context.Context) (map[string]any, error) {
	var active, brands int
	var lastCrawl *time.Time
	if err := r.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM listings WHERE is_active), (SELECT count(*) FROM brands WHERE listing_count>0), (SELECT max(finished_at) FROM crawl_runs WHERE status IN ('ok','partial'))`).Scan(&active, &brands, &lastCrawl); err != nil {
		return nil, err
	}
	perSource := []map[string]any{}
	rows, err := r.pool.Query(ctx, `SELECT s.key, s.name, count(l.id) FROM sources s LEFT JOIN listings l ON l.source_id=s.id AND l.is_active GROUP BY s.id ORDER BY s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var k, n string
		var c int
		if err := rows.Scan(&k, &n, &c); err != nil {
			return nil, err
		}
		perSource = append(perSource, map[string]any{"key": k, "name": n, "count": c})
	}
	return map[string]any{"activeListings": active, "brands": brands, "lastCrawlAt": lastCrawl, "sources": perSource}, nil
}

// ---------- query builder ----------

type builder struct {
	conds []string
	args  []any
}

func newBuilder() *builder { return &builder{} }

func (b *builder) arg(v any) string {
	b.args = append(b.args, v)
	return fmt.Sprintf("$%d", len(b.args))
}

func (b *builder) where(c string) { b.conds = append(b.conds, c) }

func (b *builder) whereSQL() string {
	if len(b.conds) == 0 {
		return " WHERE TRUE"
	}
	return " WHERE " + strings.Join(b.conds, " AND ")
}

func hasNonASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return true
		}
	}
	return false
}

func toStrings[T ~string](in []T) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = string(v)
	}
	return out
}
