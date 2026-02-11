CREATE TABLE IF NOT EXISTS aggregated_transactions (
  id uuid PRIMARY KEY,
  tenant_id uuid NOT NULL,
  wallet text NOT NULL,
  source text NOT NULL,
  import_id uuid NOT NULL,
  time_utc timestamptz NOT NULL,
  kind text NOT NULL,
  in_money jsonb NULL,
  out_money jsonb NULL,
  fee_money jsonb NULL,
  contract_symbol text NULL,
  derivative_kind text NULL,
  position_id text NULL,
  order_id text NULL,
  tx_hash text NULL,
  note text NULL,
  tx_fingerprint text NOT NULL,
  created_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_aggregated_transactions_import
ON aggregated_transactions (tenant_id, import_id, time_utc DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_aggregated_transactions_fingerprint
ON aggregated_transactions (tenant_id, tx_fingerprint);
