CREATE TABLE IF NOT EXISTS tax_report_jobs (
  id uuid PRIMARY KEY,
  tenant_id uuid NOT NULL,
  tax_year int NOT NULL,
  jurisdiction text NOT NULL,
  status text NOT NULL,
  requested_at timestamptz NOT NULL DEFAULT now(),
  started_at timestamptz NULL,
  completed_at timestamptz NULL,
  error text NULL,
  params jsonb NOT NULL,
  summary jsonb NULL,
  dataset_object_key text NULL,
  pdf_object_key text NULL
);

CREATE INDEX IF NOT EXISTS idx_tax_report_jobs_tenant_year
ON tax_report_jobs (tenant_id, tax_year);

CREATE INDEX IF NOT EXISTS idx_tax_report_jobs_tenant_requested
ON tax_report_jobs (tenant_id, requested_at DESC);
