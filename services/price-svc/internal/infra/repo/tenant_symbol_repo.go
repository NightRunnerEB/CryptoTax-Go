package repository

import (
	"context"

	db "github.com/NightRunner/CryptoTax-Go/services/price-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
	"github.com/google/uuid"
)

type tenantSymbolRepository struct {
	store db.Store
}

func NewTenantSymbolRepo(store db.Store) domain.TenantSymbolRepo {
	return &tenantSymbolRepository{store: store}
}

func (r *tenantSymbolRepository) Upsert(ctx context.Context, s domain.TenantSymbol) error {
	var violations []apperr.FieldViolation
	if s.TenantID == uuid.Nil {
		violations = append(violations, apperr.FieldViolation{
			Field:       "tenant_id",
			Description: "required",
		})
	}
	if s.Source == "" {
		violations = append(violations, apperr.FieldViolation{
			Field:       "source",
			Description: "required",
		})
	}
	if s.Symbol == "" {
		violations = append(violations, apperr.FieldViolation{
			Field:       "symbol",
			Description: "required",
		})
	}
	if s.CoinID == "" {
		violations = append(violations, apperr.FieldViolation{
			Field:       "coin_id",
			Description: "required",
		})
	}
	if len(violations) > 0 {
		return apperr.InvalidArgument("invalid tenant symbol", nil, violations...)
	}

	if err := r.store.UpsertTenantSymbol(ctx, db.UpsertTenantSymbolParams{
		TenantID: s.TenantID,
		Source:   s.Source,
		Symbol:   s.Symbol,
		CoinID:   s.CoinID,
	}); err != nil {
		return apperr.Internal("upsert tenant symbol failed", err, map[string]string{
			"tenant_id": s.TenantID.String(),
			"source":    s.Source,
			"symbol":    s.Symbol,
			"coin_id":   s.CoinID,
		})
	}

	return nil
}

func (r *tenantSymbolRepository) Delete(ctx context.Context, tenantID uuid.UUID, source, symbol string) error {
	var violations []apperr.FieldViolation
	if tenantID == uuid.Nil {
		violations = append(violations, apperr.FieldViolation{
			Field:       "tenant_id",
			Description: "required",
		})
	}
	if source == "" {
		violations = append(violations, apperr.FieldViolation{
			Field:       "source",
			Description: "required",
		})
	}
	if symbol == "" {
		violations = append(violations, apperr.FieldViolation{
			Field:       "symbol",
			Description: "required",
		})
	}
	if len(violations) > 0 {
		return apperr.InvalidArgument("invalid tenant symbol", nil, violations...)
	}

	rowsAffected, err := r.store.DeleteTenantSymbol(ctx, db.DeleteTenantSymbolParams{
		TenantID: tenantID,
		Source:   source,
		Symbol:   symbol,
	})
	if err != nil {
		return apperr.Internal("delete tenant symbol failed", err, map[string]string{
			"tenant_id": tenantID.String(),
			"source":    source,
			"symbol":    symbol,
		})
	}

	if rowsAffected == 0 {
		name := tenantID.String() + ":" + source + ":" + symbol
		return apperr.NotFound("tenant symbol not found", apperr.Resource{
			Type: "tenant_symbol",
			Name: name,
		}, nil)
	}

	return nil
}

func (r *tenantSymbolRepository) GetList(
	ctx context.Context,
	tenantID uuid.UUID,
	source string,
	symbols []string,
) ([]domain.TenantSymbol, error) {
	var violations []apperr.FieldViolation
	if tenantID == uuid.Nil {
		violations = append(violations, apperr.FieldViolation{
			Field:       "tenant_id",
			Description: "required",
		})
	}
	if source == "" {
		violations = append(violations, apperr.FieldViolation{
			Field:       "source",
			Description: "required",
		})
	}
	if len(violations) > 0 {
		return nil, apperr.InvalidArgument("invalid tenant symbol query", nil, violations...)
	}
	if len(symbols) == 0 {
		return []domain.TenantSymbol{}, nil
	}

	rows, err := r.store.GetTenantSymbols(ctx, db.GetTenantSymbolsParams{
		TenantID: tenantID,
		Source:   source,
		Column3:  symbols,
	})
	if err != nil {
		return nil, apperr.Internal("get tenant symbols failed", err, map[string]string{
			"tenant_id": tenantID.String(),
			"source":    source,
		})
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
	var violations []apperr.FieldViolation
	if tenantID == uuid.Nil {
		violations = append(violations, apperr.FieldViolation{
			Field:       "tenant_id",
			Description: "required",
		})
	}
	if source == "" {
		violations = append(violations, apperr.FieldViolation{
			Field:       "source",
			Description: "required",
		})
	}
	if len(violations) > 0 {
		return nil, apperr.InvalidArgument("invalid tenant symbol query", nil, violations...)
	}

	rows, err := r.store.ListTenantSymbolsBySource(ctx, db.ListTenantSymbolsBySourceParams{
		TenantID: tenantID,
		Source:   source,
	})
	if err != nil {
		return nil, apperr.Internal("get tenant symbols by source failed", err, map[string]string{
			"tenant_id": tenantID.String(),
			"source":    source,
		})
	}

	out := make([]domain.TenantSymbol, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapTenantSymbolDBToDomain(row))
	}
	return out, nil
}
