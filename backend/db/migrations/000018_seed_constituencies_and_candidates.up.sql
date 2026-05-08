-- ─────────────────────────────────────────────────────────────
-- Seed constituencies for Bangkok (33) and sample provinces,
-- plus sample candidates for the simulator and demo.
-- Full 400-constituency data would be seeded via a Go seed script.
-- ─────────────────────────────────────────────────────────────

-- Bangkok constituencies (33 districts → 33 constituencies)
INSERT INTO constituencies (id, province_id, constituency_no, name, eligible_voters) VALUES
    ('b0000000-0000-0000-0000-000000000001', 10, 1,  'Bangkok District 1 (Phra Nakhon)',       120000),
    ('b0000000-0000-0000-0000-000000000002', 10, 2,  'Bangkok District 2 (Dusit)',              110000),
    ('b0000000-0000-0000-0000-000000000003', 10, 3,  'Bangkok District 3 (Nong Chok)',           95000),
    ('b0000000-0000-0000-0000-000000000004', 10, 4,  'Bangkok District 4 (Bang Rak)',           130000),
    ('b0000000-0000-0000-0000-000000000005', 10, 5,  'Bangkok District 5 (Lat Krabang)',        145000),
    ('b0000000-0000-0000-0000-000000000006', 10, 6,  'Bangkok District 6 (Yannawa)',            125000),
    ('b0000000-0000-0000-0000-000000000007', 10, 7,  'Bangkok District 7 (Samphanthawong)',      80000),
    ('b0000000-0000-0000-0000-000000000008', 10, 8,  'Bangkok District 8 (Phaya Thai)',         140000),
    ('b0000000-0000-0000-0000-000000000009', 10, 9,  'Bangkok District 9 (Min Buri)',           155000),
    ('b0000000-0000-0000-0000-00000000000a', 10, 10, 'Bangkok District 10 (Lat Phrao)',         160000),
    ('b0000000-0000-0000-0000-00000000000b', 10, 11, 'Bangkok District 11 (Wang Thonglang)',    150000),
    ('b0000000-0000-0000-0000-00000000000c', 10, 12, 'Bangkok District 12 (Khlong San)',        105000),
    ('b0000000-0000-0000-0000-00000000000d', 10, 13, 'Bangkok District 13 (Taling Chan)',       135000),
    ('b0000000-0000-0000-0000-00000000000e', 10, 14, 'Bangkok District 14 (Bangkok Noi)',       115000),
    ('b0000000-0000-0000-0000-00000000000f', 10, 15, 'Bangkok District 15 (Bangkok Yai)',       110000),
    ('b0000000-0000-0000-0000-000000000010', 10, 16, 'Bangkok District 16 (Huai Khwang)',       140000),
    ('b0000000-0000-0000-0000-000000000011', 10, 17, 'Bangkok District 17 (Khlong Toei)',       120000),
    ('b0000000-0000-0000-0000-000000000012', 10, 18, 'Bangkok District 18 (Suan Luang)',        155000),
    ('b0000000-0000-0000-0000-000000000013', 10, 19, 'Bangkok District 19 (Chatuchak)',         165000),
    ('b0000000-0000-0000-0000-000000000014', 10, 20, 'Bangkok District 20 (Bang Khen)',         170000),
    ('b0000000-0000-0000-0000-000000000015', 10, 21, 'Bangkok District 21 (Don Mueang)',        160000),
    ('b0000000-0000-0000-0000-000000000016', 10, 22, 'Bangkok District 22 (Ratchathewi)',       125000),
    ('b0000000-0000-0000-0000-000000000017', 10, 23, 'Bangkok District 23 (Lat Phrao 2)',       150000),
    ('b0000000-0000-0000-0000-000000000018', 10, 24, 'Bangkok District 24 (Bueng Kum)',         145000),
    ('b0000000-0000-0000-0000-000000000019', 10, 25, 'Bangkok District 25 (Saphan Sung)',       135000),
    ('b0000000-0000-0000-0000-00000000001a', 10, 26, 'Bangkok District 26 (Wang Thonglang 2)',  130000),
    ('b0000000-0000-0000-0000-00000000001b', 10, 27, 'Bangkok District 27 (Khlong Sam Wa)',     155000),
    ('b0000000-0000-0000-0000-00000000001c', 10, 28, 'Bangkok District 28 (Bang Na)',           140000),
    ('b0000000-0000-0000-0000-00000000001d', 10, 29, 'Bangkok District 29 (Thawi Watthana)',    110000),
    ('b0000000-0000-0000-0000-00000000001e', 10, 30, 'Bangkok District 30 (Thung Khru)',        125000),
    ('b0000000-0000-0000-0000-00000000001f', 10, 31, 'Bangkok District 31 (Bang Bon)',          115000),
    ('b0000000-0000-0000-0000-000000000020', 10, 32, 'Bangkok District 32 (Bang Khae)',         130000),
    ('b0000000-0000-0000-0000-000000000021', 10, 33, 'Bangkok District 33 (Lak Si)',            145000)
ON CONFLICT (province_id, constituency_no) DO NOTHING;

