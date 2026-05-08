-- ─────────────────────────────────────────────────────────────
-- VOTE RECEIPTS (PRD §4.2 V-07, V-10)
--
-- Linked to anonymous_vote_token — NOT to voter identity.
-- Allows public verification that a vote was counted,
-- without revealing the choice or the voter's identity.
--
-- receipt_hash = SHA-256(anonymous_token + choice_id + voted_at_unix)
-- ─────────────────────────────────────────────────────────────
CREATE TABLE vote_receipts (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_hash    VARCHAR(64) NOT NULL,                    -- SHA-256 hex
    anonymous_token UUID        NOT NULL,                    -- matches vote_events.anonymous_token
    ballot_type     ballot_type NOT NULL,
    issued_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()

    -- No voter_id, no national_id, no province linkage to voter
);

CREATE UNIQUE INDEX idx_receipts_hash           ON vote_receipts (receipt_hash);
CREATE INDEX        idx_receipts_anonymous_token ON vote_receipts (anonymous_token);
