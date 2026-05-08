-- name: ListParties :many
SELECT id, name, short_name, color_hex, logo_url
FROM parties
ORDER BY name;

-- name: GetPartyByID :one
SELECT id, name, short_name, color_hex, logo_url
FROM parties
WHERE id = $1;

-- name: ListCandidatesByConstituency :many
SELECT
    c.id,
    c.full_name,
    c.ballot_number,
    c.photo_url,
    c.party_id,
    p.name  AS party_name,
    p.short_name AS party_short_name,
    p.color_hex  AS party_color
FROM candidates c
JOIN parties p ON p.id = c.party_id
WHERE c.constituency_id = $1
  AND c.ballot_type = 'CONSTITUENCY'
  AND c.is_active = TRUE
ORDER BY c.ballot_number;

-- name: ListPartyListCandidates :many
SELECT
    c.id,
    c.full_name,
    c.party_list_order,
    c.photo_url,
    c.party_id,
    p.name       AS party_name,
    p.short_name AS party_short_name,
    p.color_hex  AS party_color
FROM candidates c
JOIN parties p ON p.id = c.party_id
WHERE c.ballot_type = 'PARTY_LIST'
  AND c.is_active = TRUE
ORDER BY p.name, c.party_list_order;

-- name: GetCandidateByID :one
SELECT
    c.id,
    c.full_name,
    c.ballot_type,
    c.ballot_number,
    c.party_list_order,
    c.photo_url,
    c.party_id,
    c.constituency_id,
    p.name       AS party_name,
    p.short_name AS party_short_name,
    p.color_hex  AS party_color
FROM candidates c
JOIN parties p ON p.id = c.party_id
WHERE c.id = $1;
