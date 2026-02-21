package repository

import (
	"context"
	"errors"

	db "github.com/NightRunner/CryptoTax-Go/services/tax-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type taxProfileRepo struct {
	store db.Store
}

func NewTaxProfileRepo(store db.Store) domain.TaxProfileRepo {
	return &taxProfileRepo{store: store}
}

func (r *taxProfileRepo) Get(ctx context.Context, tenantID uuid.UUID) (domain.TaxProfile, error) {
	row, err := r.store.GetTaxProfile(ctx, tenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.TaxProfile{}, apperr.NotFound("tax profile not found", apperr.Resource{
				Type: "tax_profile",
				Name: tenantID.String(),
			}, err)
		}
		return domain.TaxProfile{}, apperr.Internal("get tax profile failed", err, map[string]string{
			"tenant_id": tenantID.String(),
		})
	}

	return domain.TaxProfile{
		TenantID:                    row.TenantID,
		Jurisdiction:                row.Jurisdiction,
		CostBasisMethod:             row.CostBasisMethod,
		Timezone:                    row.Timezone,
		TreatSwapAsDisposition:      row.TreatSwapAsDisposition,
		TreatCryptoFeeAsDisposition: row.TreatCryptoFeeAsDisposition,
		IncludeIncomeEvents:         row.IncludeIncomeEvents,
		AllowLossEventsDeduction:    row.AllowLossEventsDeduction,
		FailOnNegativeInventory:     row.FailOnNegativeInventory,
		FailOnMissingFiat:           row.FailOnMissingFiat,
		CreatedAt:                   fromTimestamptz(row.CreatedAt),
		UpdatedAt:                   fromTimestamptz(row.UpdatedAt),
	}, nil
}

func (r *taxProfileRepo) Upsert(ctx context.Context, profile domain.TaxProfile) (domain.TaxProfile, error) {
	row, err := r.store.UpsertTaxProfile(ctx, db.UpsertTaxProfileParams{
		TenantID:                    profile.TenantID,
		Jurisdiction:                profile.Jurisdiction,
		CostBasisMethod:             profile.CostBasisMethod,
		Timezone:                    profile.Timezone,
		TreatSwapAsDisposition:      profile.TreatSwapAsDisposition,
		TreatCryptoFeeAsDisposition: profile.TreatCryptoFeeAsDisposition,
		IncludeIncomeEvents:         profile.IncludeIncomeEvents,
		AllowLossEventsDeduction:    profile.AllowLossEventsDeduction,
		FailOnNegativeInventory:     profile.FailOnNegativeInventory,
		FailOnMissingFiat:           profile.FailOnMissingFiat,
	})
	if err != nil {
		return domain.TaxProfile{}, apperr.Internal("upsert tax profile failed", err, map[string]string{
			"tenant_id": profile.TenantID.String(),
		})
	}

	return domain.TaxProfile{
		TenantID:                    row.TenantID,
		Jurisdiction:                row.Jurisdiction,
		CostBasisMethod:             row.CostBasisMethod,
		Timezone:                    row.Timezone,
		TreatSwapAsDisposition:      row.TreatSwapAsDisposition,
		TreatCryptoFeeAsDisposition: row.TreatCryptoFeeAsDisposition,
		IncludeIncomeEvents:         row.IncludeIncomeEvents,
		AllowLossEventsDeduction:    row.AllowLossEventsDeduction,
		FailOnNegativeInventory:     row.FailOnNegativeInventory,
		FailOnMissingFiat:           row.FailOnMissingFiat,
		CreatedAt:                   fromTimestamptz(row.CreatedAt),
		UpdatedAt:                   fromTimestamptz(row.UpdatedAt),
	}, nil
}

var _ domain.TaxProfileRepo = (*taxProfileRepo)(nil)
