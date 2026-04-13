package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
)

func TestUserSettingsRepoIntegration_UpsertAndGet(t *testing.T) {
	store := setupIntegrationStore(t)
	repo := NewUserSettingsRepo(store)

	ctx := context.Background()
	userID := uuid.New()

	upserted, err := repo.Upsert(ctx, domain.UserSettings{
		UserID:       userID,
		FiatCurrency: "USD",
		Timezone:     "UTC",
	})
	if err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}
	if upserted.UserID != userID {
		t.Fatalf("unexpected user id after upsert: %s", upserted.UserID)
	}
	if upserted.FiatCurrency != "USD" || upserted.Timezone != "UTC" {
		t.Fatalf("unexpected settings after upsert: %+v", upserted)
	}

	got, err := repo.Get(ctx, userID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.FiatCurrency != "USD" || got.Timezone != "UTC" {
		t.Fatalf("unexpected settings after get: %+v", got)
	}
}
