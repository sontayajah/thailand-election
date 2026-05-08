CREATE TABLE parties (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    short_name  VARCHAR(20)  NOT NULL,
    color_hex   CHAR(7)      NOT NULL,   -- e.g. '#1a73e8'
    logo_url    TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_parties_short_name ON parties (short_name);
