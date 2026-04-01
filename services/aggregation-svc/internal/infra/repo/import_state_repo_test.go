package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
)

func TestImportStateRepo_Get_NotFound(t *testing.T) {
	t.Parallel()

	repo := NewImportStateRepo(&fakeStore{
		getImportStateFn: func(context.Context, db.GetAggregationImportStateParams) (db.AggregationImportState, error) {
			return db.AggregationImportState{}, pgx.ErrNoRows
		},
	})

	_, err := repo.Get(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertRepoErrorCode(t, err, apperr.ErrNotFound)
}

func TestImportStateRepo_Get_MapsCompletedAt(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New()
	importID := uuid.New()
	completedAt := time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC)
	repo := NewImportStateRepo(&fakeStore{
		getImportStateFn: func(context.Context, db.GetAggregationImportStateParams) (db.AggregationImportState, error) {
			return db.AggregationImportState{
				TenantID:    tenantID,
				ImportID:    importID,
				EventID:     uuid.New(),
				Status:      string(domain.ImportStatusCompleted),
				StartedAt:   pgtype.Timestamptz{Time: completedAt.Add(-time.Hour), Valid: true},
				CompletedAt: pgtype.Timestamptz{Time: completedAt, Valid: true},
			}, nil
		},
	})

	state, err := repo.Get(context.Background(), tenantID, importID)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if state.CompletedAt == nil || !state.CompletedAt.Equal(completedAt) {
		t.Fatalf("unexpected completed_at: %+v", state.CompletedAt)
	}
}

func TestImportStateRepo_MarkFailed_EmptyMessage(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New()
	importID := uuid.New()

	repo := NewImportStateRepo(&fakeStore{
		markFailedFn: func(_ context.Context, arg db.MarkAggregationImportStateFailedParams) error {
			if arg.TenantID != tenantID || arg.ImportID != importID {
				t.Fatalf("unexpected ids: %+v", arg)
			}
			if arg.Error == nil || *arg.Error != "unknown error" {
				t.Fatalf("unexpected error message: %+v", arg.Error)
			}
			return nil
		},
	})

	if err := repo.MarkFailed(context.Background(), tenantID, importID, ""); err != nil {
		t.Fatalf("MarkFailed returned error: %v", err)
	}
}

func TestImportStateRepo_MarkCompleted_WrapsInternal(t *testing.T) {
	t.Parallel()

	repo := NewImportStateRepo(&fakeStore{
		markCompletedFn: func(context.Context, db.MarkAggregationImportStateCompletedParams) error {
			return errors.New("db unavailable")
		},
	})

	err := repo.MarkCompleted(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertRepoErrorCode(t, err, apperr.ErrInternal)
}
