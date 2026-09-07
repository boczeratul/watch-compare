INSERT INTO sources (key, name, base_url, country, currency) VALUES
    ('sevenhours',    '7hours',          'https://7hours.jp',          'JP', 'JPY'),
    ('lips',          'Brand Shop LIPS', 'https://lips-online.jp',     'JP', 'JPY'),
    ('allu',          'ALLU',            'https://allu-official.com',  'JP', 'JPY'),
    ('housekihiroba', 'Housekihiroba',   'https://housekihiroba.jp',   'JP', 'JPY')
ON CONFLICT (key) DO NOTHING;
