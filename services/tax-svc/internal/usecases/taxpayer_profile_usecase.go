package usecase

import (
	"context"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/google/uuid"
)

type taxpayerProfileUC struct {
	repo domain.TaxpayerProfileRepo
}

func NewTaxpayerProfileUC(repo domain.TaxpayerProfileRepo) domain.TaxpayerProfileUseCase {
	return &taxpayerProfileUC{repo: repo}
}

func (u *taxpayerProfileUC) Get(ctx context.Context, tenantID uuid.UUID) (domain.TaxpayerProfile, error) {
	if tenantID == uuid.Nil {
		return domain.TaxpayerProfile{}, apperr.InvalidArgument(
			"invalid tenant id",
			nil,
			apperr.FieldViolation{Field: "tenant_id", Description: "required"},
		)
	}

	profile, err := u.repo.Get(ctx, tenantID)
	if err == nil {
		return profile, nil
	}
	if isNotFound(err) {
		return domain.TaxpayerProfile{TenantID: tenantID}, nil
	}
	return domain.TaxpayerProfile{}, err
}

func (u *taxpayerProfileUC) Upsert(ctx context.Context, profile domain.TaxpayerProfile) (domain.TaxpayerProfile, error) {
	if profile.TenantID == uuid.Nil {
		return domain.TaxpayerProfile{}, apperr.InvalidArgument(
			"invalid tenant id",
			nil,
			apperr.FieldViolation{Field: "tenant_id", Description: "required"},
		)
	}
	return u.repo.Upsert(ctx, profile)
}
