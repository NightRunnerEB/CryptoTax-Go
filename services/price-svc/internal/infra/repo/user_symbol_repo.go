package repository

import (
	"context"

	"github.com/google/uuid"

	db "github.com/NightRunner/CryptoTax-Go/services/price-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

type userSymbolRepository struct {
	store db.Store
}

func NewUserSymbolRepo(store db.Store) domain.UserSymbolRepo {
	return &userSymbolRepository{store: store}
}

func (r *userSymbolRepository) Upsert(ctx context.Context, s domain.UserSymbol) error {
	var violations []apperr.FieldViolation
	if s.UserID == uuid.Nil {
		violations = append(violations, apperr.FieldViolation{
			Field:       "user_id",
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
		return apperr.InvalidArgument("invalid user symbol", nil, violations...)
	}

	if err := r.store.UpsertUserSymbol(ctx, db.UpsertUserSymbolParams{
		UserID: s.UserID,
		Source: s.Source,
		Symbol: s.Symbol,
		CoinID: s.CoinID,
	}); err != nil {
		return apperr.Internal("upsert user symbol failed", err, map[string]string{
			"user_id": s.UserID.String(),
			"source":  s.Source,
			"symbol":  s.Symbol,
			"coin_id": s.CoinID,
		})
	}

	return nil
}

func (r *userSymbolRepository) Delete(ctx context.Context, userID uuid.UUID, source, symbol string) error {
	var violations []apperr.FieldViolation
	if userID == uuid.Nil {
		violations = append(violations, apperr.FieldViolation{
			Field:       "user_id",
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
		return apperr.InvalidArgument("invalid user symbol", nil, violations...)
	}

	rowsAffected, err := r.store.DeleteUserSymbol(ctx, db.DeleteUserSymbolParams{
		UserID: userID,
		Source: source,
		Symbol: symbol,
	})
	if err != nil {
		return apperr.Internal("delete user symbol failed", err, map[string]string{
			"user_id": userID.String(),
			"source":  source,
			"symbol":  symbol,
		})
	}

	if rowsAffected == 0 {
		name := userID.String() + ":" + source + ":" + symbol
		return apperr.NotFound("user symbol not found", apperr.Resource{
			Type: "user_symbol",
			Name: name,
		}, nil)
	}

	return nil
}

func (r *userSymbolRepository) GetList(
	ctx context.Context,
	userID uuid.UUID,
	source string,
	symbols []string,
) ([]domain.UserSymbol, error) {
	var violations []apperr.FieldViolation
	if userID == uuid.Nil {
		violations = append(violations, apperr.FieldViolation{
			Field:       "user_id",
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
		return nil, apperr.InvalidArgument("invalid user symbol query", nil, violations...)
	}
	if len(symbols) == 0 {
		return nil, nil
	}

	rows, err := r.store.GetUserSymbols(ctx, db.GetUserSymbolsParams{
		UserID:  userID,
		Source:  source,
		Column3: symbols,
	})
	if err != nil {
		return nil, apperr.Internal("get user symbols failed", err, map[string]string{
			"user_id": userID.String(),
			"source":  source,
		})
	}

	out := make([]domain.UserSymbol, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapUserSymbolDBToDomain(row))
	}
	return out, nil
}

func (r *userSymbolRepository) GetListBySource(
	ctx context.Context,
	userID uuid.UUID,
	source string,
) ([]domain.UserSymbol, error) {
	var violations []apperr.FieldViolation
	if userID == uuid.Nil {
		violations = append(violations, apperr.FieldViolation{
			Field:       "user_id",
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
		return nil, apperr.InvalidArgument("invalid user symbol query", nil, violations...)
	}

	rows, err := r.store.ListUserSymbolsBySource(ctx, db.ListUserSymbolsBySourceParams{
		UserID: userID,
		Source: source,
	})
	if err != nil {
		return nil, apperr.Internal("get user symbols by source failed", err, map[string]string{
			"user_id": userID.String(),
			"source":  source,
		})
	}

	out := make([]domain.UserSymbol, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapUserSymbolDBToDomain(row))
	}
	return out, nil
}
