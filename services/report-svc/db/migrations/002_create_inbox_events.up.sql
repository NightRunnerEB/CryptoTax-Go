CREATE TABLE IF NOT EXISTS inbox_events (
  event_id uuid PRIMARY KEY,
  processed_at timestamptz NOT NULL DEFAULT now()
);

