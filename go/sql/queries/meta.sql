
-- name: SetMeta :exec
INSERT INTO "meta" (key, value)
VALUES ($1, $2)
ON CONFLICT (key)
    DO UPDATE SET value = EXCLUDED.value;

-- name: GetMeta :one
SELECT value
FROM "meta"
WHERE key = $1;
