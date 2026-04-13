package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	pricev1 "github.com/NightRunner/CryptoTax-Go/gen/price/v1"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/mocks"
)

func TestProcessImport_HappyPath(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	txRepo := mocks.NewMockAggregatedTransactionRepo(ctrl)
	importRepo := mocks.NewMockImportStateRepo(ctrl)
	settingsRepo := mocks.NewMockUserSettingsRepo(ctrl)
	ledgerClient := mocks.NewMockLedgerClient(ctrl)
	priceClient := mocks.NewMockPriceClient(ctrl)
	lockManager := mocks.NewMockLockManager(ctrl)

	uc := NewAggregationUC(
		txRepo,
		importRepo,
		settingsRepo,
		ledgerClient,
		priceClient,
		lockManager,
		100,
		10*time.Minute,
	)

	userID := uuid.New()
	importID := uuid.New()
	eventID := uuid.New()
	txID := uuid.New()
	now := time.Date(2026, 3, 1, 10, 30, 0, 0, time.UTC)

	ledgerTxs := []domain.LedgerTransaction{
		{
			ID:            txID,
			UserID:        userID,
			Source:        "",
			ImportID:      importID,
			TimeUTC:       now,
			Kind:          "Spot",
			InMoney:       &domain.LedgerAsset{Symbol: "BTC", Amount: "0.1"},
			OutMoney:      &domain.LedgerAsset{Symbol: "USDT", Amount: "5000"},
			FeeMoney:      &domain.LedgerAsset{Symbol: "USDT", Amount: "5"},
			TxFingerprint: "fp-1",
			CreatedAt:     now,
		},
	}

	valuated := &pricev1.ValuateTransactionsResponse{
		Transactions: []*pricev1.ValuatedTx{
			{
				TxId: txID.String(),
				InFiat: &pricev1.FiatLeg{
					Fiat: "5000",
				},
				OutFiat: &pricev1.FiatLeg{
					Fiat: "5000",
				},
				FeeFiat: &pricev1.FiatLeg{
					Fiat: "5",
				},
			},
		},
	}

	settingsRepo.EXPECT().
		Get(gomock.Any(), userID).
		Return(domain.UserSettings{UserID: userID, FiatCurrency: "rub", Timezone: "Europe/Moscow"}, nil)

	importRepo.EXPECT().
		Get(gomock.Any(), userID, importID).
		Return(domain.AggregationImportState{}, apperr.NotFound("not found", apperr.Resource{Type: "aggregation_import_state", Name: userID.String() + ":" + importID.String()}, nil))

	lockManager.EXPECT().
		AcquireImportLock(gomock.Any(), userID, importID, 10*time.Minute).
		Return(true, nil)

	lockManager.EXPECT().
		ReleaseImportLock(gomock.Any(), userID, importID).
		Return(nil)

	importRepo.EXPECT().
		UpsertProcessing(gomock.Any(), gomock.AssignableToTypeOf(domain.AggregationImportState{})).
		DoAndReturn(func(_ context.Context, state domain.AggregationImportState) error {
			if state.UserID != userID || state.ImportID != importID || state.EventId != eventID {
				t.Fatalf("unexpected processing state: %+v", state)
			}
			if state.Status != domain.ImportStatusProcessing {
				t.Fatalf("unexpected state status: %s", state.Status)
			}
			return nil
		})

	ledgerClient.EXPECT().
		ListTransactionsByImport(gomock.Any(), userID, importID).
		Return(ledgerTxs, nil)

	priceClient.EXPECT().
		ValuateTransactionsBatch(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *pricev1.ValuateTransactionsRequest) (*pricev1.ValuateTransactionsResponse, error) {
			if req.GetUserId() != userID.String() {
				t.Fatalf("unexpected user_id: %s", req.GetUserId())
			}
			if req.GetSource() != defaultImportSource {
				t.Fatalf("unexpected source: %s", req.GetSource())
			}
			if req.GetFiatCurrency() != "RUB" {
				t.Fatalf("unexpected fiat: %s", req.GetFiatCurrency())
			}
			if len(req.GetTransactions()) != 1 {
				t.Fatalf("unexpected tx count: %d", len(req.GetTransactions()))
			}
			if got := req.GetTransactions()[0].GetTimeUtc(); got == nil || !got.AsTime().Equal(now) {
				t.Fatalf("unexpected tx timestamp: %v", got)
			}
			return valuated, nil
		})

	txRepo.EXPECT().
		UpsertBatch(gomock.Any(), gomock.AssignableToTypeOf([]domain.AggregatedTransaction{})).
		DoAndReturn(func(_ context.Context, got []domain.AggregatedTransaction) error {
			if len(got) != 1 {
				t.Fatalf("unexpected tx count: %d", len(got))
			}
			item := got[0]
			if item.ID != txID || item.UserID != userID || item.ImportID != importID {
				t.Fatalf("unexpected ids in tx: %+v", item)
			}
			if item.Source != defaultImportSource {
				t.Fatalf("unexpected source: %s", item.Source)
			}
			if item.TxFingerprint != "fp-1" {
				t.Fatalf("unexpected fingerprint: %s", item.TxFingerprint)
			}
			if item.InMoney == nil || item.InMoney.FiatAmount == nil || *item.InMoney.FiatAmount != "5000" {
				t.Fatalf("unexpected in_money valuation: %+v", item.InMoney)
			}
			if item.OutMoney == nil || item.OutMoney.FiatAmount == nil || *item.OutMoney.FiatAmount != "5000" {
				t.Fatalf("unexpected out_money valuation: %+v", item.OutMoney)
			}
			if item.FeeMoney == nil || item.FeeMoney.FiatAmount == nil || *item.FeeMoney.FiatAmount != "5" {
				t.Fatalf("unexpected fee_money valuation: %+v", item.FeeMoney)
			}
			return nil
		})

	importRepo.EXPECT().
		MarkCompleted(gomock.Any(), userID, importID).
		Return(nil)

	err := uc.ProcessImport(context.Background(), domain.ImportEvent{
		EventId:  eventID,
		UserID:   userID,
		ImportID: importID,
	})
	if err != nil {
		t.Fatalf("ProcessImport returned error: %v", err)
	}
}

