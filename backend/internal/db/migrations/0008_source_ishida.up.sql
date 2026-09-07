INSERT INTO sources (key, name, base_url, country, currency) VALUES
    ('ishida', 'BEST ISHIDA', 'https://ishida-watch.com', 'JP', 'JPY')
ON CONFLICT (key) DO NOTHING;
