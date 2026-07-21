-- Per-user language preference, defaulting to English.
ALTER TABLE users
    ADD COLUMN locale TEXT NOT NULL DEFAULT 'en';
