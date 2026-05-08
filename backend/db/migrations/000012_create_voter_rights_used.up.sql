-- ─────────────────────────────────────────────────────────────
-- CRITICAL ANONYMIZATION TABLE (PRD §6.1, §9.2)
--
-- Records WHICH ballot types a voter session has cast.
-- Records NO vote choice — the choice lives only in vote_events.
-- Has NO foreign key to vote_events — joining them is impossible by design.
-- The only bridge (anonymous_vote_token) exists only in-memory during
-- the atomic cast transaction and in the voter's browser session.
-- ─────────────────────────────────────────────────────────────
CREATE TABLE voter_rights_used (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    voter_session_id UUID        NOT NULL REFERENCES voter_sessions(id),
    ballot_type      ballot_type NOT NULL,
    voted_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()

    -- ⚠️  NO reference to vote_events — by design
    -- ⚠️  NO vote choice stored here
);

-- Prevents double-voting per session per ballot type
CREATE UNIQUE INDEX idx_voter_rights_session_ballot
    ON voter_rights_used (voter_session_id, ballot_type);

CREATE INDEX idx_voter_rights_session
    ON voter_rights_used (voter_session_id);
