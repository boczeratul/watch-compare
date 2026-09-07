-- Trigram index so CJK words (which the 'simple' text-search parser ignores) can be matched
-- with ILIKE '%…%' on titles at index speed. Available on Cloud SQL and vanilla PostgreSQL.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS listings_title_trgm_idx ON listings USING GIN (title gin_trgm_ops);
