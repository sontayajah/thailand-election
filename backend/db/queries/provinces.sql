-- name: ListProvinces :many
SELECT id, name_th, name_en, region, constituency_count, eligible_voters, svg_path_id
FROM provinces
ORDER BY id;

-- name: GetProvinceByID :one
SELECT id, name_th, name_en, region, constituency_count, eligible_voters, svg_path_id
FROM provinces
WHERE id = $1;

-- name: ListProvincesByRegion :many
SELECT id, name_th, name_en, region, constituency_count, eligible_voters, svg_path_id
FROM provinces
WHERE region = $1
ORDER BY id;

-- name: ListConstituenciesByProvince :many
SELECT id, province_id, constituency_no, name, eligible_voters
FROM constituencies
WHERE province_id = $1
ORDER BY constituency_no;

-- name: GetConstituencyByID :one
SELECT id, province_id, constituency_no, name, eligible_voters
FROM constituencies
WHERE id = $1;
