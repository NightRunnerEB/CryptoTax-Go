package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/google/uuid"
)

type taxProfileUC struct {
	repo                domain.TaxProfileRepo
	defaultJurisdiction string
	defaultTimezone     string
	defaultCostBasis    string
}

func NewTaxProfileUC(
	repo domain.TaxProfileRepo,
	defaultJurisdiction string,
	defaultTimezone string,
	defaultCostBasis string,
) domain.TaxProfileUseCase {
	return &taxProfileUC{
		repo:                repo,
		defaultJurisdiction: defaultJurisdiction,
		defaultTimezone:     defaultTimezone,
		defaultCostBasis:    defaultCostBasis,
	}
}

func (u *taxProfileUC) Get(ctx context.Context, tenantID uuid.UUID) (domain.TaxProfile, error) {
	if tenantID == uuid.Nil {
		return domain.TaxProfile{}, apperr.InvalidArgument(
			"invalid tenant id",
			nil,
			apperr.FieldViolation{Field: "tenant_id", Description: "required"},
		)
	}

	profile, err := u.repo.Get(ctx, tenantID)
	if err == nil {
		profile.Jurisdiction = normalizeDefault(profile.Jurisdiction, u.defaultJurisdiction)
		profile.Timezone = normalizeDefault(profile.Timezone, u.defaultTimezone)
		profile.CostBasisMethod = normalizeDefault(profile.CostBasisMethod, u.defaultCostBasis)
		return profile, nil
	}

	if isNotFound(err) {
		return domain.TaxProfile{
			TenantID:                    tenantID,
			Jurisdiction:                u.defaultJurisdiction,
			CostBasisMethod:             u.defaultCostBasis,
			Timezone:                    u.defaultTimezone,
			TreatSwapAsDisposition:      false,
			TreatCryptoFeeAsDisposition: defaultTreatCryptoFeeAsDisposition,
			IncludeIncomeEvents:         defaultIncludeIncomeEvents,
			AllowLossEventsDeduction:    defaultAllowLossEventsDeduction,
			FailOnNegativeInventory:     defaultFailOnNegativeInventory,
			FailOnMissingFiat:           defaultFailOnMissingFiat,
		}, nil
	}

	return domain.TaxProfile{}, err
}

func (u *taxProfileUC) Upsert(ctx context.Context, profile domain.TaxProfile) (domain.TaxProfile, error) {
	if profile.TenantID == uuid.Nil {
		return domain.TaxProfile{}, apperr.InvalidArgument(
			"invalid tenant id",
			nil,
			apperr.FieldViolation{Field: "tenant_id", Description: "required"},
		)
	}

	profile.Jurisdiction = normalizeDefault(profile.Jurisdiction, u.defaultJurisdiction)
	profile.Timezone = normalizeDefault(profile.Timezone, u.defaultTimezone)
	profile.CostBasisMethod = normalizeDefault(profile.CostBasisMethod, u.defaultCostBasis)

	if err := validateTaxProfile(profile); err != nil {
		return domain.TaxProfile{}, err
	}

	return u.repo.Upsert(ctx, profile)
}

func normalizeDefault(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func validateTaxProfile(profile domain.TaxProfile) error {
	if profile.Jurisdiction == "" {
		return apperr.InvalidArgument(
			"invalid jurisdiction",
			nil,
			apperr.FieldViolation{Field: "jurisdiction", Description: "required"},
		)
	}
	if profile.CostBasisMethod == "" {
		return apperr.InvalidArgument(
			"invalid cost basis method",
			nil,
			apperr.FieldViolation{Field: "cost_basis_method", Description: "required"},
		)
	}
	if profile.Timezone == "" {
		return apperr.InvalidArgument(
			"invalid timezone",
			nil,
			apperr.FieldViolation{Field: "timezone", Description: "required"},
		)
	}
	if _, err := time.LoadLocation(profile.Timezone); err != nil {
		return apperr.InvalidArgument(
			"invalid timezone",
			err,
			apperr.FieldViolation{Field: "timezone", Description: "unknown timezone"},
		)
	}
	return nil
}

func isNotFound(err error) bool {
	var ae *apperr.Error
	return errors.As(err, &ae) && ae.Code == apperr.ErrNotFound
}
