ALTER TABLE tax_profile
ADD COLUMN IF NOT EXISTS treat_swap_as_disposition boolean NOT NULL DEFAULT false,
ADD COLUMN IF NOT EXISTS treat_crypto_fee_as_disposition boolean NOT NULL DEFAULT true,
ADD COLUMN IF NOT EXISTS include_income_events boolean NOT NULL DEFAULT true,
ADD COLUMN IF NOT EXISTS allow_loss_events_deduction boolean NOT NULL DEFAULT false,
ADD COLUMN IF NOT EXISTS fail_on_negative_inventory boolean NOT NULL DEFAULT true,
ADD COLUMN IF NOT EXISTS fail_on_missing_fiat boolean NOT NULL DEFAULT true;

