-- ─────────────────────────────────────────────────────────────
-- ONLINE VOTING — IDENTITY SIDE
-- (PRD §6.1, §9.2)
--
-- NEVER stores plaintext national_id.
-- NEVER joined with vote_events — enforced by schema (no FK).
-- ─────────────────────────────────────────────────────────────

-- Eligible voter registry — national_id stored as SHA-256(id + pepper)
CREATE TABLE voter_registry (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    -- SHA-256(national_id + NATIONAL_ID_PEPPER) — never the raw ID
    national_id_hash  VARCHAR(64) NOT NULL,
    province_id       SMALLINT    NOT NULL REFERENCES provinces(id),
    constituency_id   UUID        NOT NULL REFERENCES constituencies(id),
    -- Phone number encrypted at rest (AES-256-GCM, key from Vault)
    registered_phone  VARCHAR(200) NOT NULL,
    is_eligible       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_voter_registry_id_hash ON voter_registry (national_id_hash);
CREATE INDEX idx_voter_registry_province       ON voter_registry (province_id);
