INSERT INTO sources (key, name, base_url, country, currency) VALUES
    ('bbc0804', '仁川當舖', 'https://www.bbc0804.com.tw', 'TW', 'TWD')
ON CONFLICT (key) DO NOTHING;
