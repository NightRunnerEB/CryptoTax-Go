CREATE TABLE user_symbols (
    user_id uuid NOT NULL,
    source text NOT NULL,
    symbol text NOT NULL,
    coin_id text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, source, symbol)
);

CREATE INDEX IF NOT EXISTS idx_tso_coin_id ON user_symbols (coin_id);
