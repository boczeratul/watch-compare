DROP INDEX IF EXISTS listings_ref_bare_idx;
ALTER TABLE listings DROP COLUMN IF EXISTS search_tsv;
ALTER TABLE listings ADD COLUMN search_tsv tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(brand_name, '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(reference_number, '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(model, '')), 'B') ||
    setweight(to_tsvector('simple', coalesce(title, '')), 'C')
) STORED;
CREATE INDEX IF NOT EXISTS listings_tsv_idx ON listings USING GIN (search_tsv);
