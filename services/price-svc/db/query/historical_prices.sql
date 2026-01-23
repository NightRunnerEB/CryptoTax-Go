-- name: UpsertHistoricalPrice :exec
INSERT INTO historical_prices (coin_id, bucket_start_utc, price_usd, granularity_seconds, fetched_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (coin_id, bucket_start_utc)
DO UPDATE SET
  price_usd = EXCLUDED.price_usd,
  granularity_seconds = EXCLUDED.granularity_seconds,
  fetched_at = now()
WHERE EXCLUDED.granularity_seconds < historical_prices.granularity_seconds;

-- name: UpsertHistoricalPricesBatch :exec
WITH rows AS (
  SELECT
    c.coin_id,
    b.bucket_start_utc,
    p.price_usd,
    g.granularity_seconds
  FROM unnest($1::text[])        WITH ORDINALITY AS c(coin_id, ord)
  JOIN unnest($2::timestamptz[]) WITH ORDINALITY AS b(bucket_start_utc, ord) USING (ord)
  JOIN unnest($3::numeric[])     WITH ORDINALITY AS p(price_usd, ord) USING (ord)
  JOIN unnest($4::int4[])        WITH ORDINALITY AS g(granularity_seconds, ord) USING (ord)
)
INSERT INTO historical_prices (
  coin_id,
  bucket_start_utc,
  price_usd,
  granularity_seconds,
  fetched_at
)
SELECT
  coin_id,
  bucket_start_utc,
  price_usd,
  granularity_seconds,
  now()
FROM rows
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
  SELECT c.coin_id, b.bucket_start_utc, c.ord
  FROM unnest($1::text[]) WITH ORDINALITY AS c(coin_id, ord)
  JOIN unnest($2::timestamptz[]) WITH ORDINALITY AS b(bucket_start_utc, ord)
    USING (ord)
)
SELECT
  k.coin_id::text                        AS coin_id,
  k.bucket_start_utc::timestamptz        AS bucket_start_utc,
  hp.price_usd                           AS price_usd,
  hp.granularity_seconds                 AS granularity_seconds,
  hp.fetched_at                          AS fetched_at
FROM keys k
LEFT JOIN historical_prices hp
  ON hp.coin_id = k.coin_id
 AND hp.bucket_start_utc = k.bucket_start_utc
ORDER BY k.ord;