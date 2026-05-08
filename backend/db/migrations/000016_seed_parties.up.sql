-- 2026 Thai Election parties with real results from PRD §1.3.4
-- Colors chosen to approximate official party branding

INSERT INTO parties (id, name, short_name, color_hex) VALUES
    ('a1000000-0000-0000-0000-000000000001', 'Bhumjaithai Party',          'BJT',  '#1a5c24'),
    ('a1000000-0000-0000-0000-000000000002', 'People''s Party',            'PP',   '#F97316'),
    ('a1000000-0000-0000-0000-000000000003', 'Pheu Thai Party',            'PT',   '#e53e3e'),
    ('a1000000-0000-0000-0000-000000000004', 'Kla Tham Party',             'KT',   '#6366f1'),
    ('a1000000-0000-0000-0000-000000000005', 'Democrat Party',             'DP',   '#3b82f6'),
    ('a1000000-0000-0000-0000-000000000006', 'Thai Liberal Party',         'TLP',  '#f59e0b'),
    ('a1000000-0000-0000-0000-000000000007', 'New Economics Party',        'NEP',  '#10b981'),
    ('a1000000-0000-0000-0000-000000000008', 'Chartthaipattana Party',     'CTP',  '#8b5cf6'),
    ('a1000000-0000-0000-0000-000000000009', 'Thai Nation Power Party',    'TNP',  '#ef4444'),
    ('a1000000-0000-0000-0000-00000000000a', 'United Thai Nation Party',   'UTN',  '#64748b')
ON CONFLICT (id) DO NOTHING;
