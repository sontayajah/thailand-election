-- name: GetVoterByNationalIDHash :one
-- Used in verify-id flow. Never returns plaintext national_id.
SELECT
    vr.id,
    vr.province_id,
    vr.constituency_id,
    vr.registered_phone,
    vr.is_eligible
FROM voter_registry vr
WHERE vr.national_id_hash = $1;

-- name: InsertVoterSession :one
INSERT INTO voter_sessions (
    voter_registry_id,
    status,
    ip_address,
    user_agent,
    expires_at
) VALUES (
    $1,
    'otp_pending',
    $2,
    $3,
    NOW() + INTERVAL '30 minutes'
)
RETURNING id, expires_at;

-- name: GetVoterSession :one
SELECT
    vs.id,
    vs.voter_registry_id,
    vs.status,
    vs.otp_attempts,
    vs.expires_at,
    vs.completed_at,
    vr.province_id,
    vr.constituency_id
FROM voter_sessions vs
JOIN voter_registry vr ON vr.id = vs.voter_registry_id
WHERE vs.id = $1;

-- name: UpdateVoterSessionStatus :exec
UPDATE voter_sessions
SET    status = $2,
       completed_at = CASE WHEN $2 IN ('completed', 'expired') THEN NOW() ELSE completed_at END
WHERE  id = $1;

-- name: IncrementOTPAttempts :one
UPDATE voter_sessions
SET    otp_attempts = otp_attempts + 1
WHERE  id = $1
RETURNING otp_attempts;

-- name: GetVoterRightsUsed :many
-- Returns which ballot types this session has already cast
SELECT ballot_type, voted_at
FROM voter_rights_used
WHERE voter_session_id = $1;

-- name: InsertVoterRightsUsed :exec
-- ⚠️ CRITICAL: records ONLY that a ballot was cast — NO vote choice stored here
INSERT INTO voter_rights_used (voter_session_id, ballot_type)
VALUES ($1, $2);

-- name: HasVoterCastBallot :one
SELECT EXISTS (
    SELECT 1 FROM voter_rights_used
    WHERE voter_session_id = $1
      AND ballot_type = $2
) AS has_voted;

-- name: CleanupExpiredSessions :exec
UPDATE voter_sessions
SET    status = 'expired'
WHERE  status NOT IN ('completed', 'expired')
  AND  expires_at < NOW();

-- name: InsertVoterRegistry :one
-- Used by the seed/simulator scripts to create test voters
INSERT INTO voter_registry (
    national_id_hash,
    province_id,
    constituency_id,
    registered_phone,
    is_eligible
) VALUES ($1, $2, $3, $4, $5)
RETURNING id;
