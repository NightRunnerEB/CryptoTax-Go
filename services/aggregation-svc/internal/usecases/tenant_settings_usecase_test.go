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

func TestTenantSettingsUC_Get_NotFoundReturnsDefaults(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockTenantSettingsRepo(ctrl)
	uc := NewTenantSettingsUC(repo)

	tenantID := uuid.New()
	repo.EXPECT().
		Get(gomock.Any(), tenantID).
		Return(domain.TenantSettings{}, apperr.NotFound("not found", apperr.Resource{Type: "tenant_settings", Name: tenantID.String()}, nil))

	got, err := uc.Get(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.TenantID != tenantID {
		t.Fatalf("unexpected tenant id: %s", got.TenantID)
	}
	if got.FiatCurrency != DefaultFiatCurrency {
		t.Fatalf("unexpected default fiat: %s", got.FiatCurrency)
	}
	if got.Timezone != DefaultTimezone {
		t.Fatalf("unexpected default timezone: %s", got.Timezone)
	}
}

func TestTenantSettingsUC_Upsert_UnsupportedFiat(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockTenantSettingsRepo(ctrl)
	uc := NewTenantSettingsUC(repo)

	_, err := uc.Upsert(context.Background(), domain.TenantSettings{
		TenantID:     uuid.New(),
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

func TestTenantSettingsUC_Upsert_NormalizesAndDelegates(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockTenantSettingsRepo(ctrl)
	uc := NewTenantSettingsUC(repo)

	tenantID := uuid.New()
	repo.EXPECT().
		Upsert(gomock.Any(), gomock.AssignableToTypeOf(domain.TenantSettings{})).
		DoAndReturn(func(_ context.Context, settings domain.TenantSettings) (domain.TenantSettings, error) {
			if settings.TenantID != tenantID {
				t.Fatalf("unexpected tenant id: %s", settings.TenantID)
			}
			if settings.FiatCurrency != "RUB" {
				t.Fatalf("unexpected normalized fiat: %s", settings.FiatCurrency)
			}
			if settings.Timezone != "Europe/Moscow" {
				t.Fatalf("unexpected normalized timezone: %s", settings.Timezone)
			}
			return settings, nil
		})

	got, err := uc.Upsert(context.Background(), domain.TenantSettings{
		TenantID: tenantID,
	})
	if err != nil {
		t.Fatalf("Upsert returned error: %v", err)
	}
	if got.FiatCurrency != "RUB" || got.Timezone != "Europe/Moscow" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestTenantSettingsUC_ListSupportedFiatCurrencies_Deterministic(t *testing.T) {
	t.Parallel()

	uc := NewTenantSettingsUC(nil)
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
