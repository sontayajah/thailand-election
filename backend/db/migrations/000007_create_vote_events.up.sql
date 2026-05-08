-- ─────────────────────────────────────────────────────────────
-- WRITE MODEL — Append-only event log (physical + online votes)
-- (PRD §6.1, §9.2)
--
-- CRITICAL ANONYMIZATION RULE:
--   For online votes: anonymous_token = random UUID (NOT national_id)
--   For physical votes: anonymous_token = NULL
--   The vote record contains NO personally identifiable information.
-- ─────────────────────────────────────────────────────────────
CREATE TABLE vote_events (
    id                BIGSERIAL    PRIMARY KEY,
    ballot_type       ballot_type  NOT NULL,
    source            vote_source  NOT NULL DEFAULT 'physical',
    province_id       SMALLINT     NOT NULL REFERENCES provinces(id),
    constituency_id   UUID         REFERENCES constituencies(id),   -- NULL for PARTY_LIST / REFERENDUM
    candidate_id      UUID         REFERENCES candidates(id),        -- NULL for REFERENDUM
    referendum_vote   referendum_vote,                               -- populated for REFERENDUM only
    vote_count        INT          NOT NULL CHECK (vote_count > 0),
    -- online: anonymous_vote_token (NEVER national_id — PRD §9.2)
    -- physical/admin: NULL
    anonymous_token   UUID,
    -- Prevents double-counting on retries (PRD §1.2 Idempotency)
    -- Format: {source}-{ballot_type}-prov{id}-{unix_ts}-{uuid}
    idempotency_key   VARCHAR(150) NOT NULL,
    payload_signature TEXT,                                          -- Ed25519 sig for physical votes
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Unique constraint enforces idempotency
CREATE UNIQUE INDEX idx_vote_events_idempotency
    ON vote_events (idempotency_key);

-- Efficient province+ballot queries for summaries
CREATE INDEX idx_vote_events_province_ballot
    ON vote_events (province_id, ballot_type, created_at DESC);

-- Source breakdown reporting
CREATE INDEX idx_vote_events_source
    ON vote_events (source, ballot_type);

-- Candidate aggregation
CREATE INDEX idx_vote_events_candidate
    ON vote_events (candidate_id) WHERE candidate_id IS NOT NULL;

-- Worker DB role: INSERT only — enforced in application layer
-- ALTER TABLE vote_events ENABLE ROW LEVEL SECURITY; (for production)
