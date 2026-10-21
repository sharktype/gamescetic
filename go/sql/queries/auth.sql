-- name: GetUserBySub :one
SELECT
    id, sub, name, email, avatar_url, created_at, updated_at
FROM
    "user"
WHERE
    sub = $1;
