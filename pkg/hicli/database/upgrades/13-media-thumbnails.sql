-- v13 (compatible with v10+): Add columns for media thumbnails
ALTER TABLE media ADD COLUMN thumbnail_size INTEGER;
ALTER TABLE media ADD COLUMN thumbnail_hash BLOB;
ALTER TABLE media ADD COLUMN thumbnail_error TEXT;
