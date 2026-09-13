INSERT INTO sources (key, name, base_url, country, currency) VALUES
    ('kitamura',     'Kitamura',                    'https://shop.kitamura.jp',     'JP', 'JPY'),
    ('watchandtime', 'James Huang Vintage Watches', 'https://www.watchandtime.com', 'TW', 'TWD')
ON CONFLICT (key) DO NOTHING;
