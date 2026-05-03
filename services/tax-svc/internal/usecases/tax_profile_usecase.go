package usecases

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
)

var (
	innPattern   = regexp.MustCompile(`^\d{12}$`)
	oktmoPattern = regexp.MustCompile(`^\d{8}(\d{3})?$`)
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
	p.OKTMO = strings.TrimSpace(p.OKTMO)
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
	if !innPattern.MatchString(p.INN) {
		return domain.TaxProfile{}, apperr.InvalidArgument("invalid inn", nil, apperr.FieldViolation{
			Field:       "inn",
			Description: "must be 12 digits",
		})
	}
	if !isValidIndividualINN(p.INN) {
		return domain.TaxProfile{}, apperr.InvalidArgument("invalid inn", nil, apperr.FieldViolation{
			Field:       "inn",
			Description: "control digits do not match",
		})
	}
	if p.OKTMO == "" {
		return domain.TaxProfile{}, apperr.InvalidArgument("invalid oktmo", nil, apperr.FieldViolation{
			Field:       "oktmo",
			Description: "required",
		})
	}
	if !oktmoPattern.MatchString(p.OKTMO) {
		return domain.TaxProfile{}, apperr.InvalidArgument("invalid oktmo", nil, apperr.FieldViolation{
			Field:       "oktmo",
			Description: "must be 8 or 11 digits",
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

func isValidIndividualINN(inn string) bool {
	if len(inn) != 12 {
		return false
	}
	digits := make([]int, 12)
	for i := range len(inn) {
		digits[i] = int(inn[i] - '0')
	}

	check11 := checksumMod11Mod10(digits[:10], []int{7, 2, 4, 10, 3, 5, 9, 4, 6, 8})
	if check11 != digits[10] {
		return false
	}
	check12 := checksumMod11Mod10(digits[:11], []int{3, 7, 2, 4, 10, 3, 5, 9, 4, 6, 8})
	return check12 == digits[11]
}

func checksumMod11Mod10(digits []int, coeffs []int) int {
	sum := 0
	for i := range len(coeffs) {
		sum += digits[i] * coeffs[i]
	}
	return (sum % 11) % 10
}

var _ domain.TaxProfileUseCase = (*TaxProfileUC)(nil)
