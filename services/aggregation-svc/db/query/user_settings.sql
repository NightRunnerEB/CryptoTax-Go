-- name: GetUserSettings :one
SELECT user_id, fiat_currency, timezone, updated_at
FROM user_settings
WHERE user_id = $1;

-- name: UpsertUserSettings :one
INSERT INTO user_settings (user_id, fiat_currency, timezone)
VALUES ($1, $2, $3)
ON CONFLICT (user_id)
DO UPDATE SET
  fiat_currency = EXCLUDED.fiat_currency,
  timezone = EXCLUDED.timezone,
  updated_at = now()
RETURNING user_id, fiat_currency, timezone, updated_at;
