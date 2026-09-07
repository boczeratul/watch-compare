-- Data fix: straps and bracelet links were crawled under watch brands before the crawler learned
-- to drop them (normalize.NonWatchBrands / NonWatchTitleMarkers). Retire the rows that are
-- already stored so they disappear from search now rather than after the 72h stale window.
UPDATE listings SET is_active = false, updated_at = now()
WHERE is_active AND (title ILIKE '%vagenari%' OR model ILIKE '%vagenari%' OR title LIKE '%錶節%');
