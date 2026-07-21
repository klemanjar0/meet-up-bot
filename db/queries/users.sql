-- name: UpsertUser :one
INSERT INTO users (id, username, first_name, last_name)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE
    SET username   = EXCLUDED.username,
        first_name = EXCLUDED.first_name,
        last_name  = EXCLUDED.last_name
RETURNING *;

-- name: GetUser :one
SELECT *
FROM users
WHERE id = $1;

-- name: SetUserLocale :exec
UPDATE users
SET locale = $2
WHERE id = $1;

-- name: SetUserTimezone :exec
UPDATE users
SET timezone = $2
WHERE id = $1;

-- name: SetUserCity :exec
UPDATE users
SET city = $2
WHERE id = $1;

-- name: SetUserTimeFilter :exec
UPDATE users
SET time_filter = $2
WHERE id = $1;
