CREATE TABLE day_resolution (
  coin_id text NOT NULL,
  day_utc date NOT NULL,
  granularity_seconds int NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (coin_id, day_utc)
);
