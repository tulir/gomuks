-- v10 (compatible with v10+): Add support for spaces
ALTER TABLE room ADD COLUMN room_type TEXT;
UPDATE room SET room_type=COALESCE(creation_content->>'$.type', '');
DROP INDEX room_type_idx;
CREATE INDEX room_type_idx ON room (room_type);

CREATE TABLE space_edge (
	space_id           TEXT    NOT NULL,
	child_id           TEXT    NOT NULL,
	depth              INTEGER,

	-- m.space.child fields
	child_event_rowid  INTEGER,
	"order"            TEXT    NOT NULL DEFAULT '',
	suggested          INTEGER NOT NULL DEFAULT false CHECK ( suggested IN (false, true) ),
	-- m.space.parent fields
	parent_event_rowid INTEGER,
	canonical          INTEGER NOT NULL DEFAULT false CHECK ( canonical IN (false, true) ),
	parent_validated   INTEGER NOT NULL DEFAULT false CHECK ( parent_validated IN (false, true) ),

	PRIMARY KEY (space_id, child_id),
	CONSTRAINT space_edge_child_event_fkey FOREIGN KEY (child_event_rowid) REFERENCES event (rowid) ON DELETE CASCADE,
	CONSTRAINT space_edge_parent_event_fkey FOREIGN KEY (parent_event_rowid) REFERENCES event (rowid) ON DELETE CASCADE,
	CONSTRAINT space_edge_child_event_unique UNIQUE (child_event_rowid),
	CONSTRAINT space_edge_parent_event_unique UNIQUE (parent_event_rowid)
) STRICT;
CREATE INDEX space_edge_child_idx ON space_edge (child_id);

INSERT INTO space_edge (space_id, child_id, child_event_rowid, "order", suggested)
SELECT
	event.room_id,
	event.state_key,
	event.rowid,
	CASE WHEN typeof(content->>'$.order')='TEXT' THEN content->>'$.order' ELSE '' END,
	CASE WHEN json_type(content, '$.suggested') IN ('true', 'false') THEN content->>'$.suggested' ELSE false END
FROM current_state
	INNER JOIN event ON current_state.event_rowid = event.rowid
	LEFT JOIN room ON current_state.room_id = room.room_id
WHERE type = 'm.space.child'
	AND json_array_length(event.content, '$.via') > 0
	AND event.state_key LIKE '!%'
	AND (room.room_id IS NULL OR room.room_type = 'm.space');

INSERT INTO space_edge (space_id, child_id, parent_event_rowid, canonical)
SELECT
	event.state_key,
	event.room_id,
	event.rowid,
	CASE WHEN json_type(content, '$.canonical') IN ('true', 'false') THEN content->>'$.canonical' ELSE false END
FROM current_state
	INNER JOIN event ON current_state.event_rowid = event.rowid
	LEFT JOIN room ON event.state_key = room.room_id
WHERE type = 'm.space.parent'
	AND json_array_length(event.content, '$.via') > 0
	AND event.state_key LIKE '!%'
	AND (room.room_id IS NULL OR room.room_type = 'm.space')
ON CONFLICT (space_id, child_id) DO UPDATE
	SET parent_event_rowid = excluded.parent_event_rowid,
	    canonical = excluded.canonical;

UPDATE space_edge
SET parent_validated=(SELECT EXISTS(
	SELECT 1
	FROM room
		INNER JOIN current_state cs ON cs.room_id = room.room_id AND cs.event_type = 'm.room.power_levels' AND cs.state_key = ''
		INNER JOIN event pls ON cs.event_rowid = pls.rowid
		INNER JOIN event edgeevt ON space_edge.parent_event_rowid = edgeevt.rowid
	WHERE	room.room_id = space_edge.space_id
		AND room.room_type = 'm.space'
		AND COALESCE(
			(
				SELECT value
				FROM json_each(pls.content, '$.users')
				WHERE key=edgeevt.sender AND type='integer'
			),
			pls.content->>'$.users_default',
			0
		) >= COALESCE(
			pls.content->>'$.events."m.space.child"',
			pls.content->>'$.state_default',
			50
		)
))
WHERE parent_event_rowid IS NOT NULL;

WITH RECURSIVE
	top_level_spaces AS (
		SELECT space_id
		FROM (SELECT DISTINCT(space_id) FROM space_edge) outeredge
		WHERE NOT EXISTS(
			SELECT 1
			FROM space_edge inneredge
			INNER JOIN room ON inneredge.space_id = room.room_id
			WHERE inneredge.child_id=outeredge.space_id
				AND (inneredge.child_event_rowid IS NOT NULL OR inneredge.parent_validated)
		)
	),
	children AS (
		SELECT space_id, child_id, 1 AS depth, space_id AS path
		FROM space_edge
		WHERE space_id IN top_level_spaces AND (child_event_rowid IS NOT NULL OR parent_validated)
		UNION
		SELECT se.space_id, se.child_id, c.depth+1, c.path || se.space_id
		FROM space_edge se
			INNER JOIN children c ON se.space_id=c.child_id
		WHERE instr(c.path, se.space_id)=0
		  AND c.depth < 10
		  AND (child_event_rowid IS NOT NULL OR parent_validated)
	)
UPDATE space_edge
SET depth = c.depth
FROM children c
WHERE space_edge.space_id = c.space_id AND space_edge.child_id = c.child_id;
