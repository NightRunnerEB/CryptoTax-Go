-- name: GetTenantSettings :one
SELECT tenant_id, fiat_currency, timezone, updated_at
FROM tenant_settings
WHERE tenant_id = $1;

-- name: UpsertTenantSettings :one
INSERT INTO tenant_settings (tenant_id, fiat_currency, timezone)
VALUES ($1, $2, $3)
ON CONFLICT (tenant_id)
DO UPDATE SET
  fiat_currency = EXCLUDED.fiat_currency,
  timezone = EXCLUDED.timezone,
  updated_at = now()
RETURNING tenant_id, fiat_currency, timezone, updated_at;
