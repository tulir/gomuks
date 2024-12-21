-- v0 -> v9 (compatible with v5+): Latest revision
CREATE TABLE account (
	user_id        TEXT NOT NULL PRIMARY KEY,
	device_id      TEXT NOT NULL,
	access_token   TEXT NOT NULL,
	homeserver_url TEXT NOT NULL,

	next_batch     TEXT NOT NULL
) STRICT;

CREATE TABLE room (
	room_id              TEXT    NOT NULL PRIMARY KEY,
	creation_content     TEXT,
	tombstone_content    TEXT,

	name                 TEXT,
	name_quality         INTEGER NOT NULL DEFAULT 0,
	avatar               TEXT,
	explicit_avatar      INTEGER NOT NULL DEFAULT 0,
	topic                TEXT,
	canonical_alias      TEXT,
	lazy_load_summary    TEXT,

	encryption_event     TEXT,
	has_member_list      INTEGER NOT NULL DEFAULT false,

	preview_event_rowid  INTEGER,
	sorting_timestamp    INTEGER,
	unread_highlights    INTEGER NOT NULL DEFAULT 0,
	unread_notifications INTEGER NOT NULL DEFAULT 0,
	unread_messages      INTEGER NOT NULL DEFAULT 0,
	marked_unread        INTEGER NOT NULL DEFAULT false,

	prev_batch           TEXT,

	CONSTRAINT room_preview_event_fkey FOREIGN KEY (preview_event_rowid) REFERENCES event (rowid) ON DELETE SET NULL
) STRICT;
CREATE INDEX room_type_idx ON room (creation_content ->> 'type');
CREATE INDEX room_sorting_timestamp_idx ON room (sorting_timestamp DESC);
CREATE INDEX room_preview_idx ON room (preview_event_rowid);
-- CREATE INDEX room_sorting_timestamp_idx ON room (unread_notifications > 0);
-- CREATE INDEX room_sorting_timestamp_idx ON room (unread_messages > 0);

CREATE TABLE invited_room (
	room_id      TEXT    NOT NULL PRIMARY KEY,
	received_at  INTEGER NOT NULL,
	invite_state TEXT    NOT NULL
) STRICT;

CREATE TRIGGER invited_room_delete_on_room_insert
	AFTER INSERT
	ON room
BEGIN
	DELETE FROM invited_room WHERE room_id = NEW.room_id;
END;

CREATE TABLE account_data (
	user_id TEXT NOT NULL,
	type    TEXT NOT NULL,
	content TEXT NOT NULL,

	PRIMARY KEY (user_id, type)
) STRICT;

CREATE TABLE room_account_data (
	user_id TEXT NOT NULL,
	room_id TEXT NOT NULL,
	type    TEXT NOT NULL,
	content TEXT NOT NULL,

	PRIMARY KEY (user_id, room_id, type),
	CONSTRAINT room_account_data_room_fkey FOREIGN KEY (room_id) REFERENCES room (room_id) ON DELETE CASCADE
) STRICT;
CREATE INDEX room_account_data_room_id_idx ON room_account_data (room_id);

CREATE TABLE event (
	rowid             INTEGER PRIMARY KEY,

	room_id           TEXT    NOT NULL,
	event_id          TEXT    NOT NULL,
	sender            TEXT    NOT NULL,
	type              TEXT    NOT NULL,
	state_key         TEXT,
	timestamp         INTEGER NOT NULL,

	content           TEXT    NOT NULL,
	decrypted         TEXT,
	decrypted_type    TEXT,
	unsigned          TEXT    NOT NULL,
	local_content     TEXT,

	transaction_id    TEXT,

	redacted_by       TEXT,
	relates_to        TEXT,
	relation_type     TEXT,

	megolm_session_id TEXT,
	decryption_error  TEXT,
	send_error        TEXT,

	reactions         TEXT,
	last_edit_rowid   INTEGER,
	unread_type       INTEGER NOT NULL DEFAULT 0,

	CONSTRAINT event_id_unique_key UNIQUE (event_id),
	CONSTRAINT transaction_id_unique_key UNIQUE (transaction_id),
	CONSTRAINT event_room_fkey FOREIGN KEY (room_id) REFERENCES room (room_id) ON DELETE CASCADE
) STRICT;
CREATE INDEX event_room_id_idx ON event (room_id);
CREATE INDEX event_redacted_by_idx ON event (room_id, redacted_by);
CREATE INDEX event_relates_to_idx ON event (room_id, relates_to);
CREATE INDEX event_megolm_session_id_idx ON event (room_id, megolm_session_id);

