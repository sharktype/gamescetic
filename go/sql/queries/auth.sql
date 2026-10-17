-- name: GetUserBySub :one
SELECT
    sqlc.embed(u)
FROM
    "user" u
WHERE
    u.sub = $1;
