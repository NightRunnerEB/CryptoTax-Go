package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
)

func TestTenantSettingsRepoIntegration_UpsertAndGet(t *testing.T) {
	store := setupIntegrationStore(t)
	repo := NewTenantSettingsRepo(store)

	ctx := context.Background()
	tenantID := uuid.New()

	upserted, err := repo.Upsert(ctx, domain.TenantSettings{
		TenantID:     tenantID,
		FiatCurrency: "USD",
		Timezone:     "UTC",
	})
	if err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}
	if upserted.TenantID != tenantID {
		t.Fatalf("unexpected tenant id after upsert: %s", upserted.TenantID)
	}
	if upserted.FiatCurrency != "USD" || upserted.Timezone != "UTC" {
		t.Fatalf("unexpected settings after upsert: %+v", upserted)
	}

	got, err := repo.Get(ctx, tenantID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.FiatCurrency != "USD" || got.Timezone != "UTC" {
		t.Fatalf("unexpected settings after get: %+v", got)
	}
}
