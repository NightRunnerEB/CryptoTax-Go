CREATE TABLE IF NOT EXISTS tax_profile (
  tenant_id uuid PRIMARY KEY,
  jurisdiction text NOT NULL DEFAULT 'RU',
  cost_basis_method text NOT NULL DEFAULT 'FIFO',
  timezone text NOT NULL DEFAULT 'UTC',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
