CREATE TABLE users (
    id         BIGINT PRIMARY KEY, -- telegram user id
    username   TEXT,
    first_name TEXT        NOT NULL DEFAULT '',
    last_name  TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TYPE lobby_visibility AS ENUM ('public', 'private');

CREATE TABLE lobbies (
    id         BIGSERIAL PRIMARY KEY,
    creator_id BIGINT           NOT NULL REFERENCES users (id),
    name       TEXT             NOT NULL,
    place      TEXT             NOT NULL,
    event_time TIMESTAMPTZ      NOT NULL,
    chat_link  TEXT,
    visibility lobby_visibility NOT NULL DEFAULT 'public',
    created_at TIMESTAMPTZ      NOT NULL DEFAULT now()
);

CREATE INDEX idx_lobbies_event_time ON lobbies (event_time);
CREATE INDEX idx_lobbies_creator ON lobbies (creator_id);

CREATE TYPE membership_status AS ENUM ('pending', 'approved', 'rejected');

CREATE TABLE lobby_members (
    lobby_id  BIGINT            NOT NULL REFERENCES lobbies (id) ON DELETE CASCADE,
    user_id   BIGINT            NOT NULL REFERENCES users (id),
    status    membership_status NOT NULL DEFAULT 'pending',
    joined_at TIMESTAMPTZ       NOT NULL DEFAULT now(),
    PRIMARY KEY (lobby_id, user_id)
);

CREATE INDEX idx_lobby_members_user ON lobby_members (user_id);
