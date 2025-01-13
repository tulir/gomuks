-- v12 (compatible with v10+): Add table for push registrations
CREATE TABLE push_registration (
	device_id  TEXT    NOT NULL,
	type       TEXT    NOT NULL,
	data       TEXT    NOT NULL,
	encryption TEXT    NOT NULL,
	expiration INTEGER NOT NULL,

	PRIMARY KEY (device_id)
) STRICT;
