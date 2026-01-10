CREATE TABLE historical_prices (
    coin_id text NOT NULL,
    bucket_start_utc timestamptz NOT NULL,
    price_usd numeric NOT NULL,
    granularity_seconds integer NOT NULL DEFAULT 86400,
    fetched_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (coin_id, bucket_start_utc)
);

CREATE INDEX idx_prices_bucket ON historical_prices (bucket_start_utc DESC);