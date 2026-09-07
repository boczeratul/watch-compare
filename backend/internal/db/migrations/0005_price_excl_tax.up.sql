-- Japanese dealers must display tax-included prices (総額表示義務), but an exporting buyer pays the
-- tax-free amount. Store it alongside so the detail page can show both; price_usd (used for all
-- sorting and range filters) is computed from this column when it is present.
ALTER TABLE listings ADD COLUMN IF NOT EXISTS price_excl_tax NUMERIC(14,2);
