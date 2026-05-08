-- name: UpsertConstituencySummary :exec
INSERT INTO constituency_summaries (
    constituency_id,
    candidate_id,
    total_votes,
    online_votes,
    physical_votes,
    updated_at
) VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (constituency_id, candidate_id)
DO UPDATE SET
    total_votes    = constituency_summaries.total_votes    + EXCLUDED.total_votes,
    online_votes   = constituency_summaries.online_votes   + EXCLUDED.online_votes,
    physical_votes = constituency_summaries.physical_votes + EXCLUDED.physical_votes,
    updated_at     = NOW();

-- name: SetConstituencyLeader :exec
UPDATE constituency_summaries
SET    is_leading = (candidate_id = $2),
       updated_at = NOW()
WHERE  constituency_id = $1;

-- name: GetConstituencySummary :many
SELECT
    cs.candidate_id,
    cs.total_votes,
    cs.online_votes,
    cs.physical_votes,
    cs.is_leading,
    c.full_name,
    c.ballot_number,
    c.photo_url,
    p.id         AS party_id,
    p.name       AS party_name,
    p.short_name AS party_short_name,
    p.color_hex  AS party_color
FROM constituency_summaries cs
JOIN candidates c ON c.id = cs.candidate_id
JOIN parties    p ON p.id = c.party_id
WHERE cs.constituency_id = $1
ORDER BY cs.total_votes DESC;

-- name: UpsertPartyListNational :exec
INSERT INTO party_list_national (
    party_id,
    total_votes,
    online_votes,
    physical_votes,
    updated_at
) VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (party_id)
DO UPDATE SET
    total_votes    = party_list_national.total_votes    + EXCLUDED.total_votes,
    online_votes   = party_list_national.online_votes   + EXCLUDED.online_votes,
    physical_votes = party_list_national.physical_votes + EXCLUDED.physical_votes,
    updated_at     = NOW();

-- name: SetPartyListSeats :exec
UPDATE party_list_national
SET    seat_count = $2,
       updated_at = NOW()
WHERE  party_id = $1;

-- name: GetPartyListNational :many
SELECT
    pln.party_id,
    pln.total_votes,
    pln.online_votes,
    pln.physical_votes,
    pln.seat_count,
    p.name       AS party_name,
    p.short_name AS party_short_name,
    p.color_hex  AS party_color
FROM party_list_national pln
JOIN parties p ON p.id = pln.party_id
ORDER BY pln.total_votes DESC;

-- name: UpsertReferendumProvinceSummary :exec
INSERT INTO referendum_province_summaries (
    province_id,
    agree_votes,
    disagree_votes,
    abstain_votes,
    total_votes,
    updated_at
) VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (province_id)
DO UPDATE SET
    agree_votes    = referendum_province_summaries.agree_votes    + EXCLUDED.agree_votes,
    disagree_votes = referendum_province_summaries.disagree_votes + EXCLUDED.disagree_votes,
    abstain_votes  = referendum_province_summaries.abstain_votes  + EXCLUDED.abstain_votes,
    total_votes    = referendum_province_summaries.total_votes    + EXCLUDED.total_votes,
    updated_at     = NOW();

-- name: GetReferendumProvinceSummary :one
SELECT province_id, agree_votes, disagree_votes, abstain_votes, total_votes, updated_at
FROM referendum_province_summaries
WHERE province_id = $1;

-- name: UpdateReferendumNationalSummary :exec
UPDATE referendum_national_summary
SET
    agree_votes    = $1,
    disagree_votes = $2,
    abstain_votes  = $3,
    total_votes    = $4,
    online_total   = $5,
    physical_total = $6,
    agree_pct      = CASE WHEN $4 > 0 THEN ROUND(($1::DECIMAL / $4) * 100, 2) ELSE 0 END,
    disagree_pct   = CASE WHEN $4 > 0 THEN ROUND(($2::DECIMAL / $4) * 100, 2) ELSE 0 END,
    updated_at     = NOW()
WHERE id = 1;

-- name: GetReferendumNationalSummary :one
SELECT agree_votes, disagree_votes, abstain_votes, total_votes,
       online_total, physical_total,
       agree_pct::FLOAT8    AS agree_pct,
       disagree_pct::FLOAT8 AS disagree_pct,
       updated_at
FROM referendum_national_summary
WHERE id = 1;

-- name: UpsertProvinceSummary :exec
-- For REFERENDUM rows: pass '00000000-0000-0000-0000-000000000000' as candidate_id
INSERT INTO province_summaries (province_id, ballot_type, candidate_id, total_votes, updated_at)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (province_id, ballot_type, candidate_id)
DO UPDATE SET
    total_votes = province_summaries.total_votes + EXCLUDED.total_votes,
    updated_at  = NOW();

-- name: GetProvinceSummary :many
-- candidate_id = '00000000-...' for REFERENDUM rows (no candidate)
SELECT
    ps.ballot_type,
    ps.candidate_id,
    ps.total_votes,
    c.full_name,
    p.id         AS party_id,
    p.name       AS party_name,
    p.short_name AS party_short_name,
    p.color_hex  AS party_color
FROM province_summaries ps
LEFT JOIN candidates c ON c.id = ps.candidate_id
                       AND ps.candidate_id != '00000000-0000-0000-0000-000000000000'
LEFT JOIN parties    p ON p.id = c.party_id
WHERE ps.province_id = $1
  AND ps.ballot_type = $2
ORDER BY ps.total_votes DESC;

-- name: GetNationalLeaderboard :many
-- Top parties by constituency wins + party list votes combined
SELECT
    p.id,
    p.name,
    p.short_name,
    p.color_hex,
    COALESCE(pln.total_votes, 0) AS party_list_votes,
    COALESCE(pln.seat_count,  0) AS party_list_seats
FROM parties p
LEFT JOIN party_list_national pln ON pln.party_id = p.id
ORDER BY pln.total_votes DESC NULLS LAST;
