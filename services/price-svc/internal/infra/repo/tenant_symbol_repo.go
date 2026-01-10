package repository

import (
	"context"
	"fmt"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/db"
	sqlc "github.com/NightRunner/CryptoTax-Go/services/price-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	"github.com/google/uuid"
)

type tenantSymbolRepository struct {
	store db.Store
}

func NewTenantSymbolRepo(store db.Store) domain.TenantSymbolRepo {
	return &tenantSymbolRepository{store: store}
}

func (r *tenantSymbolRepository) Upsert(ctx context.Context, s domain.TenantSymbol) error {
	if s.TenantID == uuid.Nil {
		return fmt.Errorf("Upsert: tenantID is nil")
	}
	if s.Source == "" {
		return fmt.Errorf("Upsert: source is empty")
	}
	if s.Symbol == "" {
		return fmt.Errorf("Upsert: symbol is empty")
	}
	if s.CoinID == "" {
		return fmt.Errorf("Upsert: coinID is empty")
	}

	if err := r.store.UpsertTenantSymbol(ctx, sqlc.UpsertTenantSymbolParams{
		TenantID: s.TenantID,
		Source:   s.Source,
		Symbol:   s.Symbol,
		CoinID:   s.CoinID,
	}); err != nil {
		return fmt.Errorf("Upsert: query failed: %w", err)
	}

	return nil
}

func (r *tenantSymbolRepository) Delete(ctx context.Context, tenantID uuid.UUID, source, symbol string) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("Delete: tenantID is nil")
	}
	if source == "" {
		return fmt.Errorf("Delete: source is empty")
	}
	if symbol == "" {
		return fmt.Errorf("Delete: symbol is empty")
	}

	rowsAffected, err := r.store.DeleteTenantSymbol(ctx, sqlc.DeleteTenantSymbolParams{
		TenantID: tenantID,
		Source:   source,
		Symbol:   symbol,
	})
	if err != nil {
		return fmt.Errorf("Delete: query failed: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("Delete: tenant symbol not found")
	}

	return nil
}

func (r *tenantSymbolRepository) GetList(
	ctx context.Context,
	tenantID uuid.UUID,
	source string,
	symbols []string,
) ([]domain.TenantSymbol, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("GetList: tenantID is nil")
	}
	if source == "" {
		return nil, fmt.Errorf("GetList: source is empty")
	}
	if len(symbols) == 0 {
		return []domain.TenantSymbol{}, nil
	}

	rows, err := r.store.GetTenantSymbols(ctx, sqlc.GetTenantSymbolsParams{
		TenantID: tenantID,
		Source:   source,
		Column3:  symbols,
	})
	if err != nil {
		return nil, fmt.Errorf("GetList: query failed: %w", err)
	}

	out := make([]domain.TenantSymbol, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapTenantSymbolDBToDomain(row))
	}
	return out, nil
}

func (r *tenantSymbolRepository) GetListBySource(
	ctx context.Context,
	tenantID uuid.UUID,
	source string,
) ([]domain.TenantSymbol, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("GetListBySource: tenantID is nil")
	}
	if source == "" {
		return nil, fmt.Errorf("GetListBySource: source is empty")
	}

	rows, err := r.store.ListTenantSymbolsBySource(ctx, sqlc.ListTenantSymbolsBySourceParams{
		TenantID: tenantID,
		Source:   source,
	})
	if err != nil {
		return nil, fmt.Errorf("GetListBySource: query failed: %w", err)
	}

	out := make([]domain.TenantSymbol, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapTenantSymbolDBToDomain(row))
	}
	return out, nil
}
