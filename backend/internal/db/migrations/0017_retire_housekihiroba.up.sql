-- Housekihiroba (宝石広場) is no longer crawled. Disable the source and retire its listings so
-- they disappear from search now rather than never (the stale window only runs for crawled sources).
-- The source row stays because listings and price history reference it.
UPDATE sources SET enabled = false WHERE key = 'housekihiroba';
UPDATE listings SET is_active = false, updated_at = now()
WHERE is_active AND source_id = (SELECT id FROM sources WHERE key = 'housekihiroba');
