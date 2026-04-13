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

type userSettingsUC struct {
	repo domain.UserSettingsRepo
}

func NewUserSettingsUC(repo domain.UserSettingsRepo) domain.UserSettingsUseCase {
	return &userSettingsUC{
		repo: repo,
	}
}

func (u *userSettingsUC) Get(ctx context.Context, userID uuid.UUID) (domain.UserSettings, error) {
	if userID == uuid.Nil {
		return domain.UserSettings{}, apperr.InvalidArgument(
			"invalid user id",
			nil,
			apperr.FieldViolation{
				Field:       "user_id",
				Description: "required",
			},
		)
	}

	settings, err := u.repo.Get(ctx, userID)
	if err == nil {
		settings.FiatCurrency = normalizeFiatCurrency(settings.FiatCurrency)
		settings.Timezone = normalizeTimezone(settings.Timezone)
		return settings, nil
	}

	if isNotFound(err) {
		return domain.UserSettings{
			UserID:       userID,
			FiatCurrency: DefaultFiatCurrency,
			Timezone:     DefaultTimezone,
		}, nil
	}

	return domain.UserSettings{}, err
}

func (u *userSettingsUC) Upsert(ctx context.Context, settings domain.UserSettings) (domain.UserSettings, error) {
	if settings.UserID == uuid.Nil {
		return domain.UserSettings{}, apperr.InvalidArgument(
			"invalid user id",
			nil,
			apperr.FieldViolation{
				Field:       "user_id",
				Description: "required",
			},
		)
	}

	settings.FiatCurrency = normalizeFiatCurrency(settings.FiatCurrency)
	settings.Timezone = normalizeTimezone(settings.Timezone)

	if err := validateUserSettings(settings); err != nil {
		return domain.UserSettings{}, err
	}

	return u.repo.Upsert(ctx, settings)
}

func (u *userSettingsUC) ListSupportedFiatCurrencies(ctx context.Context) ([]domain.SupportedFiatCurrency, error) {
	_ = ctx
	return listSupportedFiatCurrencies(), nil
}

func normalizeFiatCurrency(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return DefaultFiatCurrency
	}
	return strings.ToUpper(value)
}

func normalizeTimezone(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return DefaultTimezone
	}
	return value
}

func validateUserSettings(settings domain.UserSettings) error {
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
	if !isSupportedFiatCurrency(settings.FiatCurrency) {
		return apperr.InvalidArgument(
			"invalid fiat currency",
			nil,
			apperr.FieldViolation{
				Field:       "fiat_currency",
				Description: "unsupported value",
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
