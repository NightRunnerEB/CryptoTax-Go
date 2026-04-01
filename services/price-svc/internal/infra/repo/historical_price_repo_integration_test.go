package repository

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

func TestHistoricalPriceRepoIntegration_UpsertGetAndBatch(t *testing.T) {
	store := setupIntegrationStore(t)
	repo := NewHistoricalPriceRepo(store)

	ctx := context.Background()
	bucket1 := time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC)
	bucket2 := time.Date(2026, 3, 25, 11, 0, 0, 0, time.UTC)

	g1 := 300
	p1 := decimal.RequireFromString("100.5")
	if err := repo.Upsert(ctx, domain.HistoricalPrice{
		CoinID:             "bitcoin",
		Time:               bucket1,
		PriceUsd:           &p1,
		GranularitySeconds: &g1,
	}); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	got, err := repo.Get(ctx, "bitcoin", bucket1)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.PriceUsd == nil || !got.PriceUsd.Equal(p1) {
		t.Fatalf("unexpected price after Get: %+v", got.PriceUsd)
	}

	g2 := 3600
	p2 := decimal.RequireFromString("200.75")
	if err := repo.UpsertBatch(ctx, []domain.HistoricalPrice{
		{
			CoinID:             "bitcoin",
			Time:               bucket2,
			PriceUsd:           &p2,
			GranularitySeconds: &g2,
		},
	}); err != nil {
		t.Fatalf("UpsertBatch failed: %v", err)
	}

	rows, err := repo.GetBatch(ctx, []domain.PriceKey{
		{CoinID: "bitcoin", BucketStartUtc: bucket1},
		{CoinID: "bitcoin", BucketStartUtc: bucket2},
	})
	if err != nil {
		t.Fatalf("GetBatch failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].PriceUsd == nil || !rows[0].PriceUsd.Equal(p1) {
		t.Fatalf("unexpected row[0] price: %+v", rows[0].PriceUsd)
	}
	if rows[1].PriceUsd == nil || !rows[1].PriceUsd.Equal(p2) {
		t.Fatalf("unexpected row[1] price: %+v", rows[1].PriceUsd)
	}
}

func TestHistoricalPriceRepoIntegration_GetNotFound(t *testing.T) {
	store := setupIntegrationStore(t)
	repo := NewHistoricalPriceRepo(store)

	_, err := repo.Get(context.Background(), "bitcoin", time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected not found error, got nil")
	}
	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.ErrNotFound {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}
