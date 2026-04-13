package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	pricev1 "github.com/NightRunner/CryptoTax-Go/gen/price/v1"
)

type AggregatedTxPage struct {
	Transactions []AggregatedTransaction
	Total        int64
}

type AggregationUseCase interface {
	ProcessImport(ctx context.Context, event ImportEvent) error
	ListTransactionsByImport(ctx context.Context, userID, importID uuid.UUID, limit, offset int32) (AggregatedTxPage, error)
	ListTransactionsByRange(ctx context.Context, userID uuid.UUID, fromUTC, toUTC time.Time, limit, offset int32, targetFiat string) (AggregatedTxPage, error)
}

type UserSettingsUseCase interface {
	Get(ctx context.Context, userID uuid.UUID) (UserSettings, error)
	Upsert(ctx context.Context, settings UserSettings) (UserSettings, error)
	ListSupportedFiatCurrencies(ctx context.Context) ([]SupportedFiatCurrency, error)
}

type AggregatedTransactionRepo interface {
	UpsertBatch(ctx context.Context, txs []AggregatedTransaction) error
	ListByImport(ctx context.Context, userID, importID uuid.UUID, limit, offset int32) (AggregatedTxPage, error)
	ListByRange(ctx context.Context, userID uuid.UUID, fromUTC, toUTC time.Time, limit, offset int32) (AggregatedTxPage, error)
}

type ImportStateRepo interface {
	Get(ctx context.Context, userID, importID uuid.UUID) (AggregationImportState, error)
	UpsertProcessing(ctx context.Context, state AggregationImportState) error
	MarkCompleted(ctx context.Context, userID, importID uuid.UUID) error
	MarkFailed(ctx context.Context, userID, importID uuid.UUID, errMsg string) error
}

type UserSettingsRepo interface {
	Get(ctx context.Context, userID uuid.UUID) (UserSettings, error)
	Upsert(ctx context.Context, settings UserSettings) (UserSettings, error)
}

type LedgerClient interface {
	ListTransactionsByImport(ctx context.Context, userID, importID uuid.UUID) ([]LedgerTransaction, error)
}

type PriceClient interface {
	ValuateTransactionsBatch(ctx context.Context, req *pricev1.ValuateTransactionsRequest) (*pricev1.ValuateTransactionsResponse, error)
}

// LockManager coordinates import processing across multiple workers/instances.
// AcquireImportLock returns:
//   - locked=true: current worker acquired the lock and must continue processing.
//   - locked=false: lock is already held by another worker, caller should skip processing.
type LockManager interface {
	AcquireImportLock(ctx context.Context, userID, importID uuid.UUID, ttl time.Duration) (bool, error)
	// ReleaseImportLock releases previously acquired lock for (userID, importID).
	ReleaseImportLock(ctx context.Context, userID, importID uuid.UUID) error
}
