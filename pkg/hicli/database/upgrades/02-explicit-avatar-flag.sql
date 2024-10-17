-- v2 (compatible with v1+): Add explicit avatar flag to rooms
ALTER TABLE room ADD COLUMN explicit_avatar INTEGER NOT NULL DEFAULT 0;
