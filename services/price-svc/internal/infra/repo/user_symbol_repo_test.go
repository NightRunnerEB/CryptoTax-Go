package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"

	db "github.com/NightRunner/CryptoTax-Go/services/price-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

func TestUserSymbolRepo_Upsert_InvalidInput(t *testing.T) {
	repo := NewUserSymbolRepo(&fakeStore{})

	err := repo.Upsert(context.Background(), domain.UserSymbol{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.ErrInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestUserSymbolRepo_Upsert_CallsStore(t *testing.T) {
	userID := uuid.New()
	called := false

	store := &fakeStore{
		upsertUserSymbolFn: func(_ context.Context, arg db.UpsertUserSymbolParams) error {
			called = true
			if arg.UserID != userID || arg.Source != "MEXC" || arg.Symbol != "BTC" || arg.CoinID != "bitcoin" {
				t.Fatalf("unexpected args: %+v", arg)
			}
			return nil
		},
	}
	repo := NewUserSymbolRepo(store)

	err := repo.Upsert(context.Background(), domain.UserSymbol{
		UserID: userID,
		Source: "MEXC",
		Symbol: "BTC",
		CoinID: "bitcoin",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected store upsert call")
	}
}

func TestUserSymbolRepo_Delete_NotFound(t *testing.T) {
	userID := uuid.New()
	store := &fakeStore{
		deleteUserSymbolFn: func(context.Context, db.DeleteUserSymbolParams) (int64, error) {
			return 0, nil
		},
	}
	repo := NewUserSymbolRepo(store)

	err := repo.Delete(context.Background(), userID, "MEXC", "BTC")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.ErrNotFound {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

func TestUserSymbolRepo_GetList_EmptySymbols(t *testing.T) {
	called := false
	store := &fakeStore{
		getUserSymbolsFn: func(context.Context, db.GetUserSymbolsParams) ([]db.UserSymbol, error) {
			called = true
			return nil, nil
		},
	}
	repo := NewUserSymbolRepo(store)

	out, err := repo.GetList(context.Background(), uuid.New(), "MEXC", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != nil {
		t.Fatalf("expected nil output, got %v", out)
	}
	if called {
		t.Fatal("store must not be called for empty symbols")
	}
}

func TestUserSymbolRepo_GetListBySource_MapsRows(t *testing.T) {
	userID := uuid.New()
	store := &fakeStore{
		listUserSymbolsBySourceFn: func(context.Context, db.ListUserSymbolsBySourceParams) ([]db.UserSymbol, error) {
			return []db.UserSymbol{{
				UserID: userID,
				Source: "MEXC",
				Symbol: "BTC",
				CoinID: "bitcoin",
			}}, nil
		},
	}
	repo := NewUserSymbolRepo(store)

	out, err := repo.GetListBySource(context.Background(), userID, "MEXC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected one row, got %d", len(out))
	}
	if out[0].CoinID != "bitcoin" {
		t.Fatalf("unexpected value: %+v", out[0])
	}
}
