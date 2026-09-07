ALTER TABLE listings ADD COLUMN IF NOT EXISTS dial_color TEXT;
CREATE INDEX IF NOT EXISTS listings_dial_color_idx ON listings (dial_color) WHERE is_active;
CREATE INDEX IF NOT EXISTS listings_year_idx ON listings (year) WHERE is_active;
