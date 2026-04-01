package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	db "github.com/NightRunner/CryptoTax-Go/services/price-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

func TestHistoricalPriceRepo_Upsert_InvalidCoinID(t *testing.T) {
	repo := NewHistoricalPriceRepo(&fakeStore{})
	gs := 300
	price := decimal.NewFromInt(100)

	err := repo.Upsert(context.Background(), domain.HistoricalPrice{
		CoinID:             "",
		Time:               time.Now().UTC(),
		PriceUsd:           &price,
		GranularitySeconds: &gs,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.ErrInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestHistoricalPriceRepo_Upsert_CallsStore(t *testing.T) {
	called := false
	store := &fakeStore{
		upsertHistoricalPriceFn: func(_ context.Context, arg db.UpsertHistoricalPriceParams) error {
			called = true
			if arg.CoinID != "bitcoin" {
				t.Fatalf("unexpected coin id: %s", arg.CoinID)
			}
			if !arg.BucketStartUtc.Valid {
				t.Fatal("bucket start must be valid")
			}
			return nil
		},
	}

	repo := NewHistoricalPriceRepo(store)
	gs := 300
	price := decimal.NewFromInt(100)

	err := repo.Upsert(context.Background(), domain.HistoricalPrice{
		CoinID:             "bitcoin",
		Time:               time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC),
		PriceUsd:           &price,
		GranularitySeconds: &gs,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected store.UpsertHistoricalPrice call")
	}
}

func TestHistoricalPriceRepo_Get_NotFound(t *testing.T) {
	store := &fakeStore{
		getHistoricalPriceFn: func(context.Context, db.GetHistoricalPriceParams) (db.HistoricalPrice, error) {
			return db.HistoricalPrice{}, pgx.ErrNoRows
		},
	}
	repo := NewHistoricalPriceRepo(store)

	_, err := repo.Get(context.Background(), "bitcoin", time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.ErrNotFound {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

func TestHistoricalPriceRepo_GetBatch_EmptyKeys(t *testing.T) {
	called := false
	store := &fakeStore{
		getHistoricalPricesBatchFn: func(context.Context, db.GetHistoricalPricesBatchParams) ([]db.GetHistoricalPricesBatchRow, error) {
			called = true
			return nil, nil
		},
	}
	repo := NewHistoricalPriceRepo(store)

	out, err := repo.GetBatch(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != nil {
		t.Fatalf("expected nil output, got %v", out)
	}
	if called {
		t.Fatal("store must not be called for empty keys")
	}
}

func TestHistoricalPriceRepo_UpsertBatch_InvalidInput(t *testing.T) {
	repo := NewHistoricalPriceRepo(&fakeStore{})
	gs := 300

	err := repo.UpsertBatch(context.Background(), []domain.HistoricalPrice{
		{
			CoinID:             "bitcoin",
			Time:               time.Now().UTC(),
			PriceUsd:           nil,
			GranularitySeconds: &gs,
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.ErrInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestHistoricalPriceRepo_GetBatch_MapsRows(t *testing.T) {
	price := decimal.RequireFromString("123.456")
	n, err := decimalToNumeric(&price)
	if err != nil {
		t.Fatalf("decimalToNumeric error: %v", err)
	}

	store := &fakeStore{
		getHistoricalPricesBatchFn: func(context.Context, db.GetHistoricalPricesBatchParams) ([]db.GetHistoricalPricesBatchRow, error) {
			gs := int32(300)
			return []db.GetHistoricalPricesBatchRow{{
				CoinID:             "bitcoin",
				BucketStartUtc:     pgtype.Timestamptz{Time: time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC), Valid: true},
				PriceUsd:           n,
				GranularitySeconds: &gs,
			}}, nil
		},
	}

	repo := NewHistoricalPriceRepo(store)
	out, err := repo.GetBatch(context.Background(), []domain.PriceKey{{
		CoinID:         "bitcoin",
		BucketStartUtc: time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC),
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected one row, got %d", len(out))
	}
	if out[0].PriceUsd == nil || !out[0].PriceUsd.Equal(price) {
		t.Fatalf("unexpected mapped price: %+v", out[0].PriceUsd)
	}
	if out[0].GranularitySeconds == nil || *out[0].GranularitySeconds != 300 {
		t.Fatalf("unexpected granularity: %+v", out[0].GranularitySeconds)
	}
}

func TestHistoricalPriceRepo_GetBatch_StoreError(t *testing.T) {
	store := &fakeStore{
		getHistoricalPricesBatchFn: func(context.Context, db.GetHistoricalPricesBatchParams) ([]db.GetHistoricalPricesBatchRow, error) {
			return nil, errors.New("db down")
		},
	}
	repo := NewHistoricalPriceRepo(store)
	_, err := repo.GetBatch(context.Background(), []domain.PriceKey{{
		CoinID:         "bitcoin",
		BucketStartUtc: time.Now().UTC(),
	}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.ErrInternal {
		t.Fatalf("expected INTERNAL_ERROR, got %v", err)
	}
}
