-- v4 (compatible with v1+): Store tombstone event in room table
ALTER TABLE room ADD COLUMN tombstone_content TEXT;
UPDATE room SET tombstone_content=(
	SELECT content FROM event WHERE type='m.room.tombstone' AND state_key='' AND event.room_id=room.room_id
);
