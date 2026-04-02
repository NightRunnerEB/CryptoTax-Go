package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/mocks"
)

func TestTaxProfileUC_Upsert_NormalizesAndPersists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockTaxProfileRepo(ctrl)
	uc := NewTaxProfileUC(repo)
	tenantID := uuid.New()

	repo.EXPECT().Upsert(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got domain.TaxProfile) error {
		if got.TenantID != tenantID {
			t.Fatalf("tenant_id mismatch: got %s want %s", got.TenantID, tenantID)
		}
		if got.FirstName != "Ivan" {
			t.Fatalf("first_name not normalized: %q", got.FirstName)
		}
		if got.LastName != "Petrov" {
			t.Fatalf("last_name not normalized: %q", got.LastName)
		}
		if got.INN != "123456789012" {
			t.Fatalf("inn not normalized: %q", got.INN)
		}
		if got.TaxResidencyStatus != domain.Resident {
			t.Fatalf("tax_residency_status not normalized: %s", got.TaxResidencyStatus)
		}
		if got.TaxPayerType != domain.INDIVIDUAL {
			t.Fatalf("taxpayer_type not normalized: %s", got.TaxPayerType)
		}
		if got.Timezone != "Europe/Moscow" {
			t.Fatalf("timezone mismatch: %q", got.Timezone)
		}
		if len(got.Wallets) != 2 {
			t.Fatalf("wallets length mismatch: got %d want 2", len(got.Wallets))
		}
		if got.Wallets[0] != domain.Wallet("0xabc") || got.Wallets[1] != domain.Wallet("binance:123") {
			t.Fatalf("wallets mismatch: %+v", got.Wallets)
		}
		return nil
	}).Times(1)

	err := uc.Upsert(context.Background(), domain.TaxProfile{
		TenantID:           tenantID,
		INN:                " 123456789012 ",
		LastName:           " Petrov ",
		FirstName:          " Ivan ",
		MiddleName:         " Ivanovich ",
		Timezone:           "Europe/Moscow",
		Phone:              " +79990000000 ",
		Wallets:            []domain.Wallet{" 0xabc ", "   ", "binance:123"},
		TaxResidencyStatus: " resident ",
		TaxPayerType:       " individual ",
	})
	if err != nil {
		t.Fatalf("Upsert() unexpected error: %v", err)
	}
}

func TestTaxProfileUC_Upsert_InvalidTimezone(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockTaxProfileRepo(ctrl)
	uc := NewTaxProfileUC(repo)

	err := uc.Upsert(context.Background(), domain.TaxProfile{
		TenantID:           uuid.New(),
		INN:                "123456789012",
		LastName:           "Petrov",
		FirstName:          "Ivan",
		Timezone:           "Mars/Colony",
		TaxResidencyStatus: domain.Resident,
		TaxPayerType:       domain.INDIVIDUAL,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.ErrInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestTaxProfileUC_Get_InvalidTenantID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockTaxProfileRepo(ctrl)
	uc := NewTaxProfileUC(repo)

	_, err := uc.Get(context.Background(), uuid.Nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.ErrInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestTaxProfileUC_Delete_InvalidTenantID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockTaxProfileRepo(ctrl)
	uc := NewTaxProfileUC(repo)

	err := uc.Delete(context.Background(), uuid.Nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.ErrInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}
