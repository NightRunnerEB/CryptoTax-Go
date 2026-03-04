-- name: CreateTaxJob :one
INSERT INTO tax_jobs (
  id,
  tenant_id,
  tax_year,
  policy_snapshot,
  status,
  attempts,
  retry_at
)
VALUES ($1, $2, $3, $4, $5, $6, now())
RETURNING
  id,
  tenant_id,
  tax_year,
  policy_snapshot,
  status,
  attempts,
  retry_at,
  summary,
  audit_zip_url,
  report_url,
  created_at,
  started_at,
  finished_at,
  last_error_code,
  last_error_message;

-- name: GetTaxJob :one
SELECT
  id,
  tenant_id,
  tax_year,
  policy_snapshot,
  status,
  attempts,
  retry_at,
  summary,
  audit_zip_url,
  report_url,
  created_at,
  started_at,
  finished_at,
  last_error_code,
  last_error_message
FROM tax_jobs
WHERE tenant_id = $1 AND id = $2;

-- name: CountTaxJobs :one
SELECT count(*)
FROM tax_jobs
WHERE tenant_id = $1;

-- name: ListTaxJobs :many
SELECT
  id,
  tenant_id,
  tax_year,
  policy_snapshot,
  status,
  attempts,
  retry_at,
  summary,
  audit_zip_url,
  report_url,
  created_at,
  started_at,
  finished_at,
  last_error_code,
  last_error_message
FROM tax_jobs
WHERE tenant_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ClaimNextQueuedTaxJob :one
WITH next_job AS (
  SELECT id
  FROM tax_jobs
  WHERE status = 'queued' AND retry_at <= now()
  ORDER BY retry_at ASC, created_at ASC
  FOR UPDATE SKIP LOCKED
  LIMIT 1
)
UPDATE tax_jobs AS j
SET
  status = 'running',
  attempts = j.attempts + 1,
  started_at = now(),
  retry_at = now(),
  finished_at = NULL,
  last_error_code = NULL,
  last_error_message = NULL
FROM next_job
WHERE j.id = next_job.id
RETURNING
  j.id,
  j.tenant_id,
  j.tax_year,
  j.policy_snapshot,
  j.status,
  j.attempts,
  j.retry_at,
  j.summary,
  j.audit_zip_url,
  j.report_url,
  j.created_at,
  j.started_at,
  j.finished_at,
  j.last_error_code,
  j.last_error_message;

-- name: SaveTaxJobResult :exec
UPDATE tax_jobs
SET
  status = 'success',
  summary = $2,
  audit_zip_url = $3,
  report_url = $4,
  retry_at = now(),
  finished_at = now(),
  last_error_code = NULL,
  last_error_message = NULL
WHERE id = $1;

-- name: RequeueTaxJob :exec
UPDATE tax_jobs
SET
  status = 'queued',
  retry_at = $2,
  started_at = NULL,
  finished_at = NULL,
  last_error_code = $3,
  last_error_message = $4
WHERE id = $1;

-- name: MarkTaxJobFailed :exec
UPDATE tax_jobs
SET
  status = 'failed',
  retry_at = now(),
  finished_at = now(),
  last_error_code = $2,
  last_error_message = $3
WHERE id = $1;

-- name: MarkTaxJobCanceled :exec
UPDATE tax_jobs
SET
  status = 'canceled',
  finished_at = now()
WHERE id = $1;
