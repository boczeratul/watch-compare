INSERT INTO sources (key, name, base_url, country, currency) VALUES
    ('jdpawn', '久大御典品', 'https://www.jdpawn.com.tw', 'TW', 'TWD')
ON CONFLICT (key) DO NOTHING;
