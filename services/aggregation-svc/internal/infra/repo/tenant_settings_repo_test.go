package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
)

func TestTenantSettingsRepo_Get_NotFound(t *testing.T) {
	t.Parallel()

	repo := NewTenantSettingsRepo(&fakeStore{
		getTenantSettingsFn: func(context.Context, uuid.UUID) (db.TenantSetting, error) {
			return db.TenantSetting{}, pgx.ErrNoRows
		},
	})

	_, err := repo.Get(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertRepoErrorCode(t, err, apperr.ErrNotFound)
}

func TestTenantSettingsRepo_UpsertAndGet_Success(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New()
	updatedAt := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	store := &fakeStore{
		upsertTenantSettingsFn: func(_ context.Context, arg db.UpsertTenantSettingsParams) (db.TenantSetting, error) {
			if arg.TenantID != tenantID {
				t.Fatalf("unexpected tenant id: %s", arg.TenantID)
			}
			if arg.FiatCurrency != "USD" || arg.Timezone != "UTC" {
				t.Fatalf("unexpected settings params: %+v", arg)
			}
			return db.TenantSetting{
				TenantID:     arg.TenantID,
				FiatCurrency: arg.FiatCurrency,
				Timezone:     arg.Timezone,
				UpdatedAt:    pgtype.Timestamptz{Time: updatedAt, Valid: true},
			}, nil
		},
		getTenantSettingsFn: func(_ context.Context, id uuid.UUID) (db.TenantSetting, error) {
			if id != tenantID {
				t.Fatalf("unexpected tenant id in get: %s", id)
			}
			return db.TenantSetting{
				TenantID:     id,
				FiatCurrency: "USD",
				Timezone:     "UTC",
				UpdatedAt:    pgtype.Timestamptz{Time: updatedAt, Valid: true},
			}, nil
		},
	}

	repo := NewTenantSettingsRepo(store)
	upserted, err := repo.Upsert(context.Background(), domain.TenantSettings{
		TenantID:     tenantID,
		FiatCurrency: "USD",
		Timezone:     "UTC",
	})
	if err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}
	if upserted.UpdatedAt.IsZero() || !upserted.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("unexpected updated_at after upsert: %v", upserted.UpdatedAt)
	}

	got, err := repo.Get(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.FiatCurrency != "USD" || got.Timezone != "UTC" {
		t.Fatalf("unexpected get result: %+v", got)
	}
}
