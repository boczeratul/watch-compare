DROP INDEX IF EXISTS listings_year_idx;
DROP INDEX IF EXISTS listings_dial_color_idx;
ALTER TABLE listings DROP COLUMN IF EXISTS dial_color;