CREATE TRIGGER event_update_redacted_by
	AFTER INSERT
	ON event
	WHEN NEW.type = 'm.room.redaction'
BEGIN
	UPDATE event SET redacted_by = NEW.event_id WHERE room_id = NEW.room_id AND event_id = NEW.content ->> 'redacts';
END;

CREATE TRIGGER event_update_last_edit_when_redacted
	AFTER UPDATE
	ON event
	WHEN OLD.redacted_by IS NULL
		AND NEW.redacted_by IS NOT NULL
		AND NEW.relation_type = 'm.replace'
		AND NEW.state_key IS NULL
BEGIN
	UPDATE event
	SET last_edit_rowid = COALESCE(
		(SELECT rowid
		 FROM event edit
		 WHERE edit.room_id = event.room_id
		   AND edit.relates_to = event.event_id
		   AND edit.relation_type = 'm.replace'
		   AND edit.type = event.type
		   AND edit.sender = event.sender
		   AND edit.redacted_by IS NULL
		   AND edit.state_key IS NULL
		 ORDER BY edit.timestamp DESC
		 LIMIT 1),
		0)
	WHERE event_id = NEW.relates_to
	  AND last_edit_rowid = NEW.rowid
	  AND state_key IS NULL
	  AND (relation_type IS NULL OR relation_type NOT IN ('m.replace', 'm.annotation'));
END;

CREATE TRIGGER event_insert_update_last_edit
	AFTER INSERT
	ON event
	WHEN NEW.relation_type = 'm.replace'
		AND NEW.redacted_by IS NULL
		AND NEW.state_key IS NULL
BEGIN
	UPDATE event
	SET last_edit_rowid = NEW.rowid
	WHERE event_id = NEW.relates_to
	  AND type = NEW.type
	  AND sender = NEW.sender
	  AND state_key IS NULL
	  AND (relation_type IS NULL OR relation_type NOT IN ('m.replace', 'm.annotation'))
	  AND NEW.timestamp >
		  COALESCE((SELECT prev_edit.timestamp FROM event prev_edit WHERE prev_edit.rowid = event.last_edit_rowid), 0);
END;

CREATE TRIGGER event_insert_fill_reactions
	AFTER INSERT
	ON event
	WHEN NEW.type = 'm.reaction'
		AND NEW.relation_type = 'm.annotation'
		AND NEW.redacted_by IS NULL
		AND typeof(NEW.content ->> '$."m.relates_to".key') = 'text'
		AND NEW.content ->> '$."m.relates_to".key' NOT LIKE '%"%'
BEGIN
	UPDATE event
	SET reactions=json_set(
		reactions,
		'$.' || json_quote(NEW.content ->> '$."m.relates_to".key'),
		coalesce(
			reactions ->> ('$.' || json_quote(NEW.content ->> '$."m.relates_to".key')),
			0
		) + 1)
	WHERE event_id = NEW.relates_to
	  AND reactions IS NOT NULL;
END;

CREATE TRIGGER event_redact_fill_reactions
	AFTER UPDATE
	ON event
	WHEN NEW.type = 'm.reaction'
		AND NEW.relation_type = 'm.annotation'
		AND NEW.redacted_by IS NOT NULL
		AND OLD.redacted_by IS NULL
		AND typeof(NEW.content ->> '$."m.relates_to".key') = 'text'
		AND NEW.content ->> '$."m.relates_to".key' NOT LIKE '%"%'
