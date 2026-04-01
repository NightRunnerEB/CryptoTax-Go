package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"

	db "github.com/NightRunner/CryptoTax-Go/services/price-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

func TestTenantSymbolRepo_Upsert_InvalidInput(t *testing.T) {
	repo := NewTenantSymbolRepo(&fakeStore{})

	err := repo.Upsert(context.Background(), domain.TenantSymbol{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.ErrInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestTenantSymbolRepo_Upsert_CallsStore(t *testing.T) {
	tenantID := uuid.New()
	called := false

	store := &fakeStore{
		upsertTenantSymbolFn: func(_ context.Context, arg db.UpsertTenantSymbolParams) error {
			called = true
			if arg.TenantID != tenantID || arg.Source != "MEXC" || arg.Symbol != "BTC" || arg.CoinID != "bitcoin" {
				t.Fatalf("unexpected args: %+v", arg)
			}
			return nil
		},
	}
	repo := NewTenantSymbolRepo(store)

	err := repo.Upsert(context.Background(), domain.TenantSymbol{
		TenantID: tenantID,
		Source:   "MEXC",
		Symbol:   "BTC",
		CoinID:   "bitcoin",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected store upsert call")
	}
}

func TestTenantSymbolRepo_Delete_NotFound(t *testing.T) {
	tenantID := uuid.New()
	store := &fakeStore{
		deleteTenantSymbolFn: func(context.Context, db.DeleteTenantSymbolParams) (int64, error) {
			return 0, nil
		},
	}
	repo := NewTenantSymbolRepo(store)

	err := repo.Delete(context.Background(), tenantID, "MEXC", "BTC")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.ErrNotFound {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

func TestTenantSymbolRepo_GetList_EmptySymbols(t *testing.T) {
	called := false
	store := &fakeStore{
		getTenantSymbolsFn: func(context.Context, db.GetTenantSymbolsParams) ([]db.TenantSymbol, error) {
			called = true
			return nil, nil
		},
	}
	repo := NewTenantSymbolRepo(store)

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

func TestTenantSymbolRepo_GetListBySource_MapsRows(t *testing.T) {
	tenantID := uuid.New()
	store := &fakeStore{
		listTenantSymbolsBySourceFn: func(context.Context, db.ListTenantSymbolsBySourceParams) ([]db.TenantSymbol, error) {
			return []db.TenantSymbol{{
				TenantID: tenantID,
				Source:   "MEXC",
				Symbol:   "BTC",
				CoinID:   "bitcoin",
			}}, nil
		},
	}
	repo := NewTenantSymbolRepo(store)

	out, err := repo.GetListBySource(context.Background(), tenantID, "MEXC")
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
