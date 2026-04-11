package repository

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/NightRunner/CryptoTax-Go/services/price-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

type fxRateRepository struct {
	store db.Store
}

func NewFXRateRepo(store db.Store) domain.FXRateRepo {
	return &fxRateRepository{store: store}
}

func (r *fxRateRepository) Upsert(ctx context.Context, rate domain.FXRate) error {
	fiat := strings.ToUpper(strings.TrimSpace(rate.Fiat))
	if fiat == "" {
		return apperr.InvalidArgument("fiat is required", nil, apperr.FieldViolation{
			Field:       "fiat",
			Description: "required",
		})
	}
	if rate.Day.IsZero() {
		return apperr.InvalidArgument("day is required", nil, apperr.FieldViolation{
			Field:       "day",
			Description: "required",
		})
	}
	if !rate.Rate.IsPositive() {
		return apperr.InvalidArgument("rate must be > 0", nil, apperr.FieldViolation{
			Field:       "rate",
			Description: "must be > 0",
		})
	}

	rateNumeric, err := decimalToNumeric(&rate.Rate)
	if err != nil {
		return apperr.InvalidArgument("rate is invalid", err, apperr.FieldViolation{
			Field:       "rate",
			Description: "invalid",
		})
	}

	day := dateOnlyUTC(rate.Day)
	if err := r.store.UpsertFXRate(ctx, db.UpsertFXRateParams{
		Fiat:   fiat,
		Day:    pgtype.Date{Time: day, Valid: true},
		Rate:   rateNumeric,
		IsReal: rate.IsReal,
		Source: strings.TrimSpace(rate.Source),
	}); err != nil {
		return apperr.Internal("upsert fx rate failed", err, map[string]string{
			"fiat": fiat,
			"day":  day.Format(time.DateOnly),
		})
	}

	return nil
}

func (r *fxRateRepository) ListByFiat(ctx context.Context, fiat string) ([]domain.FXRate, error) {
	fiat = strings.ToUpper(strings.TrimSpace(fiat))
	if fiat == "" {
		return nil, apperr.InvalidArgument("fiat is required", nil, apperr.FieldViolation{
			Field:       "fiat",
			Description: "required",
		})
	}

	rows, err := r.store.ListFXRatesByFiat(ctx, fiat)
	if err != nil {
		return nil, apperr.Internal("list fx rates failed", err, map[string]string{
			"fiat": fiat,
		})
	}

	out := make([]domain.FXRate, 0, len(rows))
	for _, row := range rows {
		rate := numericToDecimal(row.Rate)
		if rate == nil || !rate.IsPositive() {
			return nil, apperr.Internal("fx rate row contains invalid numeric value", nil, map[string]string{
				"fiat": row.Fiat,
			})
		}
		if !row.Day.Valid {
			return nil, apperr.Internal("fx rate row contains invalid day", nil, map[string]string{
				"fiat": row.Fiat,
			})
		}

		out = append(out, domain.FXRate{
			Fiat:   row.Fiat,
			Day:    dateOnlyUTC(row.Day.Time),
			Rate:   *rate,
			IsReal: row.IsReal,
			Source: row.Source,
		})
	}

	return out, nil
}

func dateOnlyUTC(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
