-- Sources we crawl
CREATE TABLE IF NOT EXISTS sources (
    id          SMALLSERIAL PRIMARY KEY,
    key         TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    base_url    TEXT NOT NULL,
    country     CHAR(2) NOT NULL,
    currency    CHAR(3) NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE
);

INSERT INTO sources (key, name, base_url, country, currency) VALUES
    ('chrono24',  'Chrono24',  'https://www.chrono24.com',      'DE', 'EUR'),
    ('ebay',      'eBay',      'https://www.ebay.com',          'US', 'USD'),
    ('watchnian', 'Watchnian', 'https://watchnian.com',         'JP', 'JPY'),
    ('jackroad',  'Jackroad',  'https://www.jackroad.co.jp',    'JP', 'JPY'),
    ('hourstack', 'Hourstack', 'https://www.hourstack.com.tw',  'TW', 'TWD')
ON CONFLICT (key) DO NOTHING;

-- Canonical brands
CREATE TABLE IF NOT EXISTS brands (
    id          SERIAL PRIMARY KEY,
    slug        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    listing_count INTEGER NOT NULL DEFAULT 0
);

-- Listings: one row per (source, external_id). Images are stored as URLs only.
CREATE TABLE IF NOT EXISTS listings (
    id                BIGSERIAL PRIMARY KEY,
    source_id         SMALLINT NOT NULL REFERENCES sources(id),
    external_id       TEXT NOT NULL,
    url               TEXT NOT NULL,
    title             TEXT NOT NULL,
    brand_id          INTEGER REFERENCES brands(id),
    brand_name        TEXT,
    model             TEXT,
    reference_number  TEXT,
    condition         TEXT NOT NULL DEFAULT 'unknown',   -- new | unworn | very_good | good | fair | poor | unknown
    year              SMALLINT,
    case_diameter_mm  NUMERIC(5,1),
    case_material     TEXT,
    movement          TEXT,                              -- automatic | manual | quartz | unknown
    gender            TEXT,                              -- men | women | unisex | unknown
    has_box           BOOLEAN,
    has_papers        BOOLEAN,
    price             NUMERIC(14,2),
    currency          CHAR(3),
    price_usd         NUMERIC(14,2),                     -- normalized at crawl time for cross-source sorting
    shipping_price    NUMERIC(14,2),
    location_country  CHAR(2),
    location_city     TEXT,
    seller_name       TEXT,
    seller_type       TEXT,                              -- dealer | private | unknown
    image_urls        TEXT[] NOT NULL DEFAULT '{}',
    description       TEXT,
    attributes        JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_active         BOOLEAN NOT NULL DEFAULT TRUE,
    first_seen_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    search_tsv        tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', coalesce(brand_name, '')), 'A') ||
        setweight(to_tsvector('simple', coalesce(reference_number, '')), 'A') ||
        setweight(to_tsvector('simple', coalesce(model, '')), 'B') ||
        setweight(to_tsvector('simple', coalesce(title, '')), 'C')
    ) STORED,
    UNIQUE (source_id, external_id)
);

CREATE INDEX IF NOT EXISTS listings_tsv_idx        ON listings USING GIN (search_tsv);
CREATE INDEX IF NOT EXISTS listings_brand_idx      ON listings (brand_id) WHERE is_active;
CREATE INDEX IF NOT EXISTS listings_price_usd_idx  ON listings (price_usd) WHERE is_active;
CREATE INDEX IF NOT EXISTS listings_last_seen_idx  ON listings (last_seen_at DESC) WHERE is_active;
CREATE INDEX IF NOT EXISTS listings_ref_idx        ON listings (upper(reference_number)) WHERE is_active;
CREATE INDEX IF NOT EXISTS listings_source_idx     ON listings (source_id) WHERE is_active;
CREATE INDEX IF NOT EXISTS listings_condition_idx  ON listings (condition) WHERE is_active;

-- Price history for "deal" detection and charts
CREATE TABLE IF NOT EXISTS price_history (
    id          BIGSERIAL PRIMARY KEY,
    listing_id  BIGINT NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
    price       NUMERIC(14,2),
    currency    CHAR(3),
    price_usd   NUMERIC(14,2),
    observed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS price_history_listing_idx ON price_history (listing_id, observed_at DESC);

-- Exchange rates, base USD
CREATE TABLE IF NOT EXISTS exchange_rates (
    quote       CHAR(3) PRIMARY KEY,
    rate        NUMERIC(18,8) NOT NULL,     -- 1 USD = rate QUOTE
    fetched_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO exchange_rates (quote, rate) VALUES ('USD', 1) ON CONFLICT DO NOTHING;

-- Crawl audit log
CREATE TABLE IF NOT EXISTS crawl_runs (
    id                BIGSERIAL PRIMARY KEY,
    source_id         SMALLINT NOT NULL REFERENCES sources(id),
    started_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at       TIMESTAMPTZ,
    status            TEXT NOT NULL DEFAULT 'running',   -- running | ok | partial | failed
    listings_seen     INTEGER NOT NULL DEFAULT 0,
    listings_new      INTEGER NOT NULL DEFAULT 0,
    listings_updated  INTEGER NOT NULL DEFAULT 0,
    listings_deactivated INTEGER NOT NULL DEFAULT 0,
    error             TEXT
);

CREATE TABLE IF NOT EXISTS schema_migrations (
    version    INTEGER PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
