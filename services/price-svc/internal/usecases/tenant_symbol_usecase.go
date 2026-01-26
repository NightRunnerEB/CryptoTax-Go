package usecase

import (
	"context"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	"github.com/google/uuid"
)

type tenantSymbolUC struct {
	tenantSymbolRepository domain.TenantSymbolRepo
	contextTimeout         time.Duration
}

func NewTenantSymbolUC(tenantSymbolRepository domain.TenantSymbolRepo, timeout time.Duration) domain.TenantSymbolUseCase {
	return &tenantSymbolUC{
		tenantSymbolRepository: tenantSymbolRepository,
		contextTimeout:         timeout,
	}
}

func (u *tenantSymbolUC) Upsert(ctx context.Context, s domain.TenantSymbol) error {
	// Implementation goes here
	return nil
}

func (u *tenantSymbolUC) Delete(ctx context.Context, tenantID uuid.UUID, source, symbol string) error {
	// Implementation goes here
	return nil
}

func (u *tenantSymbolUC) GetList(ctx context.Context, tenantID uuid.UUID, source string, symbols []string) ([]domain.TenantSymbol, error) {
	// Implementation goes here
	return nil, nil
}

func (u *tenantSymbolUC) GetListBySource(ctx context.Context, tenantID uuid.UUID, source string) ([]domain.TenantSymbol, error) {
	// Implementation goes here
	return nil, nil
}
