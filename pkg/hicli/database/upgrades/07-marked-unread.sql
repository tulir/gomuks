-- v7 (compatible with v5+): Add room column for marking unread
ALTER TABLE room ADD COLUMN marked_unread INTEGER NOT NULL DEFAULT false;
