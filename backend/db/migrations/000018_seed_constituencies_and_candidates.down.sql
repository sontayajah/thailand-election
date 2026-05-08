DELETE FROM party_list_national WHERE party_id::text LIKE 'a1000000%';
DELETE FROM candidates WHERE id::text LIKE 'd0000000%' OR id::text LIKE 'e0000000%';
DELETE FROM constituencies WHERE id::text LIKE 'b0000000%' OR id::text LIKE 'c0000000%';
