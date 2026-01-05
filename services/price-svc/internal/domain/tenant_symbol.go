package domain

import (
	"context"

	"github.com/google/uuid"
)

type TenantSymbol struct {
	TenantID  uuid.UUID `json:"tenant_id"`
	Source    string    `json:"source"`
	Symbol    string    `json:"symbol"`
	CoinID    string    `json:"coin_id"`
}

type TenantSymbolUseCase interface {
	Upsert(s TenantSymbol) error
	Delete(tenantID uuid.UUID, source, symbol string) error

	GetTenantSymbols(tenantID uuid.UUID, source string, symbols []string) ([]TenantSymbol, error)
	ListTenantSymbolsBySource(tenantID uuid.UUID, source string) ([]TenantSymbol, error)
}

type TenantSymbolRepo interface {
	Upsert(ctx context.Context, s TenantSymbol) error
	Delete(ctx context.Context, tenantID uuid.UUID, source, symbol string) error

	GetBySymbols(ctx context.Context, tenantID uuid.UUID, source string, symbols []string) ([]TenantSymbol, error)
	ListBySource(ctx context.Context, tenantID uuid.UUID, source string) ([]TenantSymbol, error)
}

