-- name: GetUserBySub :one
SELECT
    *
FROM
    "user"
WHERE
    sub = $1;
