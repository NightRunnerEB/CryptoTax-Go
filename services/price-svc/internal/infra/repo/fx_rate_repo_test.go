package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	db "github.com/NightRunner/CryptoTax-Go/services/price-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

func TestFXRateRepo_Upsert_InvalidFiat(t *testing.T) {
	repo := NewFXRateRepo(&fakeStore{})
	err := repo.Upsert(context.Background(), domain.FXRate{
		Fiat: "",
		Day:  time.Now(),
		Rate: decimal.RequireFromString("1.23"),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.ErrInvalidArgument {
		t.Fatalf("expected INVALID_ARGUMENT, got %v", err)
	}
}

func TestFXRateRepo_Upsert_CallsStore(t *testing.T) {
	called := false
	store := &fakeStore{
		upsertFXRateFn: func(_ context.Context, arg db.UpsertFXRateParams) error {
			called = true
			if arg.Fiat != "RUB" {
				t.Fatalf("unexpected fiat: %s", arg.Fiat)
			}
			if !arg.Day.Valid {
				t.Fatal("day must be valid")
			}
			if !arg.IsReal {
				t.Fatal("expected IsReal=true")
			}
			if arg.Source != "cbr" {
				t.Fatalf("unexpected source: %s", arg.Source)
			}
			return nil
		},
	}
	repo := NewFXRateRepo(store)
	rate := decimal.RequireFromString("90.11")
	err := repo.Upsert(context.Background(), domain.FXRate{
		Fiat:   "rub",
		Day:    time.Date(2026, 4, 10, 18, 10, 0, 0, time.FixedZone("MSK", 3*3600)),
		Rate:   rate,
		IsReal: true,
		Source: "cbr",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected store.UpsertFXRate call")
	}
}

func TestFXRateRepo_ListByFiat_MapsRows(t *testing.T) {
	n, err := decimalToNumeric(ptrDecimal("92.34"))
	if err != nil {
		t.Fatalf("decimalToNumeric failed: %v", err)
	}

	store := &fakeStore{
		listFXRatesByFiatFn: func(context.Context, string) ([]db.FxRate, error) {
			return []db.FxRate{
				{
					Fiat:      "RUB",
					Day:       pgtype.Date{Time: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), Valid: true},
					Rate:      n,
					IsReal:    false,
					Source:    "carry",
					UpdatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
				},
			}, nil
		},
	}
	repo := NewFXRateRepo(store)

	out, err := repo.ListByFiat(context.Background(), "rub")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 row, got %d", len(out))
	}
	if out[0].Fiat != "RUB" || out[0].Source != "carry" || out[0].IsReal {
		t.Fatalf("unexpected row: %+v", out[0])
	}
	if !out[0].Rate.Equal(decimal.RequireFromString("92.34")) {
		t.Fatalf("unexpected rate: %s", out[0].Rate.String())
	}
}

func ptrDecimal(v string) *decimal.Decimal {
	d := decimal.RequireFromString(v)
	return &d
}
