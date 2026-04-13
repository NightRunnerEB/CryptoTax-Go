package usecase

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/mocks"
)

func TestUserSettingsUC_Get_NotFoundReturnsDefaults(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockUserSettingsRepo(ctrl)
	uc := NewUserSettingsUC(repo)

	userID := uuid.New()
	repo.EXPECT().
		Get(gomock.Any(), userID).
		Return(domain.UserSettings{}, apperr.NotFound("not found", apperr.Resource{Type: "user_settings", Name: userID.String()}, nil))

	got, err := uc.Get(context.Background(), userID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.UserID != userID {
		t.Fatalf("unexpected user id: %s", got.UserID)
	}
	if got.FiatCurrency != DefaultFiatCurrency {
		t.Fatalf("unexpected default fiat: %s", got.FiatCurrency)
	}
	if got.Timezone != DefaultTimezone {
		t.Fatalf("unexpected default timezone: %s", got.Timezone)
	}
}

func TestUserSettingsUC_Upsert_UnsupportedFiat(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockUserSettingsRepo(ctrl)
	uc := NewUserSettingsUC(repo)

	_, err := uc.Upsert(context.Background(), domain.UserSettings{
		UserID:       uuid.New(),
		FiatCurrency: "ABC",
		Timezone:     "UTC",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ae *apperr.Error
	if !errors.As(err, &ae) {
		t.Fatalf("expected *apperr.Error, got %T", err)
	}
	if ae.Code != apperr.ErrInvalidArgument {
		t.Fatalf("unexpected error code: %s", ae.Code)
	}
}

func TestUserSettingsUC_Upsert_NormalizesAndDelegates(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockUserSettingsRepo(ctrl)
	uc := NewUserSettingsUC(repo)

	userID := uuid.New()
	repo.EXPECT().
		Upsert(gomock.Any(), gomock.AssignableToTypeOf(domain.UserSettings{})).
		DoAndReturn(func(_ context.Context, settings domain.UserSettings) (domain.UserSettings, error) {
			if settings.UserID != userID {
				t.Fatalf("unexpected user id: %s", settings.UserID)
			}
			if settings.FiatCurrency != "RUB" {
				t.Fatalf("unexpected normalized fiat: %s", settings.FiatCurrency)
			}
			if settings.Timezone != "Europe/Moscow" {
				t.Fatalf("unexpected normalized timezone: %s", settings.Timezone)
			}
			return settings, nil
		})

	got, err := uc.Upsert(context.Background(), domain.UserSettings{
		UserID: userID,
	})
	if err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}
	if got.FiatCurrency != "RUB" || got.Timezone != "Europe/Moscow" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestUserSettingsUC_ListSupportedFiatCurrencies_Deterministic(t *testing.T) {
	t.Parallel()

	uc := NewUserSettingsUC(nil)
	got, err := uc.ListSupportedFiatCurrencies(context.Background())
	if err != nil {
		t.Fatalf("ListSupportedFiatCurrencies returned error: %v", err)
	}

	want := []domain.SupportedFiatCurrency{
		{Code: "USD", DisplayName: "US Dollar"},
		{Code: "RUB", DisplayName: "Russian Ruble"},
		{Code: "KZT", DisplayName: "Kazakhstani Tenge"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected currencies: got=%+v want=%+v", got, want)
	}
}
