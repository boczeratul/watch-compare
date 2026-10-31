INSERT INTO sources (key, name, base_url, country, currency) VALUES
    ('369rolexwatch', '金宸名錶', 'https://www.369rolexwatch.com', 'TW', 'TWD'),
    ('dzwatch', '大眾名錶', 'https://www.dzwatch.tw', 'TW', 'TWD'),
    ('goodtimezone', '好時計', 'https://www.goodtimezone.com.tw', 'TW', 'TWD'),
    ('egps', '永生／永久名錶', 'https://www.egps.com.tw', 'TW', 'TWD'),
    ('xintai', '鑫泰當鋪', 'https://www.89675555.com', 'TW', 'TWD'),
    ('watchstore', '名錶窩', 'https://www.watchstore.tw', 'TW', 'TWD')
ON CONFLICT (key) DO NOTHING;
