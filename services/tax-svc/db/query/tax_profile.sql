-- name: GetTaxProfile :one
SELECT
  tenant_id,
  jurisdiction,
  cost_basis_method,
  timezone,
  treat_swap_as_disposition,
  treat_crypto_fee_as_disposition,
  include_income_events,
  allow_loss_events_deduction,
  fail_on_negative_inventory,
  fail_on_missing_fiat,
  created_at,
  updated_at
FROM tax_profile
WHERE tenant_id = $1;

-- name: UpsertTaxProfile :one
INSERT INTO tax_profile (
  tenant_id, jurisdiction, cost_basis_method, timezone,
  treat_swap_as_disposition, treat_crypto_fee_as_disposition, include_income_events,
  allow_loss_events_deduction, fail_on_negative_inventory, fail_on_missing_fiat
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (tenant_id)
DO UPDATE SET
  jurisdiction = EXCLUDED.jurisdiction,
  cost_basis_method = EXCLUDED.cost_basis_method,
  treat_swap_as_disposition = EXCLUDED.treat_swap_as_disposition,
  treat_crypto_fee_as_disposition = EXCLUDED.treat_crypto_fee_as_disposition,
  include_income_events = EXCLUDED.include_income_events,
  allow_loss_events_deduction = EXCLUDED.allow_loss_events_deduction,
  fail_on_negative_inventory = EXCLUDED.fail_on_negative_inventory,
  fail_on_missing_fiat = EXCLUDED.fail_on_missing_fiat,
  timezone = EXCLUDED.timezone,
  updated_at = now()
RETURNING
  tenant_id,
  jurisdiction,
  cost_basis_method,
  timezone,
  treat_swap_as_disposition,
  treat_crypto_fee_as_disposition,
  include_income_events,
  allow_loss_events_deduction,
  fail_on_negative_inventory,
  fail_on_missing_fiat,
  created_at,
  updated_at;
