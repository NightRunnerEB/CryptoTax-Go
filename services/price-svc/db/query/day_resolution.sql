INSERT INTO day_resolution (coin_id, day_utc, granularity_seconds)
VALUES ($1, $2, $3)
ON CONFLICT (coin_id, day_utc) DO NOTHING;
