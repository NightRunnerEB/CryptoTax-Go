-- name: GetAggregationImportState :one
SELECT user_id, import_id, event_id, status, started_at, completed_at, error
FROM aggregation_import_state
WHERE user_id = $1 AND import_id = $2;

-- name: UpsertAggregationImportStateProcessing :exec
INSERT INTO aggregation_import_state (user_id, import_id, event_id, status)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, import_id)
DO UPDATE SET
  event_id = EXCLUDED.event_id,
  status = EXCLUDED.status,
  started_at = now(),
  completed_at = NULL,
  error = NULL;

-- name: MarkAggregationImportStateCompleted :exec
UPDATE aggregation_import_state
SET status = 'completed', completed_at = now(), error = NULL
WHERE user_id = $1 AND import_id = $2;

-- name: MarkAggregationImportStateFailed :exec
UPDATE aggregation_import_state
SET status = 'failed', completed_at = now(), error = $3
WHERE user_id = $1 AND import_id = $2;
