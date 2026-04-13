package usecases

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
)

type TaxProfileUC struct {
	repo domain.TaxProfileRepo
}

func NewTaxProfileUC(repo domain.TaxProfileRepo) *TaxProfileUC {
	return &TaxProfileUC{
		repo: repo,
	}
}

func (uc *TaxProfileUC) Upsert(ctx context.Context, p domain.TaxProfile) error {
	normalized, err := validateAndNormalizeTaxProfile(p)
	if err != nil {
		return err
	}

	return uc.repo.Upsert(ctx, normalized)
}

func validateAndNormalizeTaxProfile(p domain.TaxProfile) (domain.TaxProfile, error) {
	if p.UserID == uuid.Nil {
		return domain.TaxProfile{}, apperr.InvalidArgument("invalid user id", nil, apperr.FieldViolation{
			Field:       "user_id",
			Description: "required",
		})
	}

	p.INN = strings.TrimSpace(p.INN)
	p.LastName = strings.TrimSpace(p.LastName)
	p.FirstName = strings.TrimSpace(p.FirstName)
	p.MiddleName = strings.TrimSpace(p.MiddleName)
	p.Phone = strings.TrimSpace(p.Phone)

	if p.FirstName == "" {
		return domain.TaxProfile{}, apperr.InvalidArgument("invalid first_name", nil, apperr.FieldViolation{
			Field:       "first_name",
			Description: "required",
		})
	}
	if p.LastName == "" {
		return domain.TaxProfile{}, apperr.InvalidArgument("invalid last_name", nil, apperr.FieldViolation{
			Field:       "last_name",
			Description: "required",
		})
	}
	if p.INN == "" {
		return domain.TaxProfile{}, apperr.InvalidArgument("invalid inn", nil, apperr.FieldViolation{
			Field:       "inn",
			Description: "required",
		})
	}

	p.Timezone = strings.TrimSpace(p.Timezone)
	if p.Timezone == "" {
		return domain.TaxProfile{}, apperr.InvalidArgument("invalid timezone", nil, apperr.FieldViolation{
			Field:       "timezone",
			Description: "required",
		})
	}
	if _, err := time.LoadLocation(p.Timezone); err != nil {
		return domain.TaxProfile{}, apperr.InvalidArgument("invalid timezone", err, apperr.FieldViolation{
			Field:       "timezone",
			Description: "must be valid IANA timezone",
		})
	}

	p.TaxResidencyStatus = domain.TaxResidency(strings.ToUpper(strings.TrimSpace(string(p.TaxResidencyStatus))))
	if p.TaxResidencyStatus == "" {
		return domain.TaxProfile{}, apperr.InvalidArgument("invalid tax_residency_status", nil, apperr.FieldViolation{
			Field:       "tax_residency_status",
			Description: "required",
		})
	}
	if err := p.TaxResidencyStatus.Validate(); err != nil {
		return domain.TaxProfile{}, apperr.InvalidArgument("invalid tax_residency_status", err, apperr.FieldViolation{
			Field:       "tax_residency_status",
			Description: "unsupported value",
		})
	}

	p.TaxPayerType = domain.TaxPayerType(strings.ToUpper(strings.TrimSpace(string(p.TaxPayerType))))
	if p.TaxPayerType == "" {
		return domain.TaxProfile{}, apperr.InvalidArgument("invalid taxpayer_type", nil, apperr.FieldViolation{
			Field:       "taxpayer_type",
			Description: "required",
		})
	}
	if err := p.TaxPayerType.Validate(); err != nil {
		return domain.TaxProfile{}, apperr.InvalidArgument("invalid taxpayer_type", err, apperr.FieldViolation{
			Field:       "taxpayer_type",
			Description: "unsupported value",
		})
	}

	wallets := make([]domain.Wallet, 0, len(p.Wallets))
	for _, wallet := range p.Wallets {
		w := strings.TrimSpace(string(wallet))
		if w == "" {
			continue
		}
		wallets = append(wallets, domain.Wallet(w))
	}
	p.Wallets = wallets

	return p, nil
}

func (uc *TaxProfileUC) Get(ctx context.Context, userID uuid.UUID) (domain.TaxProfile, error) {
	if userID == uuid.Nil {
		return domain.TaxProfile{}, apperr.InvalidArgument("invalid user id", nil, apperr.FieldViolation{
			Field:       "user_id",
			Description: "required",
		})
	}
	return uc.repo.Get(ctx, userID)
}

func (uc *TaxProfileUC) Delete(ctx context.Context, userID uuid.UUID) error {
	if userID == uuid.Nil {
		return apperr.InvalidArgument("invalid user id", nil, apperr.FieldViolation{
			Field:       "user_id",
			Description: "required",
		})
	}
	return uc.repo.Delete(ctx, userID)
}

var _ domain.TaxProfileUseCase = (*TaxProfileUC)(nil)
