-- v11 (compatible with v10+): Store direct chat user ID in database
ALTER TABLE room ADD COLUMN dm_user_id TEXT;
WITH dm_user_ids AS (
	SELECT room_id, value
	FROM room
	INNER JOIN json_each(lazy_load_summary, '$."m.heroes"')
	WHERE value NOT IN (SELECT value FROM json_each((
		SELECT event.content
		FROM current_state cs
		INNER JOIN event ON cs.event_rowid = event.rowid
		WHERE cs.room_id=room.room_id AND cs.event_type='io.element.functional_members' AND cs.state_key=''
	), '$.service_members'))
	GROUP BY room_id
	HAVING COUNT(*) = 1
)
UPDATE room
SET dm_user_id=value
FROM dm_user_ids du
WHERE room.room_id=du.room_id;
