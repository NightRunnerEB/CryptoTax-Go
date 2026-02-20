-- name: GetTaxpayerProfile :one
SELECT
  tenant_id, inn, last_name, first_name, middle_name, birth_date,
  document_type_code, document_number, tax_residency_status, phone,
  created_at, updated_at
FROM taxpayer_profile
WHERE tenant_id = $1;

-- name: UpsertTaxpayerProfile :one
INSERT INTO taxpayer_profile (
  tenant_id, inn, last_name, first_name, middle_name, birth_date,
  document_type_code, document_number, tax_residency_status, phone
)
VALUES (
  $1, $2, $3, $4, $5, $6,
  $7, $8, $9, $10
)
ON CONFLICT (tenant_id)
DO UPDATE SET
  inn = EXCLUDED.inn,
  last_name = EXCLUDED.last_name,
  first_name = EXCLUDED.first_name,
  middle_name = EXCLUDED.middle_name,
  birth_date = EXCLUDED.birth_date,
  document_type_code = EXCLUDED.document_type_code,
  document_number = EXCLUDED.document_number,
  tax_residency_status = EXCLUDED.tax_residency_status,
  phone = EXCLUDED.phone,
  updated_at = now()
RETURNING
  tenant_id, inn, last_name, first_name, middle_name, birth_date,
  document_type_code, document_number, tax_residency_status, phone,
  created_at, updated_at;
