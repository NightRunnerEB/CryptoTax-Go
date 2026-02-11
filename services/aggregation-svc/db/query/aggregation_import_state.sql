-- name: GetAggregationImportState :one
SELECT tenant_id, import_id, source, status, started_at, completed_at, error
FROM aggregation_import_state
WHERE tenant_id = $1 AND import_id = $2;

-- name: UpsertAggregationImportStateProcessing :exec
INSERT INTO aggregation_import_state (tenant_id, import_id, source, status)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tenant_id, import_id)
DO UPDATE SET
  source = EXCLUDED.source,
  status = EXCLUDED.status,
  started_at = now(),
  completed_at = NULL,
  error = NULL;

-- name: MarkAggregationImportStateCompleted :exec
UPDATE aggregation_import_state
SET status = 'completed', completed_at = now(), error = NULL
WHERE tenant_id = $1 AND import_id = $2;

-- name: MarkAggregationImportStateFailed :exec
UPDATE aggregation_import_state
SET status = 'failed', completed_at = now(), error = $3
WHERE tenant_id = $1 AND import_id = $2;
