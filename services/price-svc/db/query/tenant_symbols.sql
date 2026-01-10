-- name: UpsertTenantSymbol :exec
INSERT INTO tenant_symbols (tenant_id, source, symbol, coin_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tenant_id, source, symbol)
DO UPDATE SET coin_id = EXCLUDED.coin_id, updated_at = now();

-- name: GetTenantSymbols :many
SELECT tenant_id, source, symbol, coin_id, created_at, updated_at
FROM tenant_symbols
WHERE tenant_id = $1
  AND source = $2
  AND symbol = ANY($3::text[]);

-- name: ListTenantSymbolsBySource :many
SELECT tenant_id, source, symbol, coin_id, created_at, updated_at
FROM tenant_symbols
WHERE tenant_id = $1
  AND source = $2
ORDER BY symbol ASC;

-- name: DeleteTenantSymbol :execrows
DELETE FROM tenant_symbols
WHERE tenant_id = $1 AND source = $2 AND symbol = $3;
