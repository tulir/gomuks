-- v14 (compatible with v10+): Remove nulls from room types
UPDATE room SET room_type='' WHERE room_type IS NULL;
