DELETE FROM sources WHERE key IN ('rdwatch', 'commitwatch') AND NOT EXISTS (SELECT 1 FROM listings l JOIN sources s ON s.id = l.source_id WHERE s.key IN ('rdwatch', 'commitwatch'));
