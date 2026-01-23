package domain

import (
	"context"

	"github.com/google/uuid"
)

type TenantSymbol struct {
	TenantID uuid.UUID `json:"tenant_id"`
	Source   string    `json:"source"`
	Symbol   string    `json:"symbol"`
	CoinID   string    `json:"coin_id"`
}

type TenantSymbolUseCase interface {
	Upsert(ctx context.Context, s TenantSymbol) error
	Delete(ctx context.Context, tenantID uuid.UUID, source, symbol string) error

	GetList(ctx context.Context, tenantID uuid.UUID, source string, symbols []string) ([]TenantSymbol, error)
	GetListBySource(ctx context.Context, tenantID uuid.UUID, source string) ([]TenantSymbol, error)
}

type TenantSymbolRepo interface {
	Upsert(ctx context.Context, s TenantSymbol) error
	Delete(ctx context.Context, tenantID uuid.UUID, source, symbol string) error

	GetList(ctx context.Context, tenantID uuid.UUID, source string, symbols []string) ([]TenantSymbol, error)
	GetListBySource(ctx context.Context, tenantID uuid.UUID, source string) ([]TenantSymbol, error)
}
