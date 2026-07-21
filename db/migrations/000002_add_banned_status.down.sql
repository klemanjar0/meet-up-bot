-- Postgres cannot drop a single enum value, so rebuild the type without
-- 'banned'. Any banned rows fall back to 'rejected'.
ALTER TABLE lobby_members
    ALTER COLUMN status DROP DEFAULT;

UPDATE lobby_members
SET status = 'rejected'
WHERE status = 'banned';

ALTER TYPE membership_status RENAME TO membership_status_old;

CREATE TYPE membership_status AS ENUM ('pending', 'approved', 'rejected');

ALTER TABLE lobby_members
    ALTER COLUMN status TYPE membership_status
        USING status::text::membership_status;

ALTER TABLE lobby_members
    ALTER COLUMN status SET DEFAULT 'pending';

DROP TYPE membership_status_old;
