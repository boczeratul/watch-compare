-- Data fix: Chrono24 "Certified" cards render the seal (…/certified/certified-filled.svg) as an
-- <img> ahead of the photo, and the crawler stored it as the listing's first image. The parser now
-- skips it; this removes the SVG URLs already stored, keeping the remaining images in order.
-- The upsert keeps the previous images when a crawl returns none, so badge-only rows would
-- otherwise never heal.
UPDATE listings l
SET image_urls = ARRAY(
        SELECT u FROM unnest(l.image_urls) WITH ORDINALITY AS t(u, ord)
        WHERE lower(split_part(u, '?', 1)) NOT LIKE '%.svg'
        ORDER BY ord),
    updated_at = now()
WHERE EXISTS (SELECT 1 FROM unnest(l.image_urls) AS u WHERE lower(split_part(u, '?', 1)) LIKE '%.svg');
