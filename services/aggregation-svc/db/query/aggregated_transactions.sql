-- name: UpsertAggregatedTransaction :exec
INSERT INTO aggregated_transactions (
  id,
  tenant_id,
  wallet,
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
  $1, $2, $3, $4, $5, $6, $7,
  $8, $9, $10,
  $11, $12, $13, $14, $15, $16,
  $17, $18
)
ON CONFLICT (id)
DO UPDATE SET
  wallet = EXCLUDED.wallet,
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

-- name: ListAggregatedTransactionsByImport :many
SELECT
  id,
  tenant_id,
  wallet,
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
WHERE tenant_id = $1 AND import_id = $2
ORDER BY time_utc DESC
LIMIT $3 OFFSET $4;

-- name: CountAggregatedTransactionsByImport :one
SELECT count(*)
FROM aggregated_transactions
WHERE tenant_id = $1 AND import_id = $2;
