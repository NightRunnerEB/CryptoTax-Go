package repository

import (
	"context"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/pkg/postgres"
	"github.com/google/uuid"
)

type tenantSymbolRepository struct {
	*postgres.Postgres
}

func NewTenantSymbolRepo(pg *postgres.Postgres) domain.TenantSymbolRepo {
	return &tenantSymbolRepository{pg}
}

func (r *tenantSymbolRepository) Upsert(ctx context.Context, s domain.TenantSymbol) error {
	// Implementation goes here
	return nil
}

func (r *tenantSymbolRepository) Delete(ctx context.Context, tenantID uuid.UUID, source, symbol string) error {
	// Implementation goes here
	return nil
}

func (r *tenantSymbolRepository) GetBySymbols(ctx context.Context, tenantID uuid.UUID, source string, symbols []string) ([]domain.TenantSymbol, error) {
	// Implementation goes here
	return nil, nil
}

func (r *tenantSymbolRepository) ListBySource(ctx context.Context, tenantID uuid.UUID, source string) ([]domain.TenantSymbol, error) {
	// Implementation goes here
	return nil, nil
}
