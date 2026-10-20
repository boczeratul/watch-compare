DELETE FROM sources WHERE key = '999watch' AND NOT EXISTS (SELECT 1 FROM listings l JOIN sources s ON s.id = l.source_id WHERE s.key = '999watch');
