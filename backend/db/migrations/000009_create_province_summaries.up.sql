-- Province-level summary per ballot_type — powers the map color fill
-- and province drill-down panel (PRD F-01, F-02)
--
-- candidate_id uses sentinel UUID '00000000-...' for REFERENDUM rows (no candidate)
-- so we can use a simple PRIMARY KEY without nullable complexity.
CREATE TABLE province_summaries (
    province_id  SMALLINT    NOT NULL REFERENCES provinces(id),
    ballot_type  ballot_type NOT NULL,
    -- For CONSTITUENCY/PARTY_LIST: real candidate/party UUID
    -- For REFERENDUM: sentinel '00000000-0000-0000-0000-000000000000'
    candidate_id UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    total_votes  BIGINT      NOT NULL DEFAULT 0,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (province_id, ballot_type, candidate_id)
);

CREATE INDEX idx_province_summaries_ballot
    ON province_summaries (ballot_type, total_votes DESC);
