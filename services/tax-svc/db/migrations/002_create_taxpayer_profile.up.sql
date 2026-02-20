CREATE TABLE IF NOT EXISTS taxpayer_profile (
  tenant_id uuid PRIMARY KEY,
  inn text NULL,
  last_name text NULL,
  first_name text NULL,
  middle_name text NULL,
  birth_date date NULL,
  document_type_code text NULL,
  document_number text NULL,
  tax_residency_status text NULL,
  phone text NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
