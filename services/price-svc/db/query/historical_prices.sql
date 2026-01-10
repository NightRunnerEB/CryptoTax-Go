-- name: UpsertHistoricalPrice :exec
INSERT INTO historical_prices (coin_id, bucket_start_utc, price_usd, granularity_seconds, fetched_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (coin_id, bucket_start_utc)
DO UPDATE SET
  price_usd = EXCLUDED.price_usd,
  granularity_seconds = EXCLUDED.granularity_seconds,
  fetched_at = now()
WHERE EXCLUDED.granularity_seconds < historical_prices.granularity_seconds;

-- name: GetHistoricalPrice :one
SELECT coin_id, bucket_start_utc, price_usd, granularity_seconds, fetched_at
FROM historical_prices
WHERE coin_id = $1
  AND bucket_start_utc = $2;

-- name: GetHistoricalPricesBatch :many
WITH keys AS (
  SELECT c.coin_id, b.bucket_start_utc
  FROM unnest($1::text[]) WITH ORDINALITY AS c(coin_id, ord)
  JOIN unnest($2::timestamptz[]) WITH ORDINALITY AS b(bucket_start_utc, ord)
    USING (ord)
)
SELECT
  hp.coin_id,
  hp.bucket_start_utc,
  hp.price_usd,
  hp.granularity_seconds,
  hp.fetched_at
FROM historical_prices hp
JOIN keys k
  ON hp.coin_id = k.coin_id
 AND hp.bucket_start_utc = k.bucket_start_utc;