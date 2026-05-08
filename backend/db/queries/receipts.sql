-- name: InsertVoteReceipt :one
-- Linked to anonymous_token — NOT to voter identity (PRD §9.2)
INSERT INTO vote_receipts (receipt_hash, anonymous_token, ballot_type)
VALUES ($1, $2, $3)
RETURNING id, issued_at;

-- name: GetVoteReceiptByHash :one
-- Public endpoint — only confirms vote exists, never reveals choice
SELECT receipt_hash, ballot_type, issued_at
FROM vote_receipts
WHERE receipt_hash = $1;

-- name: GetVoteReceiptByAnonymousToken :many
SELECT receipt_hash, ballot_type, issued_at
FROM vote_receipts
WHERE anonymous_token = $1;
