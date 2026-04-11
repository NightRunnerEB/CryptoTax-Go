-- name: UpsertFXRate :exec
INSERT INTO fx_rates (fiat, day, rate, is_real, source)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (fiat, day)
DO UPDATE SET
  rate = EXCLUDED.rate,
  is_real = EXCLUDED.is_real,
  source = EXCLUDED.source,
  updated_at = now();

-- name: GetFXRate :one
SELECT fiat, day, rate, is_real, source, updated_at
FROM fx_rates
WHERE fiat = $1
  AND day <= $2
ORDER BY day DESC
LIMIT 1;

-- name: ListFXRatesByFiat :many
SELECT fiat, day, rate, is_real, source, updated_at
FROM fx_rates
WHERE fiat = $1
ORDER BY day ASC;
