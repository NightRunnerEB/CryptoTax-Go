-- name: UpsertHistoricalPrice :exec
INSERT INTO historical_prices (coin_id, fiat_currency, bucket_start_utc, source_profile, rate)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (coin_id, fiat_currency, bucket_start_utc, source_profile)
DO UPDATE SET rate = EXCLUDED.rate, fetched_at = now();

-- name: GetHistoricalPrice :one
SELECT coin_id, fiat_currency, bucket_start_utc, source_profile, rate, fetched_at
FROM historical_prices
WHERE coin_id = $1
  AND fiat_currency = $2
  AND bucket_start_utc = $3
  AND source_profile = $4;

-- name: GetHistoricalPricesBatch :many
WITH keys AS (
  SELECT c.coin_id, b.bucket_start_utc
  FROM unnest($1::text[]) WITH ORDINALITY AS c(coin_id, ord)
  JOIN unnest($2::timestamptz[]) WITH ORDINALITY AS b(bucket_start_utc, ord)
    USING (ord)
)
SELECT
  hp.coin_id,
  hp.fiat_currency,
  hp.bucket_start_utc,
  hp.source_profile,
  hp.rate,
  hp.fetched_at
FROM historical_prices hp
JOIN keys k
  ON hp.coin_id = k.coin_id
 AND hp.bucket_start_utc = k.bucket_start_utc
WHERE hp.fiat_currency = $3
  AND hp.source_profile = $4;