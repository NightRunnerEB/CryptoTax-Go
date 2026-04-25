package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/events"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/engines"
	enginesru "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/engines/ru"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/mocks"
)

func TestTaxJobWorkerUC_ProcessNextQueuedJob_NoJobs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jobRepo := mocks.NewMockTaxJobRepo(ctrl)
	profileRepo := mocks.NewMockTaxProfileRepo(ctrl)
	txProvider := mocks.NewMockAggregatedTxProvider(ctrl)
	report := mocks.NewMockReportClient(ctrl)
	storage := mocks.NewMockObjectStorage(ctrl)

	jobRepo.EXPECT().ClaimNextQueued(gomock.Any()).Return(nil, nil).Times(1)

	registry := mustRegistry(t)
	uc := NewTaxJobWorkerUC(jobRepo, profileRepo, txProvider, report, storage, registry, 3, time.Second, 10*time.Second)

	processed, err := uc.ProcessNextQueuedJob(context.Background())
	if err != nil {
		t.Fatalf("ProcessNextQueuedJob() unexpected error: %v", err)
	}
	if processed {
		t.Fatal("expected processed=false")
	}
}

func TestTaxJobWorkerUC_ProcessNextQueuedJob_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jobRepo := mocks.NewMockTaxJobRepo(ctrl)
	profileRepo := mocks.NewMockTaxProfileRepo(ctrl)
	txProvider := mocks.NewMockAggregatedTxProvider(ctrl)
	report := mocks.NewMockReportClient(ctrl)
	storage := mocks.NewMockObjectStorage(ctrl)

	userID := uuid.New()
	jobID := uuid.New()
	job := domain.TaxJob{
		ID:       jobID,
		UserID:   userID,
		TaxYear:  2025,
		Status:   domain.JobRunning,
		Attempts: 1,
		PolicySnapshot: domain.TaxPolicy{
			Jurisdiction:    domain.JurisdictionRU,
			CostBasisMethod: domain.FIFO,
		},
	}
	profile := domain.TaxProfile{
		UserID:             userID,
		INN:                "123456789047",
		OKTMO:              "12345678",
		LastName:           "Petrov",
		FirstName:          "Ivan",
		Timezone:           "Europe/Moscow",
		TaxResidencyStatus: domain.Resident,
		TaxPayerType:       domain.INDIVIDUAL,
	}

	jobRepo.EXPECT().ClaimNextQueued(gomock.Any()).Return(&job, nil).Times(1)
	profileRepo.EXPECT().Get(gomock.Any(), userID).Return(profile, nil).Times(1)

	expectedFrom, expectedTo, err := taxYearBoundsUTC(job.TaxYear, profile.Timezone)
	if err != nil {
		t.Fatalf("taxYearBoundsUTC failed: %v", err)
	}
	txProvider.EXPECT().ListTransactionsByRange(gomock.Any(), userID, expectedFrom, expectedTo, "RUB").Return([]domain.AggregatedTransaction{}, nil).Times(1)

	storage.EXPECT().UploadJSON(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, objectKey string, payload any) error {
		if objectKey == "" {
			t.Fatal("object key must not be empty")
		}
		if payload == nil {
			t.Fatal("payload must not be nil")
		}
		return nil
	}).Times(1)

	expectedReportKey := "reports/user/report.xml"
	report.EXPECT().RequestRender(gomock.Any(), gomock.Any()).Return(expectedReportKey, nil).Times(1)

	jobRepo.EXPECT().SaveResult(gomock.Any(), jobID, gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, gotID uuid.UUID, summary domain.TaxSummary, auditObjectKey *string, reportObjectKey *string) error {
			if gotID != jobID {
				t.Fatalf("job id mismatch: got %s want %s", gotID, jobID)
			}
			if summary.UserID != userID || summary.TaxYear != job.TaxYear {
				t.Fatalf("summary mismatch: %+v", summary)
			}
			if auditObjectKey == nil || *auditObjectKey == "" {
				t.Fatal("audit object key must be saved")
			}
			if reportObjectKey == nil || *reportObjectKey != expectedReportKey {
				t.Fatalf("report_object_key mismatch: got=%v want=%s", reportObjectKey, expectedReportKey)
			}
			return nil
		},
	).Times(1)

	registry := mustRegistry(t)
	uc := NewTaxJobWorkerUC(jobRepo, profileRepo, txProvider, report, storage, registry, 3, time.Second, 10*time.Second)

	processed, err := uc.ProcessNextQueuedJob(context.Background())
	if err != nil {
		t.Fatalf("ProcessNextQueuedJob() unexpected error: %v", err)
	}
	if !processed {
		t.Fatal("expected processed=true")
	}
}

