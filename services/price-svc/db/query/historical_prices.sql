-- name: UpsertHistoricalPrice :exec
INSERT INTO historical_prices (coin_id, bucket_start_utc, price_usd)
VALUES ($1, $2, $3)
ON CONFLICT (coin_id, bucket_start_utc)
DO UPDATE SET price_usd = EXCLUDED.price_usd, fetched_at = now();

-- name: GetHistoricalPrice :one
SELECT coin_id, bucket_start_utc, price_usd, fetched_at
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
  hp.fetched_at
FROM historical_prices hp
JOIN keys k
  ON hp.coin_id = k.coin_id
 AND hp.bucket_start_utc = k.bucket_start_utc
WHERE hp.bucket_start_utc = $2;