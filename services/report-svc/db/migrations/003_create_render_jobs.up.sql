CREATE TABLE IF NOT EXISTS render_jobs (
  report_id uuid PRIMARY KEY,
  tenant_id uuid NOT NULL,
  status text NOT NULL,
  started_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz NULL,
  error text NULL,
  dataset_object_key text NOT NULL,
  pdf_object_key text NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_render_jobs_tenant_started
ON render_jobs (tenant_id, started_at DESC);

