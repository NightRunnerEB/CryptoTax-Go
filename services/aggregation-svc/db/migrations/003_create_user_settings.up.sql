CREATE TABLE IF NOT EXISTS user_settings (
  user_id uuid PRIMARY KEY,
  fiat_currency text NOT NULL DEFAULT 'RUB',
  timezone text NOT NULL DEFAULT 'UTC',
  updated_at timestamptz NOT NULL DEFAULT now()
);
