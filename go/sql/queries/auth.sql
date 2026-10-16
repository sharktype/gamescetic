-- name: GetUserBySub :one
SELECT
    sqlc.embed(u)
FROM
    "user" u
WHERE
    u.sub = $1;

-- name: CreateUser :one
INSERT INTO
    "user" (id, sub, name, email, avatar_url)
VALUES
    ($1, $2, $3, $4, $5)
RETURNING
    *;