func TestTaxJobWorkerUC_ProcessNextQueuedJob_RetryableErrorRequeue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jobRepo := mocks.NewMockTaxJobRepo(ctrl)
	profileRepo := mocks.NewMockTaxProfileRepo(ctrl)
	txProvider := mocks.NewMockAggregatedTxProvider(ctrl)
	report := mocks.NewMockReportClient(ctrl)
	storage := mocks.NewMockObjectStorage(ctrl)

	userID := uuid.New()
	jobID := uuid.New()
	job := domain.TaxJob{
		ID:       jobID,
		UserID:   userID,
		TaxYear:  2025,
		Status:   domain.JobRunning,
		Attempts: 1,
		PolicySnapshot: domain.TaxPolicy{
			Jurisdiction:    domain.JurisdictionRU,
			CostBasisMethod: domain.FIFO,
		},
	}
	profile := domain.TaxProfile{
		UserID:             userID,
		Timezone:           "Europe/Moscow",
		TaxResidencyStatus: domain.Resident,
		TaxPayerType:       domain.INDIVIDUAL,
	}

	jobRepo.EXPECT().ClaimNextQueued(gomock.Any()).Return(&job, nil).Times(1)
	profileRepo.EXPECT().Get(gomock.Any(), userID).Return(profile, nil).Times(1)
	txProvider.EXPECT().ListTransactionsByRange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "RUB").
		Return(nil, apperr.AggregationUnavailable("aggregation unavailable", status.Error(codes.Unavailable, "down"), nil)).Times(1)

	jobRepo.EXPECT().Requeue(gomock.Any(), jobID, gomock.Any(), string(apperr.ErrAggregationUnavailable), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, retryAt time.Time, _ string, _ string) error {
			if retryAt.Before(time.Now()) {
				t.Fatalf("retry_at should be in future, got %s", retryAt)
			}
			return nil
		}).Times(1)

	registry := mustRegistry(t)
	uc := NewTaxJobWorkerUC(jobRepo, profileRepo, txProvider, report, storage, registry, 3, time.Second, 10*time.Second)

	processed, err := uc.ProcessNextQueuedJob(context.Background())
	if err != nil {
		t.Fatalf("ProcessNextQueuedJob() unexpected error: %v", err)
	}
	if !processed {
		t.Fatal("expected processed=true")
	}
}

func TestTaxJobWorkerUC_ProcessNextQueuedJob_NonRetryableMarksFailed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	jobRepo := mocks.NewMockTaxJobRepo(ctrl)
	profileRepo := mocks.NewMockTaxProfileRepo(ctrl)
	txProvider := mocks.NewMockAggregatedTxProvider(ctrl)
	report := mocks.NewMockReportClient(ctrl)
	storage := mocks.NewMockObjectStorage(ctrl)

	userID := uuid.New()
	jobID := uuid.New()
	job := domain.TaxJob{
		ID:       jobID,
		UserID:   userID,
		TaxYear:  2025,
		Status:   domain.JobRunning,
		Attempts: 1,
		PolicySnapshot: domain.TaxPolicy{
			Jurisdiction:    domain.JurisdictionRU,
			CostBasisMethod: domain.FIFO,
		},
	}
	profile := domain.TaxProfile{
		UserID:             userID,
		Timezone:           "Europe/Moscow",
		TaxResidencyStatus: domain.Resident,
		TaxPayerType:       domain.INDIVIDUAL,
	}

	jobRepo.EXPECT().ClaimNextQueued(gomock.Any()).Return(&job, nil).Times(1)
	profileRepo.EXPECT().Get(gomock.Any(), userID).Return(profile, nil).Times(1)
	txProvider.EXPECT().ListTransactionsByRange(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "RUB").
		Return(nil, apperr.NeedsPriceResolution("data is not ready", errors.New("not ready"), nil)).Times(1)

	jobRepo.EXPECT().MarkFailed(gomock.Any(), jobID, string(apperr.ErrNeedsPriceResolution), gomock.Any()).Return(nil).Times(1)

	registry := mustRegistry(t)
	uc := NewTaxJobWorkerUC(jobRepo, profileRepo, txProvider, report, storage, registry, 3, time.Second, 10*time.Second)

	processed, err := uc.ProcessNextQueuedJob(context.Background())
	if err != nil {
		t.Fatalf("ProcessNextQueuedJob() unexpected error: %v", err)
	}
	if !processed {
		t.Fatal("expected processed=true")
	}
}

