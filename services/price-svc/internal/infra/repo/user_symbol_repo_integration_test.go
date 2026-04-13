package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

func TestUserSymbolRepoIntegration_UpsertListDelete(t *testing.T) {
	store := setupIntegrationStore(t)
	repo := NewUserSymbolRepo(store)

	ctx := context.Background()
	userID := uuid.New()

	if err := repo.Upsert(ctx, domain.UserSymbol{
		UserID: userID,
		Source: "MEXC",
		Symbol: "BTC",
		CoinID: "bitcoin",
	}); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	if err := repo.Upsert(ctx, domain.UserSymbol{
		UserID: userID,
		Source: "MEXC",
		Symbol: "ETH",
		CoinID: "ethereum",
	}); err != nil {
		t.Fatalf("second Upsert failed: %v", err)
	}

	list, err := repo.GetList(ctx, userID, "MEXC", []string{"BTC", "ETH"})
	if err != nil {
		t.Fatalf("GetList failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(list))
	}

	bySource, err := repo.GetListBySource(ctx, userID, "MEXC")
	if err != nil {
		t.Fatalf("GetListBySource failed: %v", err)
	}
	if len(bySource) != 2 {
		t.Fatalf("expected 2 rows by source, got %d", len(bySource))
	}

	if err := repo.Delete(ctx, userID, "MEXC", "BTC"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	err = repo.Delete(ctx, userID, "MEXC", "BTC")
	if err == nil {
		t.Fatal("expected not found on second delete, got nil")
	}
	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.ErrNotFound {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}
