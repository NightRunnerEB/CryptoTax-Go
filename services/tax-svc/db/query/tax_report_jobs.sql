-- name: CreateTaxReportJob :one
INSERT INTO tax_report_jobs (
  id, tenant_id, tax_year, jurisdiction, status, params
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING
  id, tenant_id, tax_year, jurisdiction, status, requested_at, started_at, completed_at,
  error, params, summary, dataset_object_key, pdf_object_key;

-- name: GetTaxReportJob :one
SELECT
  id, tenant_id, tax_year, jurisdiction, status, requested_at, started_at, completed_at,
  error, params, summary, dataset_object_key, pdf_object_key
FROM tax_report_jobs
WHERE tenant_id = $1 AND id = $2;

-- name: CountTaxReportJobs :one
SELECT count(*)
FROM tax_report_jobs
WHERE tenant_id = $1 AND ($2 = 0 OR tax_year = $2);

-- name: ListTaxReportJobs :many
SELECT
  id, tenant_id, tax_year, jurisdiction, status, requested_at, started_at, completed_at,
  error, params, summary, dataset_object_key, pdf_object_key
FROM tax_report_jobs
WHERE tenant_id = $1 AND ($2 = 0 OR tax_year = $2)
ORDER BY requested_at DESC
LIMIT $3 OFFSET $4;

-- name: MarkTaxReportJobProcessing :execrows
UPDATE tax_report_jobs
SET status = 'processing', started_at = now(), completed_at = NULL, error = NULL
WHERE id = $1 AND status = 'queued';

-- name: UpdateTaxReportJobDataset :exec
UPDATE tax_report_jobs
SET dataset_object_key = $2, summary = $3
WHERE id = $1;

-- name: MarkTaxReportJobCompleted :exec
UPDATE tax_report_jobs
SET
  status = 'completed',
  completed_at = now(),
  error = NULL,
  pdf_object_key = $2
WHERE id = $1;

-- name: MarkTaxReportJobFailed :exec
UPDATE tax_report_jobs
SET
  status = 'failed',
  completed_at = now(),
  error = $2
WHERE id = $1;
