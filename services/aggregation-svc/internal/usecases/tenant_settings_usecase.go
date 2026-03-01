package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
)

type tenantSettingsUC struct {
	repo domain.TenantSettingsRepo
}

func NewTenantSettingsUC(repo domain.TenantSettingsRepo) domain.TenantSettingsUseCase {
	return &tenantSettingsUC{
		repo: repo,
	}
}

func (u *tenantSettingsUC) Get(ctx context.Context, tenantID uuid.UUID) (domain.TenantSettings, error) {
	if tenantID == uuid.Nil {
		return domain.TenantSettings{}, apperr.InvalidArgument(
			"invalid tenant id",
			nil,
			apperr.FieldViolation{
				Field:       "tenant_id",
				Description: "required",
			},
		)
	}

	settings, err := u.repo.Get(ctx, tenantID)
	if err == nil {
		settings.FiatCurrency = normalizeFiatCurrency(settings.FiatCurrency)
		settings.Timezone = normalizeTimezone(settings.Timezone)
		return settings, nil
	}

	if isNotFound(err) {
		return domain.TenantSettings{
			TenantID:     tenantID,
			FiatCurrency: DefaultFiatCurrency,
			Timezone:     DefaultTimezone,
		}, nil
	}

	return domain.TenantSettings{}, err
}

func (u *tenantSettingsUC) Upsert(ctx context.Context, settings domain.TenantSettings) (domain.TenantSettings, error) {
	if settings.TenantID == uuid.Nil {
		return domain.TenantSettings{}, apperr.InvalidArgument(
			"invalid tenant id",
			nil,
			apperr.FieldViolation{
				Field:       "tenant_id",
				Description: "required",
			},
		)
	}

	settings.FiatCurrency = normalizeFiatCurrency(settings.FiatCurrency)
	settings.Timezone = normalizeTimezone(settings.Timezone)

	if err := validateTenantSettings(settings); err != nil {
		return domain.TenantSettings{}, err
	}

	return u.repo.Upsert(ctx, settings)
}

func normalizeFiatCurrency(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return DefaultFiatCurrency
	}
	return value
}

func normalizeTimezone(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return DefaultTimezone
	}
	return value
}

func validateTenantSettings(settings domain.TenantSettings) error {
	if settings.FiatCurrency == "" {
		return apperr.InvalidArgument(
			"invalid fiat currency",
			nil,
			apperr.FieldViolation{
				Field:       "fiat_currency",
				Description: "required",
			},
		)
	}

	if settings.Timezone == "" {
		return apperr.InvalidArgument(
			"invalid timezone",
			nil,
			apperr.FieldViolation{
				Field:       "timezone",
				Description: "required",
			},
		)
	}
	if _, err := time.LoadLocation(settings.Timezone); err != nil {
		return apperr.InvalidArgument(
			"invalid timezone",
			err,
			apperr.FieldViolation{
				Field:       "timezone",
				Description: "unknown timezone",
			},
		)
	}

	return nil
}

func isNotFound(err error) bool {
	var ae *apperr.Error
	return errors.As(err, &ae) && ae.Code == apperr.ErrNotFound
}
