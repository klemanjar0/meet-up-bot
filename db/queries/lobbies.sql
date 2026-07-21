-- name: CreateLobby :one
INSERT INTO lobbies (creator_id, name, description, country, city, address, event_time, chat_link, visibility)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: GetLobby :one
SELECT *
FROM lobbies
WHERE id = $1;

-- name: DeleteLobby :exec
-- Members cascade-delete via the lobby_members foreign key.
DELETE
FROM lobbies
WHERE id = $1
  AND creator_id = $2;

-- name: UpdateLobby :one
UPDATE lobbies
SET name        = $2,
    description  = $3,
    country     = $4,
    city        = $5,
    address     = $6,
    event_time  = $7,
    chat_link   = $8,
    visibility  = $9
WHERE id = $1
  AND creator_id = $10
RETURNING *;

-- name: ListLobbiesFiltered :many
-- Upcoming lobbies, optionally narrowed to a city (case-insensitive) and to a
-- time window (events at or before @until), soonest first, paginated. Pass an
-- empty city to skip the city filter and a NULL until to skip the time filter.
SELECT *
FROM lobbies
WHERE event_time > now()
  AND (sqlc.arg(city)::text = '' OR lower(city) = lower(sqlc.arg(city)::text))
  AND (sqlc.narg(until)::timestamptz IS NULL OR event_time <= sqlc.narg(until)::timestamptz)
ORDER BY event_time ASC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: ListLobbiesByCreator :many
SELECT *
FROM lobbies
WHERE creator_id = $1
ORDER BY event_time ASC;

-- name: ListMyLobbies :many
-- Every lobby the user takes part in: ones they created, plus ones they were
-- approved to join.
SELECT *
FROM lobbies l
WHERE l.creator_id = $1
   OR l.id IN (SELECT lobby_id
               FROM lobby_members
               WHERE user_id = $1
                 AND status = 'approved')
ORDER BY l.event_time ASC;
