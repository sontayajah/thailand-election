-- Ed25519 public keys per polling station (PRD §9.3 Point 2)
-- Used to verify payload signatures from physical vote submissions
CREATE TABLE province_keys (
    province_id   SMALLINT     PRIMARY KEY REFERENCES provinces(id),
    public_key_pem TEXT        NOT NULL,
    issued_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMPTZ  NOT NULL,
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    fingerprint   VARCHAR(64)  NOT NULL  -- SHA-256 of the public key bytes
);

CREATE INDEX idx_province_keys_active ON province_keys (province_id) WHERE is_active = TRUE;
