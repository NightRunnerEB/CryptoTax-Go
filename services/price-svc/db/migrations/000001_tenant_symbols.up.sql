CREATE TABLE tenant_symbols (
    tenant_id uuid NOT NULL,
    source text NOT NULL,
    symbol text NOT NULL,
    coin_id text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, source, symbol)
);

CREATE INDEX idx_tso_coin_id ON tenant_symbols (coin_id);
