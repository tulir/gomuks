-- v3 (compatible with v1+): Add more fields to events
ALTER TABLE event ADD COLUMN local_content TEXT;
ALTER TABLE event ADD COLUMN unread_type INTEGER NOT NULL DEFAULT 0;
ALTER TABLE room ADD COLUMN unread_highlights INTEGER NOT NULL DEFAULT 0;
ALTER TABLE room ADD COLUMN unread_notifications INTEGER NOT NULL DEFAULT 0;
ALTER TABLE room ADD COLUMN unread_messages INTEGER NOT NULL DEFAULT 0;