BEGIN
	UPDATE event
	SET reactions=json_set(
		reactions,
		'$.' || json_quote(NEW.content ->> '$."m.relates_to".key'),
		coalesce(
			reactions ->> ('$.' || json_quote(NEW.content ->> '$."m.relates_to".key')),
			0
		) - 1)
	WHERE event_id = NEW.relates_to
	  AND reactions IS NOT NULL;
END;

CREATE TABLE media (
	mxc       TEXT NOT NULL PRIMARY KEY,
	enc_file  TEXT,
	file_name TEXT,
	mime_type TEXT,
	size      INTEGER,
	hash      BLOB,
	error     TEXT
) STRICT;

CREATE TABLE media_reference (
	event_rowid INTEGER NOT NULL,
	media_mxc   TEXT    NOT NULL,

	PRIMARY KEY (event_rowid, media_mxc),
	CONSTRAINT media_reference_event_fkey FOREIGN KEY (event_rowid) REFERENCES event (rowid) ON DELETE CASCADE,
	CONSTRAINT media_reference_media_fkey FOREIGN KEY (media_mxc) REFERENCES media (mxc) ON DELETE CASCADE
) STRICT;

CREATE TABLE session_request (
	room_id        TEXT    NOT NULL,
	session_id     TEXT    NOT NULL,
	sender         TEXT    NOT NULL,
	min_index      INTEGER NOT NULL,
	backup_checked INTEGER NOT NULL DEFAULT false,
	request_sent   INTEGER NOT NULL DEFAULT false,

	PRIMARY KEY (session_id),
	CONSTRAINT session_request_queue_room_fkey FOREIGN KEY (room_id) REFERENCES room (room_id) ON DELETE CASCADE
) STRICT;
CREATE INDEX session_request_room_idx ON session_request (room_id);

CREATE TABLE timeline (
	rowid       INTEGER PRIMARY KEY,
	room_id     TEXT    NOT NULL,
	event_rowid INTEGER NOT NULL,

	CONSTRAINT timeline_room_fkey FOREIGN KEY (room_id) REFERENCES room (room_id) ON DELETE CASCADE,
	CONSTRAINT timeline_event_fkey FOREIGN KEY (event_rowid) REFERENCES event (rowid) ON DELETE CASCADE,
	CONSTRAINT timeline_event_unique_key UNIQUE (event_rowid)
) STRICT;
CREATE INDEX timeline_room_id_idx ON timeline (room_id);

CREATE TABLE current_state (
	room_id     TEXT    NOT NULL,
	event_type  TEXT    NOT NULL,
	state_key   TEXT    NOT NULL,
	event_rowid INTEGER NOT NULL,

	membership  TEXT,

	PRIMARY KEY (room_id, event_type, state_key),
	CONSTRAINT current_state_room_fkey FOREIGN KEY (room_id) REFERENCES room (room_id) ON DELETE CASCADE,
	CONSTRAINT current_state_event_fkey FOREIGN KEY (event_rowid) REFERENCES event (rowid),
	CONSTRAINT current_state_rowid_unique UNIQUE (event_rowid)
) STRICT, WITHOUT ROWID;

CREATE TABLE receipt (
	room_id      TEXT    NOT NULL,
	user_id      TEXT    NOT NULL,
	receipt_type TEXT    NOT NULL,
	thread_id    TEXT    NOT NULL,
	event_id     TEXT    NOT NULL,
	timestamp    INTEGER NOT NULL,

	PRIMARY KEY (room_id, user_id, receipt_type, thread_id),
	CONSTRAINT receipt_room_fkey FOREIGN KEY (room_id) REFERENCES room (room_id) ON DELETE CASCADE
	-- note: there's no foreign key on event ID because receipts could point at events that are too far in history.
) STRICT;
