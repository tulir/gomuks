-- v6 (compatible with v5+): Add hack for reaction aggregation
DROP TRIGGER event_insert_fill_reactions;
DROP TRIGGER event_redact_fill_reactions;

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
