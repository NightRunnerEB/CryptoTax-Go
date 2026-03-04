CREATE TABLE
  IF NOT EXISTS tax_jobs (
    id uuid PRIMARY KEY,
    tenant_id uuid NOT NULL,
    tax_year int NOT NULL,
    policy_snapshot jsonb NOT NULL,
    status text NOT NULL,
    attempts int NOT NULL DEFAULT 0,
    retry_at timestamptz NOT NULL DEFAULT now (),
    summary jsonb NULL,
    audit_zip_url text NULL,
    report_url text NULL,
    created_at timestamptz NOT NULL DEFAULT now (),
    started_at timestamptz NULL,
    finished_at timestamptz NULL,
    last_error_code text NULL,
    last_error_message text NULL
  );

CREATE INDEX IF NOT EXISTS idx_tax_jobs_tenant_created_at ON tax_jobs (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_tax_jobs_status_created_at ON tax_jobs (status, created_at ASC);

CREATE INDEX IF NOT EXISTS idx_tax_jobs_status_retry_at_created_at ON tax_jobs (status, retry_at ASC, created_at ASC);
