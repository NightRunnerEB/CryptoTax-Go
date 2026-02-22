-- name: UpsertRenderJobProcessing :exec
INSERT INTO render_jobs (
  report_id, tenant_id, status, dataset_object_key, started_at, updated_at
)
VALUES ($1, $2, 'processing', $3, now(), now())
ON CONFLICT (report_id) DO UPDATE
SET
  tenant_id = EXCLUDED.tenant_id,
  status = 'processing',
  completed_at = NULL,
  error = NULL,
  dataset_object_key = EXCLUDED.dataset_object_key,
  updated_at = now();

-- name: MarkRenderJobCompleted :exec
UPDATE render_jobs
SET
  status = 'completed',
  completed_at = now(),
  error = NULL,
  pdf_object_key = $2,
  updated_at = now()
WHERE report_id = $1;

-- name: MarkRenderJobFailed :exec
UPDATE render_jobs
SET
  status = 'failed',
  completed_at = now(),
  error = $2,
  updated_at = now()
WHERE report_id = $1;