-- Sample single constituencies for major provinces
INSERT INTO constituencies (id, province_id, constituency_no, name, eligible_voters) VALUES
    ('c0000000-0000-0000-0000-000000000001', 50, 1, 'Chiang Mai District 1', 140000),
    ('c0000000-0000-0000-0000-000000000002', 50, 2, 'Chiang Mai District 2', 130000),
    ('c0000000-0000-0000-0000-000000000003', 40, 1, 'Khon Kaen District 1',  130000),
    ('c0000000-0000-0000-0000-000000000004', 90, 1, 'Songkhla District 1',   125000),
    ('c0000000-0000-0000-0000-000000000005', 80, 1, 'Nakhon Si Thammarat District 1', 130000),
    ('c0000000-0000-0000-0000-000000000006', 30, 1, 'Nakhon Ratchasima District 1', 140000)
ON CONFLICT (province_id, constituency_no) DO NOTHING;

-- ── Sample Constituency Candidates (Bangkok District 1) ──────
-- Parties: BJT(1), PP(2), PT(3), KT(4), DP(5)
INSERT INTO candidates (id, party_id, constituency_id, full_name, ballot_type, ballot_number) VALUES
    ('d0000000-0000-0000-0000-000000000001',
     'a1000000-0000-0000-0000-000000000001',
     'b0000000-0000-0000-0000-000000000001',
     'Somchai Bhumjai', 'CONSTITUENCY', 1),
    ('d0000000-0000-0000-0000-000000000002',
     'a1000000-0000-0000-0000-000000000002',
     'b0000000-0000-0000-0000-000000000001',
     'Wanchai Prachakon', 'CONSTITUENCY', 2),
    ('d0000000-0000-0000-0000-000000000003',
     'a1000000-0000-0000-0000-000000000003',
     'b0000000-0000-0000-0000-000000000001',
     'Malee Phuethai', 'CONSTITUENCY', 3),
    ('d0000000-0000-0000-0000-000000000004',
     'a1000000-0000-0000-0000-000000000004',
     'b0000000-0000-0000-0000-000000000001',
     'Prasit Klatham', 'CONSTITUENCY', 4),
    ('d0000000-0000-0000-0000-000000000005',
     'a1000000-0000-0000-0000-000000000005',
     'b0000000-0000-0000-0000-000000000001',
     'Narong Democrat', 'CONSTITUENCY', 5)
ON CONFLICT DO NOTHING;

-- ── Party List Candidates (Top 5 per major party) ─────────────
INSERT INTO candidates (id, party_id, full_name, ballot_type, party_list_order) VALUES
    -- Bhumjaithai
    ('e0000000-0000-0000-0000-000000000001', 'a1000000-0000-0000-0000-000000000001', 'Anutin Charnvirakul',  'PARTY_LIST', 1),
    ('e0000000-0000-0000-0000-000000000002', 'a1000000-0000-0000-0000-000000000001', 'Saksayam Chidchob',    'PARTY_LIST', 2),
    ('e0000000-0000-0000-0000-000000000003', 'a1000000-0000-0000-0000-000000000001', 'Supamas Isarabhakdi',  'PARTY_LIST', 3),
    -- People''s Party
    ('e0000000-0000-0000-0000-000000000004', 'a1000000-0000-0000-0000-000000000002', 'Pita Limjaroenrat',    'PARTY_LIST', 1),
    ('e0000000-0000-0000-0000-000000000005', 'a1000000-0000-0000-0000-000000000002', 'Sirikanya Tansakul',   'PARTY_LIST', 2),
    ('e0000000-0000-0000-0000-000000000006', 'a1000000-0000-0000-0000-000000000002', 'Chaithawat Tulathon',  'PARTY_LIST', 3),
    -- Pheu Thai
    ('e0000000-0000-0000-0000-000000000007', 'a1000000-0000-0000-0000-000000000003', 'Srettha Thavisin',     'PARTY_LIST', 1),
    ('e0000000-0000-0000-0000-000000000008', 'a1000000-0000-0000-0000-000000000003', 'Chonlanan Srikaew',    'PARTY_LIST', 2),
    -- Kla Tham
    ('e0000000-0000-0000-0000-000000000009', 'a1000000-0000-0000-0000-000000000004', 'Korn Chatikavanij',    'PARTY_LIST', 1),
    ('e0000000-0000-0000-0000-000000000010', 'a1000000-0000-0000-0000-000000000004', 'Apisak Tantivorawong', 'PARTY_LIST', 2),
    -- Democrat
    ('e0000000-0000-0000-0000-000000000011', 'a1000000-0000-0000-0000-000000000005', 'Jurin Laksanawisit',   'PARTY_LIST', 1),
    ('e0000000-0000-0000-0000-000000000012', 'a1000000-0000-0000-0000-000000000005', 'Chaichana Dejdecho',   'PARTY_LIST', 2)
ON CONFLICT DO NOTHING;

-- ── Initialize read model rows for seeded data ────────────────
INSERT INTO party_list_national (party_id, total_votes, seat_count) VALUES
    ('a1000000-0000-0000-0000-000000000001', 0, 0),
    ('a1000000-0000-0000-0000-000000000002', 0, 0),
    ('a1000000-0000-0000-0000-000000000003', 0, 0),
    ('a1000000-0000-0000-0000-000000000004', 0, 0),
    ('a1000000-0000-0000-0000-000000000005', 0, 0),
    ('a1000000-0000-0000-0000-000000000006', 0, 0),
    ('a1000000-0000-0000-0000-000000000007', 0, 0),
    ('a1000000-0000-0000-0000-000000000008', 0, 0),
    ('a1000000-0000-0000-0000-000000000009', 0, 0),
    ('a1000000-0000-0000-0000-00000000000a', 0, 0)
ON CONFLICT (party_id) DO NOTHING;
