-- Data fix: Chrono24 cards that show "Price on request" used to be stored with the shipping cost
-- as their price. The parser now stores no price for them, and this removes the price-history
-- points recorded for those listings. Only listings whose price is already NULL are touched, so
-- the statement is safe to run before or after the crawl that nulls them.
DELETE FROM price_history ph
USING listings l JOIN sources s ON s.id = l.source_id
WHERE ph.listing_id = l.id AND s.key = 'chrono24' AND l.price IS NULL;
