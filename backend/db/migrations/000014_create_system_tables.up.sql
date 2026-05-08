-- ── Election Config (key-value store for system settings) ────
CREATE TABLE election_config (
    key        VARCHAR(100) PRIMARY KEY,
    value      TEXT         NOT NULL,
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ── Admin Users ───────────────────────────────────────────────
CREATE TABLE admin_users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(100) NOT NULL,
    password_hash TEXT         NOT NULL,   -- argon2id
    role          VARCHAR(50)  NOT NULL DEFAULT 'admin',
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_admin_users_username ON admin_users (username);

-- ── Audit Log (all privileged actions) ───────────────────────
CREATE TABLE audit_logs (
    id          BIGSERIAL    PRIMARY KEY,
    admin_id    UUID         REFERENCES admin_users(id),
    action      VARCHAR(100) NOT NULL,
    target_type VARCHAR(50),
    target_id   VARCHAR(100),
    ip_address  INET         NOT NULL,
    user_agent  TEXT,
    metadata    JSONB,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_admin     ON audit_logs (admin_id, created_at DESC);
CREATE INDEX idx_audit_logs_action    ON audit_logs (action, created_at DESC);
CREATE INDEX idx_audit_logs_created   ON audit_logs (created_at DESC);
