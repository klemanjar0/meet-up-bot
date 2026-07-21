-- Add a distinct "banned" membership status. Unlike removing a member (which
-- deletes the row and lets them rejoin), a ban keeps a row that blocks the user
-- from joining again.
ALTER TYPE membership_status ADD VALUE IF NOT EXISTS 'banned';
