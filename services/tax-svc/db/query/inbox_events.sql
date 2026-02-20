-- name: InsertInboxEvent :execrows
INSERT INTO inbox_events (event_id)
VALUES ($1)
ON CONFLICT (event_id) DO NOTHING;
