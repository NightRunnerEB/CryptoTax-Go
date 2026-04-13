-- name: GetTaxProfile :one
SELECT
  user_id,
  inn,
  last_name,
  first_name,
  middle_name,
  timezone,
  phone,
  wallets,
  tax_residency_status,
  taxpayer_type,
  created_at,
  updated_at
FROM tax_profile
WHERE user_id = $1;

-- name: UpsertTaxProfile :one
INSERT INTO tax_profile (
  user_id,
  inn,
  last_name,
  first_name,
  middle_name,
  timezone,
  phone,
  wallets,
  tax_residency_status,
  taxpayer_type
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (user_id)
DO UPDATE SET
  inn = EXCLUDED.inn,
  last_name = EXCLUDED.last_name,
  first_name = EXCLUDED.first_name,
  middle_name = EXCLUDED.middle_name,
  timezone = EXCLUDED.timezone,
  phone = EXCLUDED.phone,
  wallets = EXCLUDED.wallets,
  tax_residency_status = EXCLUDED.tax_residency_status,
  taxpayer_type = EXCLUDED.taxpayer_type,
  updated_at = now()
RETURNING
  user_id,
  inn,
  last_name,
  first_name,
  middle_name,
  timezone,
  phone,
  wallets,
  tax_residency_status,
  taxpayer_type,
  created_at,
  updated_at;

-- name: DeleteTaxProfile :execrows
DELETE FROM tax_profile
WHERE user_id = $1;
