-- ─────────────────────────────────────────────────────────────
-- READ MODELS — Pre-aggregated for fast queries (PRD §6.3)
-- Updated atomically by the worker after each vote batch.
-- ─────────────────────────────────────────────────────────────

-- Per-constituency candidate vote totals
CREATE TABLE constituency_summaries (
    constituency_id  UUID    NOT NULL REFERENCES constituencies(id),
    candidate_id     UUID    NOT NULL REFERENCES candidates(id),
    total_votes      BIGINT  NOT NULL DEFAULT 0,
    online_votes     BIGINT  NOT NULL DEFAULT 0,
    physical_votes   BIGINT  NOT NULL DEFAULT 0,
    is_leading       BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (constituency_id, candidate_id)
);

CREATE INDEX idx_constituency_summaries_leading
    ON constituency_summaries (constituency_id, total_votes DESC);

-- National party list vote totals + seat allocation
CREATE TABLE party_list_national (
    party_id       UUID     PRIMARY KEY REFERENCES parties(id),
    total_votes    BIGINT   NOT NULL DEFAULT 0,
    online_votes   BIGINT   NOT NULL DEFAULT 0,
    physical_votes BIGINT   NOT NULL DEFAULT 0,
    seat_count     SMALLINT NOT NULL DEFAULT 0,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Per-province referendum results
CREATE TABLE referendum_province_summaries (
    province_id    SMALLINT PRIMARY KEY REFERENCES provinces(id),
    agree_votes    BIGINT   NOT NULL DEFAULT 0,
    disagree_votes BIGINT   NOT NULL DEFAULT 0,
    abstain_votes  BIGINT   NOT NULL DEFAULT 0,
    total_votes    BIGINT   NOT NULL DEFAULT 0,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- National referendum totals (single-row table, id always = 1)
CREATE TABLE referendum_national_summary (
    id             SMALLINT PRIMARY KEY DEFAULT 1,
    agree_votes    BIGINT   NOT NULL DEFAULT 0,
    disagree_votes BIGINT   NOT NULL DEFAULT 0,
    abstain_votes  BIGINT   NOT NULL DEFAULT 0,
    total_votes    BIGINT   NOT NULL DEFAULT 0,
    online_total   BIGINT   NOT NULL DEFAULT 0,
    physical_total BIGINT   NOT NULL DEFAULT 0,
    agree_pct      DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    disagree_pct   DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT single_row CHECK (id = 1)
);

INSERT INTO referendum_national_summary (id) VALUES (1);
