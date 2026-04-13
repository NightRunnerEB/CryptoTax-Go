package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
)

func TestImportStateRepoIntegration_Lifecycle(t *testing.T) {
	store := setupIntegrationStore(t)
	repo := NewImportStateRepo(store)

	ctx := context.Background()
	userID := uuid.New()
	importID := uuid.New()
	eventID := uuid.New()

	if err := repo.UpsertProcessing(ctx, domain.AggregationImportState{
		UserID:   userID,
		ImportID: importID,
		EventId:  eventID,
		Status:   domain.ImportStatusProcessing,
	}); err != nil {
		t.Fatalf("UpsertProcessing returned error: %v", err)
	}

	state, err := repo.Get(ctx, userID, importID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if state.Status != domain.ImportStatusProcessing {
		t.Fatalf("unexpected status after processing upsert: %s", state.Status)
	}

	if err := repo.MarkFailed(ctx, userID, importID, "valuation failed"); err != nil {
		t.Fatalf("MarkFailed returned error: %v", err)
	}
	state, err = repo.Get(ctx, userID, importID)
	if err != nil {
		t.Fatalf("Get after MarkFailed returned error: %v", err)
	}
	if state.Status != domain.ImportStatusFailed {
		t.Fatalf("unexpected status after failed mark: %s", state.Status)
	}

	if err := repo.MarkCompleted(ctx, userID, importID); err != nil {
		t.Fatalf("MarkCompleted returned error: %v", err)
	}
	state, err = repo.Get(ctx, userID, importID)
	if err != nil {
		t.Fatalf("Get after MarkCompleted returned error: %v", err)
	}
	if state.Status != domain.ImportStatusCompleted {
		t.Fatalf("unexpected status after completed mark: %s", state.Status)
	}
	if state.CompletedAt == nil || state.CompletedAt.Before(time.Now().Add(-time.Minute)) {
		t.Fatalf("unexpected completed_at: %v", state.CompletedAt)
	}
}
