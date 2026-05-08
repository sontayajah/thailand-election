-- name: InsertVoteEvent :one
INSERT INTO vote_events (
    ballot_type,
    source,
    province_id,
    constituency_id,
    candidate_id,
    referendum_vote,
    vote_count,
    anonymous_token,
    idempotency_key,
    payload_signature
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING id, created_at;

-- name: GetVoteEventByIdempotencyKey :one
SELECT id, ballot_type, source, created_at
FROM vote_events
WHERE idempotency_key = $1;

-- name: GetVoteEventsByProvince :many
SELECT
    id,
    ballot_type,
    source,
    candidate_id,
    referendum_vote,
    vote_count,
    created_at
FROM vote_events
WHERE province_id = $1
  AND ballot_type = $2
ORDER BY created_at DESC
LIMIT $3;

-- name: SumVotesByCandidate :many
SELECT
    candidate_id,
    SUM(vote_count)::BIGINT AS total_votes,
    SUM(CASE WHEN source = 'online' THEN vote_count ELSE 0 END)::BIGINT AS online_votes,
    SUM(CASE WHEN source != 'online' THEN vote_count ELSE 0 END)::BIGINT AS physical_votes
FROM vote_events
WHERE ballot_type = 'CONSTITUENCY'
  AND constituency_id = $1
GROUP BY candidate_id;

-- name: SumVotesByParty :many
SELECT
    candidate_id AS party_candidate_id,
    SUM(vote_count)::BIGINT AS total_votes,
    SUM(CASE WHEN source = 'online' THEN vote_count ELSE 0 END)::BIGINT AS online_votes,
    SUM(CASE WHEN source != 'online' THEN vote_count ELSE 0 END)::BIGINT AS physical_votes
FROM vote_events
WHERE ballot_type = 'PARTY_LIST'
GROUP BY candidate_id;

-- name: SumReferendumVotesNational :one
SELECT
    SUM(CASE WHEN referendum_vote = 'AGREE'    THEN vote_count ELSE 0 END)::BIGINT AS agree_votes,
    SUM(CASE WHEN referendum_vote = 'DISAGREE' THEN vote_count ELSE 0 END)::BIGINT AS disagree_votes,
    SUM(CASE WHEN referendum_vote = 'ABSTAIN'  THEN vote_count ELSE 0 END)::BIGINT AS abstain_votes,
    SUM(vote_count)::BIGINT AS total_votes
FROM vote_events
WHERE ballot_type = 'REFERENDUM';

-- name: SumReferendumVotesByProvince :one
SELECT
    SUM(CASE WHEN referendum_vote = 'AGREE'    THEN vote_count ELSE 0 END)::BIGINT AS agree_votes,
    SUM(CASE WHEN referendum_vote = 'DISAGREE' THEN vote_count ELSE 0 END)::BIGINT AS disagree_votes,
    SUM(CASE WHEN referendum_vote = 'ABSTAIN'  THEN vote_count ELSE 0 END)::BIGINT AS abstain_votes,
    SUM(vote_count)::BIGINT AS total_votes
FROM vote_events
WHERE ballot_type = 'REFERENDUM'
  AND province_id = $1;

-- name: GetProvinceKey :one
SELECT province_id, public_key_pem, fingerprint, expires_at
FROM province_keys
WHERE province_id = $1
  AND is_active = TRUE
  AND expires_at > NOW();
