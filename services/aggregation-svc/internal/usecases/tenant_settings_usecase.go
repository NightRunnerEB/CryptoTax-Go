package usecase

import (
	"context"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
	"github.com/google/uuid"
)

type tenantSettingsUC struct {
	repo           domain.TenantSettingsRepo
	defaultFiat    string
	defaultTZ      string
	contextTimeout time.Duration
}

func NewTenantSettingsUC(
	repo domain.TenantSettingsRepo,
	defaultFiat string,
	defaultTimezone string,
	timeout time.Duration,
) domain.TenantSettingsUseCase {
	return &tenantSettingsUC{
		repo:           repo,
		defaultFiat:    defaultFiat,
		defaultTZ:      defaultTimezone,
		contextTimeout: timeout,
	}
}

func (u *tenantSettingsUC) Get(ctx context.Context, tenantID uuid.UUID) (domain.TenantSettings, error) {
	if u.contextTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, u.contextTimeout)
		defer cancel()
	}

	return domain.TenantSettings{}, apperr.Internal("method not implemented", nil, map[string]string{
		"tenant_id": tenantID.String(),
	})
}

func (u *tenantSettingsUC) Upsert(ctx context.Context, settings domain.TenantSettings) (domain.TenantSettings, error) {
	if u.contextTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, u.contextTimeout)
		defer cancel()
	}

	return domain.TenantSettings{}, apperr.Internal("method not implemented", nil, map[string]string{
		"tenant_id": settings.TenantID.String(),
	})
}
