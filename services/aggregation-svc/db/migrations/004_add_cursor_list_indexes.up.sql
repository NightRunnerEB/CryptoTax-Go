CREATE INDEX IF NOT EXISTS idx_aggregated_transactions_user_time_id
ON aggregated_transactions (user_id, time_utc DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_aggregated_transactions_user_source_time_id
ON aggregated_transactions (user_id, source, time_utc DESC, id DESC);

DROP INDEX IF EXISTS idx_aggregated_transactions_import;
CREATE INDEX IF NOT EXISTS idx_aggregated_transactions_user_import_time_id
ON aggregated_transactions (user_id, import_id, time_utc DESC, id DESC);
