package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	db "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
)

type userSettingsRepo struct {
	store db.Store
}

func NewUserSettingsRepo(store db.Store) domain.UserSettingsRepo {
	return &userSettingsRepo{store: store}
}

func (r *userSettingsRepo) Get(ctx context.Context, userID uuid.UUID) (domain.UserSettings, error) {
	row, err := r.store.GetUserSettings(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.UserSettings{}, apperr.NotFound("user settings not found", apperr.Resource{
				Type: "user_settings",
				Name: userID.String(),
			}, err)
		}
		return domain.UserSettings{}, apperr.Internal("get user settings failed", err, map[string]string{
			"user_id": userID.String(),
		})
	}

	return domain.UserSettings{
		UserID:       row.UserID,
		FiatCurrency: row.FiatCurrency,
		Timezone:     row.Timezone,
		UpdatedAt:    fromTimestamptz(row.UpdatedAt),
	}, nil
}

func (r *userSettingsRepo) Upsert(ctx context.Context, settings domain.UserSettings) (domain.UserSettings, error) {
	row, err := r.store.UpsertUserSettings(ctx, db.UpsertUserSettingsParams{
		UserID:       settings.UserID,
		FiatCurrency: settings.FiatCurrency,
		Timezone:     settings.Timezone,
	})
	if err != nil {
		return domain.UserSettings{}, apperr.Internal("upsert user settings failed", err, map[string]string{
			"user_id": settings.UserID.String(),
		})
	}

	return domain.UserSettings{
		UserID:       row.UserID,
		FiatCurrency: row.FiatCurrency,
		Timezone:     row.Timezone,
		UpdatedAt:    fromTimestamptz(row.UpdatedAt),
	}, nil
}

var _ domain.UserSettingsRepo = (*userSettingsRepo)(nil)
