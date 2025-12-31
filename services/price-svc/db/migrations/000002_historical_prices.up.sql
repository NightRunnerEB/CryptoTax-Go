CREATE TABLE historical_prices (
    coin_id text NOT NULL,
    fiat_currency text NOT NULL,
    bucket_start_utc timestamptz NOT NULL,
    source_profile text NOT NULL,
    rate numeric NOT NULL,
    fetched_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (coin_id, fiat_currency, bucket_start_utc, source_profile)
);

CREATE INDEX idx_prices_bucket ON historical_prices (bucket_start_utc DESC);

CREATE INDEX idx_prices_fetched ON historical_prices (fetched_at DESC);
