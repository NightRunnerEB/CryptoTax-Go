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

func TestUserSettingsRepo_Get_NotFound(t *testing.T) {
	t.Parallel()

	repo := NewUserSettingsRepo(&fakeStore{
		getUserSettingsFn: func(context.Context, uuid.UUID) (db.UserSetting, error) {
			return db.UserSetting{}, pgx.ErrNoRows
		},
	})

	_, err := repo.Get(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertRepoErrorCode(t, err, apperr.ErrNotFound)
}

func TestUserSettingsRepo_UpsertAndGet_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	updatedAt := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	store := &fakeStore{
		upsertUserSettingsFn: func(_ context.Context, arg db.UpsertUserSettingsParams) (db.UserSetting, error) {
			if arg.UserID != userID {
				t.Fatalf("unexpected user id: %s", arg.UserID)
			}
			if arg.FiatCurrency != "USD" || arg.Timezone != "UTC" {
				t.Fatalf("unexpected settings params: %+v", arg)
			}
			return db.UserSetting{
				UserID:       arg.UserID,
				FiatCurrency: arg.FiatCurrency,
				Timezone:     arg.Timezone,
				UpdatedAt:    pgtype.Timestamptz{Time: updatedAt, Valid: true},
			}, nil
		},
		getUserSettingsFn: func(_ context.Context, id uuid.UUID) (db.UserSetting, error) {
			if id != userID {
				t.Fatalf("unexpected user id in get: %s", id)
			}
			return db.UserSetting{
				UserID:       id,
				FiatCurrency: "USD",
				Timezone:     "UTC",
				UpdatedAt:    pgtype.Timestamptz{Time: updatedAt, Valid: true},
			}, nil
		},
	}

	repo := NewUserSettingsRepo(store)
	upserted, err := repo.Upsert(context.Background(), domain.UserSettings{
		UserID:       userID,
		FiatCurrency: "USD",
		Timezone:     "UTC",
	})
	if err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}
	if upserted.UpdatedAt.IsZero() || !upserted.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("unexpected updated_at after upsert: %v", upserted.UpdatedAt)
	}

	got, err := repo.Get(context.Background(), userID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.FiatCurrency != "USD" || got.Timezone != "UTC" {
		t.Fatalf("unexpected get result: %+v", got)
	}
}
