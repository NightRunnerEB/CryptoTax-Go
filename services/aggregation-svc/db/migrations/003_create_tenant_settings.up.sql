CREATE TABLE IF NOT EXISTS tenant_settings (
  tenant_id uuid PRIMARY KEY,
  fiat_currency text NOT NULL DEFAULT 'RUB',
  timezone text NOT NULL DEFAULT 'UTC',
  updated_at timestamptz NOT NULL DEFAULT now()
);
