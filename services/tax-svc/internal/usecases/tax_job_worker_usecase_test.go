package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
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
		INN:                "123456789012",
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

	report.EXPECT().RequestRender(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	jobRepo.EXPECT().SaveResult(gomock.Any(), jobID, gomock.Any(), gomock.Any(), gomock.Nil()).DoAndReturn(
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
			if reportObjectKey != nil {
				t.Fatalf("report_object_key must be nil for stub renderer, got %v", *reportObjectKey)
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

func mustRegistry(t *testing.T) *engines.Registry {
	t.Helper()
	registry, err := engines.NewRegistry(enginesru.New())
	if err != nil {
		t.Fatalf("create engines registry: %v", err)
	}
	return registry
}
