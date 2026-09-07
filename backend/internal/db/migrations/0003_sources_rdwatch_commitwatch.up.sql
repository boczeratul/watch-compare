INSERT INTO sources (key, name, base_url, country, currency) VALUES
    ('rdwatch',     'RD Watch',     'https://www.rdwatch.com.tw', 'TW', 'TWD'),
    ('commitwatch', 'Commit Ginza', 'https://commit-watch.co.jp',  'JP', 'JPY')
ON CONFLICT (key) DO NOTHING;
