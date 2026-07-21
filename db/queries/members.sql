-- name: AddMember :one
INSERT INTO lobby_members (lobby_id, user_id, status)
VALUES ($1, $2, $3)
ON CONFLICT (lobby_id, user_id) DO UPDATE
    SET status = EXCLUDED.status
RETURNING *;

-- name: GetMember :one
SELECT *
FROM lobby_members
WHERE lobby_id = $1
  AND user_id = $2;

-- name: UpdateMemberStatus :one
UPDATE lobby_members
SET status = $3
WHERE lobby_id = $1
  AND user_id = $2
RETURNING *;

-- name: RemoveMember :exec
DELETE
FROM lobby_members
WHERE lobby_id = $1
  AND user_id = $2;

-- name: ListApprovedMembers :many
SELECT *
FROM lobby_members
WHERE lobby_id = $1
  AND status = 'approved'
ORDER BY joined_at ASC;

-- name: ListPendingMembers :many
SELECT *
FROM lobby_members
WHERE lobby_id = $1
  AND status = 'pending'
ORDER BY joined_at ASC;

-- name: CountApprovedMembers :one
SELECT count(*)
FROM lobby_members
WHERE lobby_id = $1
  AND status = 'approved';