func TestProcessImport_LockNotAcquired_Skip(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	txRepo := mocks.NewMockAggregatedTransactionRepo(ctrl)
	importRepo := mocks.NewMockImportStateRepo(ctrl)
	settingsRepo := mocks.NewMockUserSettingsRepo(ctrl)
	ledgerClient := mocks.NewMockLedgerClient(ctrl)
	priceClient := mocks.NewMockPriceClient(ctrl)
	lockManager := mocks.NewMockLockManager(ctrl)

	uc := NewAggregationUC(txRepo, importRepo, settingsRepo, ledgerClient, priceClient, lockManager, 100, 5*time.Minute)

	userID := uuid.New()
	importID := uuid.New()

	lockManager.EXPECT().
		AcquireImportLock(gomock.Any(), userID, importID, 5*time.Minute).
		Return(false, nil)

	err := uc.ProcessImport(context.Background(), domain.ImportEvent{
		EventId:  uuid.New(),
		UserID:   userID,
		ImportID: importID,
	})
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

func TestProcessImport_LedgerUnavailable_MarkFailed(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	txRepo := mocks.NewMockAggregatedTransactionRepo(ctrl)
	importRepo := mocks.NewMockImportStateRepo(ctrl)
	settingsRepo := mocks.NewMockUserSettingsRepo(ctrl)
	ledgerClient := mocks.NewMockLedgerClient(ctrl)
	priceClient := mocks.NewMockPriceClient(ctrl)
	lockManager := mocks.NewMockLockManager(ctrl)

	uc := NewAggregationUC(txRepo, importRepo, settingsRepo, ledgerClient, priceClient, lockManager, 100, time.Minute)

	userID := uuid.New()
	importID := uuid.New()
	ledgerErr := apperr.LedgerUnavailable("ledger down", errors.New("dial tcp: refused"), map[string]string{"status_code": "503"})

	lockManager.EXPECT().
		AcquireImportLock(gomock.Any(), userID, importID, time.Minute).
		Return(true, nil)
	lockManager.EXPECT().
		ReleaseImportLock(gomock.Any(), userID, importID).
		Return(nil)

	settingsRepo.EXPECT().
		Get(gomock.Any(), userID).
		Return(domain.UserSettings{UserID: userID, FiatCurrency: "USD", Timezone: "UTC"}, nil)

	importRepo.EXPECT().
		Get(gomock.Any(), userID, importID).
		Return(domain.AggregationImportState{}, apperr.NotFound("not found", apperr.Resource{Type: "aggregation_import_state", Name: "x"}, nil))

	importRepo.EXPECT().
		UpsertProcessing(gomock.Any(), gomock.Any()).
		Return(nil)

	ledgerClient.EXPECT().
		ListTransactionsByImport(gomock.Any(), userID, importID).
		Return(nil, ledgerErr)

	importRepo.EXPECT().
		MarkFailed(gomock.Any(), userID, importID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ uuid.UUID, msg string) error {
			if msg == "" {
				t.Fatal("expected non-empty error message")
			}
			return nil
		})

	err := uc.ProcessImport(context.Background(), domain.ImportEvent{
		EventId:  uuid.New(),
		UserID:   userID,
		ImportID: importID,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertAppErrorCode(t, err, apperr.ErrLedgerUnavailable)
}

func TestListTransactionsByRange_InvalidTargetFiat(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	txRepo := mocks.NewMockAggregatedTransactionRepo(ctrl)
	uc := NewAggregationUC(txRepo, nil, nil, nil, nil, nil, 100, 0)

	_, err := uc.ListTransactionsByRange(
		context.Background(),
		uuid.New(),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		100,
		0,
		"abc",
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertAppErrorCode(t, err, apperr.ErrInvalidArgument)
}

func TestListTransactionsByRange_RevaluationIncomplete_ReturnsDataNotReady(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	txRepo := mocks.NewMockAggregatedTransactionRepo(ctrl)
	priceClient := mocks.NewMockPriceClient(ctrl)
	uc := NewAggregationUC(txRepo, nil, nil, nil, priceClient, nil, 100, 0)

	userID := uuid.New()
	txID := uuid.New()

	page := domain.AggregatedTxPage{
		Transactions: []domain.AggregatedTransaction{
			{
				ID:            txID,
				UserID:        userID,
				Source:        "",
				ImportID:      uuid.New(),
				TimeUTC:       time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
				Kind:          "Spot",
				InMoney:       &domain.MoneyLeg{Symbol: "BTC", CryptoAmount: "0.1", FiatAmount: strPtr("100")},
				TxFingerprint: "fp-1",
			},
		},
		Total: 1,
	}

	txRepo.EXPECT().
		ListByRange(gomock.Any(), userID, gomock.Any(), gomock.Any(), int32(100), int32(0)).
		Return(page, nil)

	priceClient.EXPECT().
		ValuateTransactionsBatch(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, req *pricev1.ValuateTransactionsRequest) (*pricev1.ValuateTransactionsResponse, error) {
			if req.GetSource() != defaultImportSource {
				t.Fatalf("unexpected source: %s", req.GetSource())
			}
			if req.GetFiatCurrency() != "USD" {
				t.Fatalf("unexpected fiat currency: %s", req.GetFiatCurrency())
			}
			if len(req.GetTransactions()) != 1 || req.GetTransactions()[0].GetTxId() != txID.String() {
				t.Fatalf("unexpected tx list: %+v", req.GetTransactions())
			}
			return &pricev1.ValuateTransactionsResponse{Transactions: nil}, nil
		})

	_, err := uc.ListTransactionsByRange(
		context.Background(),
		userID,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		100,
		0,
		"USD",
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertAppErrorCode(t, err, apperr.ErrDataNotReady)
}

func assertAppErrorCode(t *testing.T, err error, expected apperr.ErrorCode) {
	t.Helper()
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		t.Fatalf("expected *apperr.Error, got %T (%v)", err, err)
	}
	if ae.Code != expected {
		t.Fatalf("unexpected code: got %s want %s", ae.Code, expected)
	}
}

func strPtr(v string) *string {
	return &v
}
