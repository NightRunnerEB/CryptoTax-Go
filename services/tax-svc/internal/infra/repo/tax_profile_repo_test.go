package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/mock/gomock"

	db "github.com/NightRunner/CryptoTax-Go/services/tax-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/mocks"
)

func TestTaxProfileRepo_Upsert_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockStore(ctrl)
	repo := NewTaxProfileRepo(store)
	userID := uuid.New()

	store.EXPECT().UpsertTaxProfile(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, arg db.UpsertTaxProfileParams) (db.TaxProfile, error) {
		if arg.UserID != userID {
			t.Fatalf("user mismatch: got %s want %s", arg.UserID, userID)
		}
		if len(arg.Wallets) == 0 {
			t.Fatal("wallets json should not be empty")
		}
		return db.TaxProfile{}, nil
	}).Times(1)

	err := repo.Upsert(context.Background(), domain.TaxProfile{
		UserID:             userID,
		INN:                "123456789012",
		LastName:           "Petrov",
		FirstName:          "Ivan",
		Timezone:           "Europe/Moscow",
		Wallets:            []domain.Wallet{"0xabc"},
		TaxResidencyStatus: domain.Resident,
		TaxPayerType:       domain.INDIVIDUAL,
	})
	if err != nil {
		t.Fatalf("Upsert() unexpected error: %v", err)
	}
}

func TestTaxProfileRepo_Get_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockStore(ctrl)
	repo := NewTaxProfileRepo(store)
	userID := uuid.New()

	store.EXPECT().GetTaxProfile(gomock.Any(), userID).Return(db.TaxProfile{}, pgx.ErrNoRows).Times(1)

	_, err := repo.Get(context.Background(), userID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertTaxProfileRepoErrCode(t, err, apperr.ErrNotFound)
}

func TestTaxProfileRepo_Delete_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockStore(ctrl)
	repo := NewTaxProfileRepo(store)
	userID := uuid.New()

	store.EXPECT().DeleteTaxProfile(gomock.Any(), userID).Return(int64(0), nil).Times(1)

	err := repo.Delete(context.Background(), userID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertTaxProfileRepoErrCode(t, err, apperr.ErrNotFound)
}

func TestMapTaxProfileRow_InvalidWalletsJSON(t *testing.T) {
	_, err := mapTaxProfileRow(db.TaxProfile{
		UserID:  uuid.New(),
		Wallets: []byte("not-json"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertTaxProfileRepoErrCode(t, err, apperr.ErrInternal)
}

func assertTaxProfileRepoErrCode(t *testing.T, err error, want apperr.ErrorCode) {
	t.Helper()
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		t.Fatalf("expected app error, got %T: %v", err, err)
	}
	if ae.Code != want {
		t.Fatalf("error code mismatch: got %s want %s", ae.Code, want)
	}
}
