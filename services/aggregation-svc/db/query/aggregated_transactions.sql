-- name: UpsertAggregatedTransaction :exec
INSERT INTO aggregated_transactions (
  id,
  user_id,
  source,
  import_id,
  time_utc,
  kind,
  in_money,
  out_money,
  fee_money,
  contract_symbol,
  derivative_kind,
  position_id,
  order_id,
  tx_hash,
  note,
  tx_fingerprint,
  created_at
)
VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9,
  $10, $11, $12, $13, $14, $15,
  $16, $17
)
ON CONFLICT (id)
DO UPDATE SET
  source = EXCLUDED.source,
  import_id = EXCLUDED.import_id,
  time_utc = EXCLUDED.time_utc,
  kind = EXCLUDED.kind,
  in_money = EXCLUDED.in_money,
  out_money = EXCLUDED.out_money,
  fee_money = EXCLUDED.fee_money,
  contract_symbol = EXCLUDED.contract_symbol,
  derivative_kind = EXCLUDED.derivative_kind,
  position_id = EXCLUDED.position_id,
  order_id = EXCLUDED.order_id,
  tx_hash = EXCLUDED.tx_hash,
  note = EXCLUDED.note,
  tx_fingerprint = EXCLUDED.tx_fingerprint,
  updated_at = now();

-- name: UpdateAggregatedTransactionByFingerprint :execrows
UPDATE aggregated_transactions
SET
  id = $1,
  source = $3,
  import_id = $4,
  time_utc = $5,
  kind = $6,
  in_money = $7,
  out_money = $8,
  fee_money = $9,
  contract_symbol = $10,
  derivative_kind = $11,
  position_id = $12,
  order_id = $13,
  tx_hash = $14,
  note = $15,
  created_at = $17,
  updated_at = now()
WHERE user_id = $2 AND tx_fingerprint = $16;

-- name: ListAggregatedTransactionsByRange :many
SELECT
  id,
  user_id,
  source,
  import_id,
  time_utc,
  kind,
  in_money,
  out_money,
  fee_money,
  contract_symbol,
  derivative_kind,
  position_id,
  order_id,
  tx_hash,
  note,
  tx_fingerprint,
  created_at,
  updated_at
FROM aggregated_transactions
WHERE user_id = $1
  AND time_utc >= $2
  AND time_utc < $3
ORDER BY time_utc ASC
LIMIT $4 OFFSET $5;

-- name: ListAggregatedTransactions :many
SELECT
  id,
  user_id,
  source,
  import_id,
  time_utc,
  kind,
  in_money,
  out_money,
  fee_money,
  contract_symbol,
  derivative_kind,
  position_id,
  order_id,
  tx_hash,
  note,
  tx_fingerprint,
  created_at,
  updated_at
FROM aggregated_transactions
WHERE user_id = sqlc.arg(user_id)
  AND (sqlc.narg(date_from)::timestamptz IS NULL OR time_utc >= sqlc.narg(date_from)::timestamptz)
  AND (sqlc.narg(date_to)::timestamptz IS NULL OR time_utc < sqlc.narg(date_to)::timestamptz)
  AND (sqlc.narg(import_id)::uuid IS NULL OR import_id = sqlc.narg(import_id)::uuid)
  AND (sqlc.narg(source)::text IS NULL OR source = sqlc.narg(source)::text)
  AND (sqlc.narg(kind)::text IS NULL OR kind = sqlc.narg(kind)::text)
  AND (
    NOT sqlc.arg(has_cursor)::bool
    OR (
      time_utc < sqlc.arg(cursor_time)::timestamptz
      OR (time_utc = sqlc.arg(cursor_time)::timestamptz AND id < sqlc.arg(cursor_id)::uuid)
    )
  )
ORDER BY time_utc DESC, id DESC
LIMIT sqlc.arg(page_limit);

-- name: CountAggregatedTransactionsByRange :one
SELECT count(*)
FROM aggregated_transactions
WHERE user_id = $1
  AND time_utc >= $2
  AND time_utc < $3;
