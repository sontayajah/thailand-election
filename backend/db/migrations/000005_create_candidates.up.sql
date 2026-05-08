-- Candidates for CONSTITUENCY and PARTY_LIST ballots
-- NOTE (PRD §1.3.2): candidate numbers on green ballot ≠ party numbers on pink ballot.
-- Models are strictly separated; join only via party_id.
CREATE TABLE candidates (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    party_id         UUID        NOT NULL REFERENCES parties(id),
    constituency_id  UUID        REFERENCES constituencies(id),  -- NULL for PARTY_LIST candidates
    full_name        VARCHAR(200) NOT NULL,
    photo_url        TEXT,
    ballot_type      ballot_type  NOT NULL,
    ballot_number    SMALLINT,                                    -- candidate number on their ballot
    party_list_order SMALLINT,                                    -- order for PARTY_LIST only
    is_active        BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_candidates_party      ON candidates (party_id);
CREATE INDEX idx_candidates_ballot     ON candidates (ballot_type, is_active);
CREATE INDEX idx_candidates_constituency ON candidates (constituency_id) WHERE constituency_id IS NOT NULL;
