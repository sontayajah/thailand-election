-- Electoral constituencies within each province
CREATE TABLE constituencies (
    id               UUID     PRIMARY KEY DEFAULT gen_random_uuid(),
    province_id      SMALLINT NOT NULL REFERENCES provinces(id),
    constituency_no  SMALLINT NOT NULL,
    name             VARCHAR(150) NOT NULL,
    eligible_voters  INT      NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (province_id, constituency_no)
);

CREATE INDEX idx_constituencies_province ON constituencies (province_id);
