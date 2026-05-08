-- Active authentication sessions (PRD §6.3)
-- Lifetime: otp_pending → authenticated → voting → completed | expired
CREATE TABLE voter_sessions (
    id                UUID                  PRIMARY KEY DEFAULT gen_random_uuid(),
    voter_registry_id UUID                  NOT NULL REFERENCES voter_registry(id),
    status            voter_session_status  NOT NULL DEFAULT 'otp_pending',
    otp_attempts      SMALLINT              NOT NULL DEFAULT 0,
    ip_address        INET                  NOT NULL,
    user_agent        TEXT,
    created_at        TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    expires_at        TIMESTAMPTZ           NOT NULL DEFAULT (NOW() + INTERVAL '30 minutes'),
    completed_at      TIMESTAMPTZ
);

CREATE INDEX idx_voter_sessions_registry
    ON voter_sessions (voter_registry_id, status);

-- Efficient cleanup of expired sessions (W-12 asynq job)
CREATE INDEX idx_voter_sessions_expires
    ON voter_sessions (expires_at) WHERE status NOT IN ('completed', 'expired');
