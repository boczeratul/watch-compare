DELETE FROM sources s
WHERE s.key IN ('369rolexwatch', 'dzwatch', 'goodtimezone', 'egps', 'xintai', 'watchstore')
  AND NOT EXISTS (SELECT 1 FROM listings l WHERE l.source_id = s.id);
