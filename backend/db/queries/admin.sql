-- name: GetAdminByUsername :one
SELECT id, username, password_hash, role, is_active
FROM admin_users
WHERE username = $1
  AND is_active = TRUE;

-- name: UpdateAdminLastLogin :exec
UPDATE admin_users
SET last_login_at = NOW()
WHERE id = $1;

-- name: InsertAuditLog :exec
INSERT INTO audit_logs (admin_id, action, target_type, target_id, ip_address, user_agent, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListAuditLogs :many
SELECT
    al.id,
    al.admin_id,
    au.username AS admin_username,
    al.action,
    al.target_type,
    al.target_id,
    al.ip_address,
    al.created_at
FROM audit_logs al
LEFT JOIN admin_users au ON au.id = al.admin_id
ORDER BY al.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetElectionConfig :one
SELECT value FROM election_config WHERE key = $1;

-- name: SetElectionConfig :exec
INSERT INTO election_config (key, value)
VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value, updated_at = NOW();

-- name: ListElectionConfig :many
SELECT key, value, updated_at FROM election_config ORDER BY key;
