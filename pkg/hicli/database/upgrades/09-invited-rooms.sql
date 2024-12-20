-- v9 (compatible with v5+): Add table for invited rooms
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
