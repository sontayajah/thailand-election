-- 77 Thai provinces (PRD §6.3)
CREATE TABLE provinces (
    id                  SMALLINT     PRIMARY KEY,   -- official province code 1–77+
    name_th             VARCHAR(100) NOT NULL,
    name_en             VARCHAR(100) NOT NULL,
    region              VARCHAR(50)  NOT NULL,       -- North, Northeast, Central, East, West, South
    constituency_count  SMALLINT     NOT NULL DEFAULT 1,
    eligible_voters     INT          NOT NULL DEFAULT 0,
    svg_path_id         VARCHAR(50),                -- matches SVG <path id="...">
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_provinces_region ON provinces (region);
