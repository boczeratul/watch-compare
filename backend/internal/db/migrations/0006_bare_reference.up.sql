-- Reference numbers are often quoted without the maker's letter prefix: IWC's IW328903 is sold and
-- searched as "328903", Breitling's AB0138 as "0138" and so on. The 'simple' parser keeps
-- "iw328903" as one lexeme, and the search only matches lexemes by prefix, so "328903" found
-- nothing. Index the bare number alongside the full reference so both spellings match.
-- A generated column's expression cannot be altered in place: rebuild the column and its index.
ALTER TABLE listings DROP COLUMN IF EXISTS search_tsv;
ALTER TABLE listings ADD COLUMN search_tsv tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('simple', coalesce(brand_name, '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(reference_number, '')), 'A') ||
    setweight(to_tsvector('simple', regexp_replace(coalesce(reference_number, ''), '^[A-Za-z]+', '')), 'A') ||
    setweight(to_tsvector('simple', coalesce(model, '')), 'B') ||
    setweight(to_tsvector('simple', coalesce(title, '')), 'C')
) STORED;
CREATE INDEX IF NOT EXISTS listings_tsv_idx ON listings USING GIN (search_tsv);
-- Same for the structured ?ref= filter, which matches by prefix on the stored reference.
CREATE INDEX IF NOT EXISTS listings_ref_bare_idx ON listings (regexp_replace(upper(reference_number), '^[A-Z]+', '')) WHERE is_active;