func TestSummarizeResult_SplitsTradeAndP2PIncome(t *testing.T) {
	userID := uuid.New()
	now := time.Date(2026, time.January, 10, 12, 0, 0, 0, time.UTC)

	job := domain.TaxJob{
		UserID:  userID,
		TaxYear: 2026,
		PolicySnapshot: domain.TaxPolicy{
			Jurisdiction: domain.JurisdictionRU,
		},
	}
	profile := domain.TaxProfile{
		UserID:             userID,
		TaxResidencyStatus: domain.Resident,
	}

	result := engines.BuildResult{
		RealizationEvents: []events.RealizationEvent{
			{
				Asset:         "BTC",
				Qty:           decimal.RequireFromString("0.2"),
				ProceedsFiat:  decimal.RequireFromString("100"),
				CostBasisFiat: decimal.RequireFromString("60"),
				Kind:          events.RealizationSellFiat,
			},
			{
				OccurredAt:    now,
				Asset:         "USDT",
				Qty:           decimal.RequireFromString("500"),
				ProceedsFiat:  decimal.RequireFromString("50"),
				CostBasisFiat: decimal.RequireFromString("40"),
				Kind:          events.RealizationP2PSell,
			},
		},
		IncomeEvents: []events.IncomeEvent{
			{
				AmountFiat: decimal.RequireFromString("20"),
			},
		},
		ExpenseEvents: []events.ExpenseEvent{
			{
				AmountFiat: decimal.RequireFromString("5"),
			},
		},
	}

	summary := summarizeResult(job, profile, result)

	if !summary.TotalIncome.Equal(decimal.RequireFromString("170")) {
		t.Fatalf("total income mismatch: got %s want 170", summary.TotalIncome)
	}
	if !summary.TotalTrade.Equal(decimal.RequireFromString("100")) {
		t.Fatalf("total trade mismatch: got %s want 100", summary.TotalTrade)
	}
	if !summary.TotalExpense.Equal(decimal.RequireFromString("105")) {
		t.Fatalf("total expense mismatch: got %s want 105", summary.TotalExpense)
	}
	if !summary.TaxBase.Equal(decimal.RequireFromString("65")) {
		t.Fatalf("tax base mismatch: got %s want 65", summary.TaxBase)
	}
	if !summary.TaxDue.Equal(calculateTaxDue(job.PolicySnapshot.Jurisdiction, profile, summary.TaxBase)) {
		t.Fatalf("tax due mismatch: got %s", summary.TaxDue)
	}

	if len(summary.TotalP2P) != 1 {
		t.Fatalf("total p2p lines mismatch: got %d want 1", len(summary.TotalP2P))
	}
	line := summary.TotalP2P[0]
	if !line.OccurredAt.Equal(now) {
		t.Fatalf("occurred_at mismatch: got %s want %s", line.OccurredAt, now)
	}
	if !line.Qty.Equal(decimal.RequireFromString("500")) {
		t.Fatalf("qty mismatch: got %s want 500", line.Qty)
	}
	if !line.GainFiat.Equal(decimal.RequireFromString("10")) {
		t.Fatalf("gain mismatch: got %s want 10", line.GainFiat)
	}
}

func mustRegistry(t *testing.T) *engines.Registry {
	t.Helper()
	registry, err := engines.NewRegistry(enginesru.New())
	if err != nil {
		t.Fatalf("create engines registry: %v", err)
	}
	return registry
}
