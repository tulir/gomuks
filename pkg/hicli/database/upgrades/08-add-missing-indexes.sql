-- v8 (compatible with v5+): Add indexes necessary for fast room deletion
CREATE INDEX room_preview_idx ON room (preview_event_rowid);
CREATE UNIQUE INDEX current_state_rowid_unique ON current_state (event_rowid);
