-- User preferences: timezone (for interpreting/displaying event times), a home
-- city (to surface nearby lobbies), and a time-window filter for the list.
ALTER TABLE users
    ADD COLUMN timezone    TEXT NOT NULL DEFAULT 'UTC',
    ADD COLUMN city        TEXT NOT NULL DEFAULT '',
    ADD COLUMN time_filter TEXT NOT NULL DEFAULT 'all';

-- Structured location for events. The old free-form "place" becomes the
-- optional address.
ALTER TABLE lobbies
    ADD COLUMN country TEXT NOT NULL DEFAULT '',
    ADD COLUMN city    TEXT NOT NULL DEFAULT '',
    ADD COLUMN address TEXT;

UPDATE lobbies
SET address = NULLIF(place, '');

ALTER TABLE lobbies
    DROP COLUMN place;

CREATE INDEX idx_lobbies_city ON lobbies (lower(city));
