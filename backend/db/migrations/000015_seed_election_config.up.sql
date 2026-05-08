-- Dev seed: election is open, online voting enabled (PRD §6.3)
INSERT INTO election_config (key, value) VALUES
    ('election_status',          'open'),
    ('online_voting_enabled',    'true'),
    ('online_voting_window_start', '2026-02-08T01:00:00Z'),
    ('online_voting_window_end', '2026-02-08T10:00:00Z'),
    ('election_year',            '2569'),
    ('referendum_question',      'Do you agree that a new Constitution should be drafted by a Constitution Drafting Assembly (CDA) elected by the people?'),
    ('total_eligible_voters',    '52186982'),
    ('dopa_api_url',             'http://dopa-mock:9090'),
    ('max_otp_attempts',         '5'),
    ('otp_ttl_seconds',          '300'),
    ('voter_session_ttl_minutes','30')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW();
