-- name: UpsertTenantSymbolOverride :exec
INSERT INTO tenant_symbol_overrides (tenant_id, source, symbol, coin_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (tenant_id, source, symbol)
DO UPDATE SET coin_id = EXCLUDED.coin_id, updated_at = now();

-- name: GetTenantSymbolOverridesBySymbols :many
SELECT tenant_id, source, symbol, coin_id, created_at, updated_at
FROM tenant_symbol_overrides
WHERE tenant_id = $1
  AND source = $2
  AND symbol = ANY($3::text[]);

-- name: ListTenantSymbolOverridesBySource :many
SELECT tenant_id, source, symbol, coin_id, created_at, updated_at
FROM tenant_symbol_overrides
WHERE tenant_id = $1
  AND source = $2
ORDER BY symbol ASC;

-- name: DeleteTenantSymbolOverride :exec
DELETE FROM tenant_symbol_overrides
WHERE tenant_id = $1 AND source = $2 AND symbol = $3;
