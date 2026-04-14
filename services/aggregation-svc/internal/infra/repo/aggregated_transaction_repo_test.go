package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
)

func TestAggregatedTransactionRepo_UpsertBatch_DuplicateFingerprint(t *testing.T) {
	t.Parallel()

	repo := NewAggregatedTransactionRepo(&fakeStore{})
	userID := uuid.New()
	importID := uuid.New()

	txs := []domain.AggregatedTransaction{
		{ID: uuid.New(), UserID: userID, ImportID: importID, TxFingerprint: "fp-1"},
		{ID: uuid.New(), UserID: userID, ImportID: importID, TxFingerprint: "fp-1"},
	}

	err := repo.UpsertBatch(context.Background(), txs)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertRepoErrorCode(t, err, apperr.ErrInvalidArgument)
}

func TestAggregatedTransactionRepo_UpsertBatch_TrimFingerprintAndPersist(t *testing.T) {
	t.Parallel()

	var captured db.UpsertAggregatedTransactionParams
	store := &fakeStore{
		upsertAggregatedTxFn: func(_ context.Context, arg db.UpsertAggregatedTransactionParams) error {
			captured = arg
			return nil
		},
	}
	repo := NewAggregatedTransactionRepo(store)

	userID := uuid.New()
	importID := uuid.New()
	txID := uuid.New()
	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)

	err := repo.UpsertBatch(context.Background(), []domain.AggregatedTransaction{
		{
			ID:            txID,
			UserID:        userID,
			ImportID:      importID,
			Source:        "MEXC",
			TimeUTC:       now,
			Kind:          "Spot",
			InMoney:       &domain.MoneyLeg{Symbol: "BTC", CryptoAmount: "0.1"},
			TxFingerprint: "  fp-1  ",
			CreatedAt:     now,
		},
	})
	if err != nil {
		t.Fatalf("UpsertBatch returned error: %v", err)
	}
	if captured.ID != txID || captured.UserID != userID || captured.ImportID != importID {
		t.Fatalf("unexpected captured ids: %+v", captured)
	}
	if captured.TxFingerprint != "fp-1" {
		t.Fatalf("unexpected trimmed fingerprint: %q", captured.TxFingerprint)
	}
	if len(captured.InMoney) == 0 {
		t.Fatal("expected in_money json to be stored")
	}
}

func TestAggregatedTransactionRepo_List_WithCursorAndFilters(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	importID := uuid.New()
	cursorID := uuid.New()
	dateFrom := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	cursorTime := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	store := &fakeStore{
		listFn: func(_ context.Context, arg db.ListAggregatedTransactionsParams) ([]db.AggregatedTransaction, error) {
			if arg.UserID != userID {
				t.Fatalf("unexpected user id: %s", arg.UserID)
			}
			if !arg.DateFrom.Valid || !arg.DateTo.Valid {
				t.Fatalf("expected date bounds to be set: %+v", arg)
			}
			if arg.ImportID == nil || *arg.ImportID != importID {
				t.Fatalf("unexpected import filter: %+v", arg.ImportID)
			}
			if arg.Source == nil || *arg.Source != "MEXC" {
				t.Fatalf("unexpected source filter: %+v", arg.Source)
			}
			if arg.Kind == nil || *arg.Kind != "spot" {
				t.Fatalf("unexpected kind filter: %+v", arg.Kind)
			}
			if !arg.HasCursor || !arg.CursorTime.Valid || arg.CursorID != cursorID {
				t.Fatalf("unexpected cursor args: %+v", arg)
			}
			if arg.PageLimit != 31 {
				t.Fatalf("unexpected page limit: %d", arg.PageLimit)
			}

			base := time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC)
			makeRow := func(id uuid.UUID, t time.Time) db.AggregatedTransaction {
				return db.AggregatedTransaction{
					ID:            id,
					UserID:        userID,
					ImportID:      importID,
					Source:        "MEXC",
					TimeUtc:       pgtype.Timestamptz{Time: t, Valid: true},
					Kind:          "spot",
					TxFingerprint: "fp-" + id.String(),
					CreatedAt:     pgtype.Timestamptz{Time: t, Valid: true},
				}
			}
			rows := make([]db.AggregatedTransaction, 0, 31)
			for i := range 31 {
				rows = append(rows, makeRow(uuid.New(), base.Add(-time.Duration(i)*time.Minute)))
			}
			return rows, nil
		},
	}

	repo := NewAggregatedTransactionRepo(store)
	items, hasMore, err := repo.List(
		context.Background(),
		userID,
		domain.ListTransactionsFilter{
			DateFrom: &dateFrom,
			DateTo:   &dateTo,
			ImportID: &importID,
			Source:   "MEXC",
			Kind:     "spot",
		},
		30,
		&domain.AggregatedTxCursor{
			LastTimeUTC: cursorTime,
			LastID:      cursorID,
		},
	)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if !hasMore {
		t.Fatal("expected hasMore=true")
	}
	if len(items) != 30 {
		t.Fatalf("expected 30 items, got %d", len(items))
	}
}

func TestAggregatedTransactionRepo_List_InvalidJSON(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	store := &fakeStore{
		listFn: func(_ context.Context, _ db.ListAggregatedTransactionsParams) ([]db.AggregatedTransaction, error) {
			return []db.AggregatedTransaction{
				{
					ID:       uuid.New(),
					UserID:   userID,
					ImportID: uuid.New(),
					TimeUtc:  pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
					Kind:     "spot",
					InMoney:  []byte("bad-json"),
				},
			}, nil
		},
	}

	repo := NewAggregatedTransactionRepo(store)
	_, _, err := repo.List(context.Background(), userID, domain.ListTransactionsFilter{}, 30, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertRepoErrorCode(t, err, apperr.ErrInternal)
}

func assertRepoErrorCode(t *testing.T, err error, expected apperr.ErrorCode) {
	t.Helper()
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		t.Fatalf("expected *apperr.Error, got %T (%v)", err, err)
	}
	if ae.Code != expected {
		t.Fatalf("unexpected code: got=%s want=%s", ae.Code, expected)
	}
}
