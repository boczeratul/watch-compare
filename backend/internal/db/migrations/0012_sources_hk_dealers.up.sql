INSERT INTO sources (key, name, base_url, country, currency) VALUES
    ('watchfinderhk', 'Watchfinder & Co. HK', 'https://www.watchfinder.hk', 'HK', 'HKD'),
    ('wristcheck',    'Wristcheck',           'https://wristcheck.com',      'HK', 'HKD'),
    ('kenwatches',    'Ken''s Watches',       'https://kenwatches.com',      'HK', 'HKD')
ON CONFLICT (key) DO NOTHING;
