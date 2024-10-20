-- v5: Refactor media cache
CREATE TABLE media (
	mxc       TEXT NOT NULL PRIMARY KEY,
	enc_file  TEXT,
	file_name TEXT,
	mime_type TEXT,
	size      INTEGER,
	hash      BLOB,
	error     TEXT
) STRICT;

INSERT INTO media (mxc, enc_file, file_name, mime_type, size, hash, error)
SELECT mxc, enc_file, file_name, mime_type, size, hash, error
FROM cached_media;

CREATE TABLE media_reference (
	event_rowid INTEGER NOT NULL,
	media_mxc   TEXT    NOT NULL,

	PRIMARY KEY (event_rowid, media_mxc),
	CONSTRAINT media_reference_event_fkey FOREIGN KEY (event_rowid) REFERENCES event (rowid) ON DELETE CASCADE,
	CONSTRAINT media_reference_media_fkey FOREIGN KEY (media_mxc) REFERENCES media (mxc) ON DELETE CASCADE
) STRICT;

INSERT INTO media_reference (event_rowid, media_mxc)
SELECT event_rowid, mxc
FROM cached_media WHERE event_rowid IS NOT NULL;

DROP TABLE cached_media;
