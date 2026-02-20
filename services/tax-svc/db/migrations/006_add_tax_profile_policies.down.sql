ALTER TABLE tax_profile
DROP COLUMN IF EXISTS fail_on_missing_fiat,
DROP COLUMN IF EXISTS fail_on_negative_inventory,
DROP COLUMN IF EXISTS allow_loss_events_deduction,
DROP COLUMN IF EXISTS include_income_events,
DROP COLUMN IF EXISTS treat_crypto_fee_as_disposition,
DROP COLUMN IF EXISTS treat_swap_as_disposition;

