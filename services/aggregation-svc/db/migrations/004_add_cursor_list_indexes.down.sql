DROP INDEX IF EXISTS idx_aggregated_transactions_user_time_id;
DROP INDEX IF EXISTS idx_aggregated_transactions_user_source_time_id;
DROP INDEX IF EXISTS idx_aggregated_transactions_user_import_time_id;

CREATE INDEX IF NOT EXISTS idx_aggregated_transactions_import
ON aggregated_transactions (user_id, import_id, time_utc DESC);
