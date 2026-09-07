INSERT INTO sources (key, name, base_url, country, currency) VALUES
    ('rww', 'RWW Watch', 'https://rwwwatch.com', 'HK', 'HKD')
ON CONFLICT (key) DO NOTHING;
