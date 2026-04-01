package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
)

func TestAggregatedTransactionRepoIntegration_UpsertAndList(t *testing.T) {
	store := setupIntegrationStore(t)
	repo := NewAggregatedTransactionRepo(store)

	ctx := context.Background()
	tenantID := uuid.New()
	importID := uuid.New()
	t1 := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)

	err := repo.UpsertBatch(ctx, []domain.AggregatedTransaction{
		{
			ID:            uuid.New(),
			TenantID:      tenantID,
			ImportID:      importID,
			Source:        "MEXC",
			TimeUTC:       t1,
			Kind:          "Spot",
			InMoney:       &domain.MoneyLeg{Symbol: "BTC", CryptoAmount: "0.1", FiatAmount: strPtr("100")},
			TxFingerprint: "fp-it-1",
			CreatedAt:     t1,
		},
		{
			ID:            uuid.New(),
			TenantID:      tenantID,
			ImportID:      importID,
			Source:        "MEXC",
			TimeUTC:       t2,
			Kind:          "Spot",
			OutMoney:      &domain.MoneyLeg{Symbol: "BTC", CryptoAmount: "0.05", FiatAmount: strPtr("60")},
			TxFingerprint: "fp-it-2",
			CreatedAt:     t2,
		},
	})
	if err != nil {
		t.Fatalf("UpsertBatch returned error: %v", err)
	}

	pageByImport, err := repo.ListByImport(ctx, tenantID, importID, 50, 0)
	if err != nil {
		t.Fatalf("ListByImport returned error: %v", err)
	}
	if pageByImport.Total != 2 || len(pageByImport.Transactions) != 2 {
		t.Fatalf("unexpected page by import: %+v", pageByImport)
	}

	pageByRange, err := repo.ListByRange(ctx, tenantID, t1.Add(-time.Hour), t2.Add(time.Hour), 50, 0)
	if err != nil {
		t.Fatalf("ListByRange returned error: %v", err)
	}
	if pageByRange.Total != 2 || len(pageByRange.Transactions) != 2 {
		t.Fatalf("unexpected page by range: %+v", pageByRange)
	}
}
