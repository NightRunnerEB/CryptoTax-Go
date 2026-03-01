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

type tenantSettingsRepo struct {
	store db.Store
}

func NewTenantSettingsRepo(store db.Store) domain.TenantSettingsRepo {
	return &tenantSettingsRepo{store: store}
}

func (r *tenantSettingsRepo) Get(ctx context.Context, tenantID uuid.UUID) (domain.TenantSettings, error) {
	row, err := r.store.GetTenantSettings(ctx, tenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.TenantSettings{}, apperr.NotFound("tenant settings not found", apperr.Resource{
				Type: "tenant_settings",
				Name: tenantID.String(),
			}, err)
		}
		return domain.TenantSettings{}, apperr.Internal("get tenant settings failed", err, map[string]string{
			"tenant_id": tenantID.String(),
		})
	}

	return domain.TenantSettings{
		TenantID:     row.TenantID,
		FiatCurrency: row.FiatCurrency,
		Timezone:     row.Timezone,
		UpdatedAt:    fromTimestamptz(row.UpdatedAt),
	}, nil
}

func (r *tenantSettingsRepo) Upsert(ctx context.Context, settings domain.TenantSettings) (domain.TenantSettings, error) {
	row, err := r.store.UpsertTenantSettings(ctx, db.UpsertTenantSettingsParams{
		TenantID:     settings.TenantID,
		FiatCurrency: settings.FiatCurrency,
		Timezone:     settings.Timezone,
	})
	if err != nil {
		return domain.TenantSettings{}, apperr.Internal("upsert tenant settings failed", err, map[string]string{
			"tenant_id": settings.TenantID.String(),
		})
	}

	return domain.TenantSettings{
		TenantID:     row.TenantID,
		FiatCurrency: row.FiatCurrency,
		Timezone:     row.Timezone,
		UpdatedAt:    fromTimestamptz(row.UpdatedAt),
	}, nil
}

var _ domain.TenantSettingsRepo = (*tenantSettingsRepo)(nil)
