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
	tenantID := uuid.New()
	importID := uuid.New()

	txs := []domain.AggregatedTransaction{
		{ID: uuid.New(), TenantID: tenantID, ImportID: importID, TxFingerprint: "fp-1"},
		{ID: uuid.New(), TenantID: tenantID, ImportID: importID, TxFingerprint: "fp-1"},
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

	tenantID := uuid.New()
	importID := uuid.New()
	txID := uuid.New()
	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)

	err := repo.UpsertBatch(context.Background(), []domain.AggregatedTransaction{
		{
			ID:            txID,
			TenantID:      tenantID,
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
	if captured.ID != txID || captured.TenantID != tenantID || captured.ImportID != importID {
		t.Fatalf("unexpected captured ids: %+v", captured)
	}
	if captured.TxFingerprint != "fp-1" {
		t.Fatalf("unexpected trimmed fingerprint: %q", captured.TxFingerprint)
	}
	if len(captured.InMoney) == 0 {
		t.Fatal("expected in_money json to be stored")
	}
}

func TestAggregatedTransactionRepo_ListByImport_MapsRows(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New()
	importID := uuid.New()
	txID := uuid.New()
	createdAt := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)

	store := &fakeStore{
		countByImportFn: func(_ context.Context, arg db.CountAggregatedTransactionsByImportParams) (int64, error) {
			if arg.TenantID != tenantID || arg.ImportID != importID {
				t.Fatalf("unexpected count args: %+v", arg)
			}
			return 1, nil
		},
		listByImportFn: func(_ context.Context, arg db.ListAggregatedTransactionsByImportParams) ([]db.AggregatedTransaction, error) {
			if arg.Limit != 10 || arg.Offset != 0 {
				t.Fatalf("unexpected paging: %+v", arg)
			}
			return []db.AggregatedTransaction{
				{
					ID:            txID,
					TenantID:      tenantID,
					ImportID:      importID,
					Source:        "MEXC",
					TimeUtc:       pgtype.Timestamptz{Time: createdAt, Valid: true},
					Kind:          "Spot",
					InMoney:       []byte(`{"symbol":"BTC","crypto_amount":"0.1","fiat_amount":"100"}`),
					OutMoney:      nil,
					FeeMoney:      nil,
					TxFingerprint: "fp-1",
					CreatedAt:     pgtype.Timestamptz{Time: createdAt, Valid: true},
				},
			}, nil
		},
	}

	repo := NewAggregatedTransactionRepo(store)
	page, err := repo.ListByImport(context.Background(), tenantID, importID, 10, 0)
	if err != nil {
		t.Fatalf("ListByImport returned error: %v", err)
	}
	if page.Total != 1 || len(page.Transactions) != 1 {
		t.Fatalf("unexpected page: %+v", page)
	}
	if page.Transactions[0].InMoney == nil || page.Transactions[0].InMoney.FiatAmount == nil {
		t.Fatalf("expected mapped in_money fiat amount, got: %+v", page.Transactions[0].InMoney)
	}
	if *page.Transactions[0].InMoney.FiatAmount != "100" {
		t.Fatalf("unexpected fiat amount: %s", *page.Transactions[0].InMoney.FiatAmount)
	}
}

func TestAggregatedTransactionRepo_ListByImport_InvalidJSON(t *testing.T) {
	t.Parallel()

	tenantID := uuid.New()
	importID := uuid.New()

	store := &fakeStore{
		countByImportFn: func(_ context.Context, _ db.CountAggregatedTransactionsByImportParams) (int64, error) {
			return 1, nil
		},
		listByImportFn: func(_ context.Context, _ db.ListAggregatedTransactionsByImportParams) ([]db.AggregatedTransaction, error) {
			return []db.AggregatedTransaction{{
				ID:       uuid.New(),
				TenantID: tenantID,
				ImportID: importID,
				TimeUtc:  pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
				Kind:     "Spot",
				InMoney:  []byte("not-json"),
			}}, nil
		},
	}

	repo := NewAggregatedTransactionRepo(store)
	_, err := repo.ListByImport(context.Background(), tenantID, importID, 10, 0)
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
