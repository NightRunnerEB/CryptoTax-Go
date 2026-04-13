CREATE TABLE IF NOT EXISTS aggregation_import_state (
  user_id uuid NOT NULL,
  import_id uuid NOT NULL,
  event_id uuid NOT NULL,
  status text NOT NULL,
  started_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz NULL,
  error text NULL,
  PRIMARY KEY (user_id, import_id)
);

CREATE INDEX IF NOT EXISTS idx_aggregation_import_state_status
ON aggregation_import_state (user_id, status);
