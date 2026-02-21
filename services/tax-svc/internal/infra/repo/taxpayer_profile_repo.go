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

type taxpayerProfileRepo struct {
	store db.Store
}

func NewTaxpayerProfileRepo(store db.Store) domain.TaxpayerProfileRepo {
	return &taxpayerProfileRepo{store: store}
}

func (r *taxpayerProfileRepo) Get(ctx context.Context, tenantID uuid.UUID) (domain.TaxpayerProfile, error) {
	row, err := r.store.GetTaxpayerProfile(ctx, tenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.TaxpayerProfile{}, apperr.NotFound("taxpayer profile not found", apperr.Resource{
				Type: "taxpayer_profile",
				Name: tenantID.String(),
			}, err)
		}
		return domain.TaxpayerProfile{}, apperr.Internal("get taxpayer profile failed", err, map[string]string{
			"tenant_id": tenantID.String(),
		})
	}

	return domain.TaxpayerProfile{
		TenantID:           row.TenantID,
		INN:                row.Inn,
		LastName:           row.LastName,
		FirstName:          row.FirstName,
		MiddleName:         row.MiddleName,
		BirthDate:          fromDate(row.BirthDate),
		DocumentTypeCode:   row.DocumentTypeCode,
		DocumentNumber:     row.DocumentNumber,
		TaxResidencyStatus: row.TaxResidencyStatus,
		Phone:              row.Phone,
		CreatedAt:          fromTimestamptz(row.CreatedAt),
		UpdatedAt:          fromTimestamptz(row.UpdatedAt),
	}, nil
}

func (r *taxpayerProfileRepo) Upsert(ctx context.Context, profile domain.TaxpayerProfile) (domain.TaxpayerProfile, error) {
	row, err := r.store.UpsertTaxpayerProfile(ctx, db.UpsertTaxpayerProfileParams{
		TenantID:           profile.TenantID,
		Inn:                profile.INN,
		LastName:           profile.LastName,
		FirstName:          profile.FirstName,
		MiddleName:         profile.MiddleName,
		BirthDate:          toDate(profile.BirthDate),
		DocumentTypeCode:   profile.DocumentTypeCode,
		DocumentNumber:     profile.DocumentNumber,
		TaxResidencyStatus: profile.TaxResidencyStatus,
		Phone:              profile.Phone,
	})
	if err != nil {
		return domain.TaxpayerProfile{}, apperr.Internal("upsert taxpayer profile failed", err, map[string]string{
			"tenant_id": profile.TenantID.String(),
		})
	}

	return domain.TaxpayerProfile{
		TenantID:           row.TenantID,
		INN:                row.Inn,
		LastName:           row.LastName,
		FirstName:          row.FirstName,
		MiddleName:         row.MiddleName,
		BirthDate:          fromDate(row.BirthDate),
		DocumentTypeCode:   row.DocumentTypeCode,
		DocumentNumber:     row.DocumentNumber,
		TaxResidencyStatus: row.TaxResidencyStatus,
		Phone:              row.Phone,
		CreatedAt:          fromTimestamptz(row.CreatedAt),
		UpdatedAt:          fromTimestamptz(row.UpdatedAt),
	}, nil
}

var _ domain.TaxpayerProfileRepo = (*taxpayerProfileRepo)(nil)
