INSERT INTO sources (key, name, base_url, country, currency) VALUES
    ('999watch', '鴻昇名錶', 'https://www.999watch.com', 'TW', 'TWD')
ON CONFLICT (key) DO NOTHING;
