INSERT INTO sources (key, name, base_url, country, currency) VALUES
    ('rasin',      'GINZA RASIN',  'https://www.rasin.co.jp',  'JP', 'JPY'),
    ('quark',      'Quark',        'https://www.909.co.jp',    'JP', 'JPY'),
    ('bellemonde', 'Belle Monde',  'https://bellemonde.tokyo', 'JP', 'JPY')
ON CONFLICT (key) DO NOTHING;
