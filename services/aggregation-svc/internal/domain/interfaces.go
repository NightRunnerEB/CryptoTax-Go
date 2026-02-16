package domain

import (
	"context"
	"time"

	pricev1 "github.com/NightRunner/CryptoTax-Go/gen/price/v1"
	"github.com/google/uuid"
)

type AggregatedTxPage struct {
	Transactions []AggregatedTransaction
	Total        int64
}

type AggregationUseCase interface {
	ProcessImport(ctx context.Context, event ImportEvent) error
	ListTransactionsByImport(ctx context.Context, tenantID, importID uuid.UUID, limit, offset int32) (AggregatedTxPage, error)
	ListTransactionsByRange(ctx context.Context, tenantID uuid.UUID, fromUTC, toUTC time.Time, limit, offset int32) (AggregatedTxPage, error)
}

type TenantSettingsUseCase interface {
	Get(ctx context.Context, tenantID uuid.UUID) (TenantSettings, error)
	Upsert(ctx context.Context, settings TenantSettings) (TenantSettings, error)
}

type AggregatedTransactionRepo interface {
	UpsertBatch(ctx context.Context, txs []AggregatedTransaction) error
	ListByImport(ctx context.Context, tenantID, importID uuid.UUID, limit, offset int32) (AggregatedTxPage, error)
	ListByRange(ctx context.Context, tenantID uuid.UUID, fromUTC, toUTC time.Time, limit, offset int32) (AggregatedTxPage, error)
}

type ImportStateRepo interface {
	Get(ctx context.Context, tenantID, importID uuid.UUID) (AggregationImportState, error)
	UpsertProcessing(ctx context.Context, state AggregationImportState) error
	MarkCompleted(ctx context.Context, tenantID, importID uuid.UUID) error
	MarkFailed(ctx context.Context, tenantID, importID uuid.UUID, errMsg string) error
}

type TenantSettingsRepo interface {
	Get(ctx context.Context, tenantID uuid.UUID) (TenantSettings, error)
	Upsert(ctx context.Context, settings TenantSettings) (TenantSettings, error)
}

type LedgerClient interface {
	ListTransactionsByImport(ctx context.Context, tenantID, importID uuid.UUID) ([]LedgerTransaction, error)
}

type PriceClient interface {
	ValuateTransactionsBatch(ctx context.Context, req *pricev1.ValuateTransactionsRequest) (*pricev1.ValuateTransactionsResponse, error)
}

type LockManager interface {
	AcquireImportLock(ctx context.Context, tenantID, importID uuid.UUID, ttl time.Duration) (bool, error)
	ReleaseImportLock(ctx context.Context, tenantID, importID uuid.UUID) error
}
