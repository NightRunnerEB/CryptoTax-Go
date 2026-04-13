-- name: UpsertUserSymbol :exec
INSERT INTO user_symbols (user_id, source, symbol, coin_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, source, symbol)
DO UPDATE SET coin_id = EXCLUDED.coin_id, updated_at = now();

-- name: GetUserSymbols :many
SELECT user_id, source, symbol, coin_id, created_at, updated_at
FROM user_symbols
WHERE user_id = $1
  AND source = $2
  AND symbol = ANY($3::text[]);

-- name: ListUserSymbolsBySource :many
SELECT user_id, source, symbol, coin_id, created_at, updated_at
FROM user_symbols
WHERE user_id = $1
  AND source = $2
ORDER BY symbol ASC;

-- name: DeleteUserSymbol :execrows
DELETE FROM user_symbols
WHERE user_id = $1 AND source = $2 AND symbol = $3;
