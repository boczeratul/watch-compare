INSERT INTO sources (key, name, base_url, country, currency) VALUES
    ('watchexchange', 'r/Watchexchange', 'https://www.reddit.com/r/Watchexchange', 'US', 'USD')
ON CONFLICT (key) DO NOTHING;
