CREATE TABLE
  IF NOT EXISTS tax_profile (
    user_id uuid PRIMARY KEY,
    inn text NOT NULL,
    oktmo text NOT NULL,
    last_name text NOT NULL,
    first_name text NOT NULL,
    middle_name text NOT NULL,
    timezone text NOT NULL,
    phone text NOT NULL,
    wallets jsonb NOT NULL DEFAULT '[]'::jsonb,
    tax_residency_status text NOT NULL,
    taxpayer_type text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now (),
    updated_at timestamptz NOT NULL DEFAULT now ()
  );
