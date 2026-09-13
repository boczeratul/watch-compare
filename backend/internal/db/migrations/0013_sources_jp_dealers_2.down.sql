DELETE FROM sources WHERE key IN ('rasin', 'quark', 'bellemonde') AND NOT EXISTS (SELECT 1 FROM listings l JOIN sources s ON s.id = l.source_id WHERE s.key IN ('rasin', 'quark', 'bellemonde'));
