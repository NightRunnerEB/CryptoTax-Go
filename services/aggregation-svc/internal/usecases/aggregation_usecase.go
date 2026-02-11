package usecase

import (
	"context"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
	"github.com/google/uuid"
)

type aggregationUC struct {
	txRepo         domain.AggregatedTransactionRepo
	importRepo     domain.ImportStateRepo
	settingsRepo   domain.TenantSettingsRepo
	ledgerClient   domain.LedgerClient
	priceClient    domain.PriceClient
	lockManager    domain.LockManager
	batchSize      int
	defaultFiat    string
	defaultTZ      string
	contextTimeout time.Duration
}

func NewAggregationUC(
	txRepo domain.AggregatedTransactionRepo,
	importRepo domain.ImportStateRepo,
	settingsRepo domain.TenantSettingsRepo,
	ledgerClient domain.LedgerClient,
	priceClient domain.PriceClient,
	lockManager domain.LockManager,
	batchSize int,
	defaultFiat string,
	defaultTimezone string,
	timeout time.Duration,
) domain.AggregationUseCase {
	return &aggregationUC{
		txRepo:         txRepo,
		importRepo:     importRepo,
		settingsRepo:   settingsRepo,
		ledgerClient:   ledgerClient,
		priceClient:    priceClient,
		lockManager:    lockManager,
		batchSize:      batchSize,
		defaultFiat:    defaultFiat,
		defaultTZ:      defaultTimezone,
		contextTimeout: timeout,
	}
}

func (u *aggregationUC) ProcessImportCompleted(ctx context.Context, event domain.ImportCompletedEvent) error {
	if u.contextTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, u.contextTimeout)
		defer cancel()
	}

	return apperr.Internal("method not implemented", nil, map[string]string{
		"tenant_id": event.TenantID.String(),
		"import_id": event.ImportID.String(),
		"source":    event.Source,
	})
}

func (u *aggregationUC) ListTransactionsByImport(ctx context.Context, tenantID, importID uuid.UUID, limit, offset int32) (domain.AggregatedTxPage, error) {
	if u.contextTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, u.contextTimeout)
		defer cancel()
	}

	return domain.AggregatedTxPage{}, apperr.Internal("method not implemented", nil, map[string]string{
		"tenant_id": tenantID.String(),
		"import_id": importID.String(),
	})
}
