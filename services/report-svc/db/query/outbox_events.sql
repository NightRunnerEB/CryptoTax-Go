-- name: InsertOutboxEvent :exec
INSERT INTO outbox_events (
  id, aggregate_type, aggregate_id, event_type, payload, status
)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListPendingOutboxEvents :many
SELECT
  id, aggregate_type, aggregate_id, event_type, payload,
  created_at, published_at, status, attempts, last_error
FROM outbox_events
WHERE (status = 'pending' OR status = 'failed') AND attempts < $1
ORDER BY created_at ASC
LIMIT $2;

-- name: MarkOutboxEventPublished :exec
UPDATE outbox_events
SET status = 'published', published_at = now(), attempts = attempts + 1, last_error = NULL
WHERE id = $1;

-- name: MarkOutboxEventFailed :exec
UPDATE outbox_events
SET
  attempts = attempts + 1,
  last_error = $3,
  status = CASE WHEN attempts + 1 >= $2 THEN 'failed' ELSE 'pending' END
WHERE id = $1;

