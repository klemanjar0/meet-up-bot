DROP INDEX IF EXISTS idx_lobbies_city;

ALTER TABLE lobbies
    ADD COLUMN place TEXT NOT NULL DEFAULT '';

UPDATE lobbies
SET place = COALESCE(address, '');

ALTER TABLE lobbies
    DROP COLUMN country,
    DROP COLUMN city,
    DROP COLUMN address;

ALTER TABLE users
    DROP COLUMN timezone,
    DROP COLUMN city,
    DROP COLUMN time_filter;
