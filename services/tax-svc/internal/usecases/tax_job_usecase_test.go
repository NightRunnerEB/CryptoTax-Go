package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/mocks"
)

func TestTaxJobUC_Enqueue_InvalidTenantID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	uc := NewTaxJobUC(
		mocks.NewMockTaxJobRepo(ctrl),
		mocks.NewMockTaxProfileRepo(ctrl),
		mocks.NewMockObjectStorage(ctrl),
	)

	_, err := uc.Enqueue(context.Background(), uuid.Nil, 2026, domain.TaxPolicy{Jurisdiction: domain.JurisdictionRU, CostBasisMethod: domain.FIFO})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	assertAppErrCode(t, err, apperr.ErrInvalidArgument)
}

func TestTaxJobUC_Enqueue_ProfileMissing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := uuid.New()
	jobRepo := mocks.NewMockTaxJobRepo(ctrl)
	profileRepo := mocks.NewMockTaxProfileRepo(ctrl)
	storage := mocks.NewMockObjectStorage(ctrl)

	profileRepo.EXPECT().Get(gomock.Any(), tenantID).Return(domain.TaxProfile{}, apperr.NotFound("tax profile not found", apperr.Resource{Type: "tax_profile", Name: tenantID.String()}, nil)).Times(1)

	uc := NewTaxJobUC(jobRepo, profileRepo, storage)
	_, err := uc.Enqueue(context.Background(), tenantID, 2026, domain.TaxPolicy{Jurisdiction: domain.JurisdictionRU, CostBasisMethod: domain.FIFO})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertAppErrCode(t, err, apperr.ErrNotFound)
}

func TestTaxJobUC_Enqueue_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := uuid.New()
	jobRepo := mocks.NewMockTaxJobRepo(ctrl)
	profileRepo := mocks.NewMockTaxProfileRepo(ctrl)
	storage := mocks.NewMockObjectStorage(ctrl)

	profileRepo.EXPECT().Get(gomock.Any(), tenantID).Return(domain.TaxProfile{TenantID: tenantID}, nil).Times(1)

	jobRepo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got domain.TaxJob) (domain.TaxJob, error) {
		if got.TenantID != tenantID {
			t.Fatalf("tenant mismatch: got %s want %s", got.TenantID, tenantID)
		}
		if got.Status != domain.JobQueued {
			t.Fatalf("status mismatch: got %s want %s", got.Status, domain.JobQueued)
		}
		if got.PolicySnapshot.Jurisdiction != domain.JurisdictionRU {
			t.Fatalf("jurisdiction mismatch: got %s want %s", got.PolicySnapshot.Jurisdiction, domain.JurisdictionRU)
		}
		if got.PolicySnapshot.CostBasisMethod != domain.FIFO {
			t.Fatalf("cost_basis mismatch: got %s want %s", got.PolicySnapshot.CostBasisMethod, domain.FIFO)
		}
		if got.ID == uuid.Nil {
			t.Fatal("job id must be generated")
		}
		return got, nil
	}).Times(1)

	uc := NewTaxJobUC(jobRepo, profileRepo, storage)
	job, err := uc.Enqueue(context.Background(), tenantID, 2026, domain.TaxPolicy{
		Jurisdiction:    "ru",
		CostBasisMethod: "",
	})
	if err != nil {
		t.Fatalf("Enqueue() unexpected error: %v", err)
	}
	if job.ID == uuid.Nil {
		t.Fatal("expected non-empty job id")
	}
}

func TestTaxJobUC_GetStatus_AttachesPresignedURLs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := uuid.New()
	jobID := uuid.New()
	auditKey := "audits/tenant/job.json"
	reportKey := "reports/tenant/job.xml"
	auditURL := "https://minio.local/audit"
	reportURL := "https://minio.local/report"

	jobRepo := mocks.NewMockTaxJobRepo(ctrl)
	profileRepo := mocks.NewMockTaxProfileRepo(ctrl)
	storage := mocks.NewMockObjectStorage(ctrl)

	jobRepo.EXPECT().Get(gomock.Any(), tenantID, jobID).Return(domain.TaxJob{
		ID:              jobID,
		TenantID:        tenantID,
		Status:          domain.JobSuccess,
		PolicySnapshot:  domain.TaxPolicy{Jurisdiction: domain.JurisdictionRU, CostBasisMethod: domain.FIFO},
		AuditObjectKey:  &auditKey,
		ReportObjectKey: &reportKey,
	}, nil).Times(1)

	storage.EXPECT().PresignGet(gomock.Any(), auditKey).Return(auditURL, nil).Times(1)
	storage.EXPECT().PresignGet(gomock.Any(), reportKey).Return(reportURL, nil).Times(1)

	uc := NewTaxJobUC(jobRepo, profileRepo, storage)
	job, err := uc.GetStatus(context.Background(), tenantID, jobID)
	if err != nil {
		t.Fatalf("GetStatus() unexpected error: %v", err)
	}
	if job.AuditZipURL == nil || *job.AuditZipURL != auditURL {
		t.Fatalf("unexpected audit url: %+v", job.AuditZipURL)
	}
	if job.ReportURL == nil || *job.ReportURL != reportURL {
		t.Fatalf("unexpected report url: %+v", job.ReportURL)
	}
}

func TestTaxJobUC_List_AttachesPresignedURLs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := uuid.New()
	jobID := uuid.New()
	auditKey := "audits/tenant/job.json"
	auditURL := "https://minio.local/audit"

	jobRepo := mocks.NewMockTaxJobRepo(ctrl)
	profileRepo := mocks.NewMockTaxProfileRepo(ctrl)
	storage := mocks.NewMockObjectStorage(ctrl)

	jobRepo.EXPECT().List(gomock.Any(), tenantID, int32(10), int32(0)).Return([]domain.TaxJob{
		{
			ID:             jobID,
			TenantID:       tenantID,
			Status:         domain.JobSuccess,
			PolicySnapshot: domain.TaxPolicy{Jurisdiction: domain.JurisdictionRU, CostBasisMethod: domain.FIFO},
			AuditObjectKey: &auditKey,
		},
	}, int64(1), nil).Times(1)

	storage.EXPECT().PresignGet(gomock.Any(), auditKey).Return(auditURL, nil).Times(1)

	uc := NewTaxJobUC(jobRepo, profileRepo, storage)
	jobs, total, err := uc.List(context.Background(), tenantID, 10, 0)
	if err != nil {
		t.Fatalf("List() unexpected error: %v", err)
	}
	if total != 1 {
		t.Fatalf("total mismatch: got %d want 1", total)
	}
	if len(jobs) != 1 {
		t.Fatalf("jobs length mismatch: got %d want 1", len(jobs))
	}
	if jobs[0].AuditZipURL == nil || *jobs[0].AuditZipURL != auditURL {
		t.Fatalf("unexpected audit url: %+v", jobs[0].AuditZipURL)
	}
}

func TestTaxJobUC_GetStatus_InvalidIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	uc := NewTaxJobUC(
		mocks.NewMockTaxJobRepo(ctrl),
		mocks.NewMockTaxProfileRepo(ctrl),
		mocks.NewMockObjectStorage(ctrl),
	)

	_, err := uc.GetStatus(context.Background(), uuid.Nil, uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertAppErrCode(t, err, apperr.ErrInvalidArgument)
}

func TestTaxJobUC_Enqueue_InvalidTaxPolicy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := uuid.New()
	jobRepo := mocks.NewMockTaxJobRepo(ctrl)
	profileRepo := mocks.NewMockTaxProfileRepo(ctrl)
	storage := mocks.NewMockObjectStorage(ctrl)

	profileRepo.EXPECT().Get(gomock.Any(), tenantID).Return(domain.TaxProfile{TenantID: tenantID}, nil).Times(1)

	uc := NewTaxJobUC(jobRepo, profileRepo, storage)
	_, err := uc.Enqueue(context.Background(), tenantID, time.Now().Year(), domain.TaxPolicy{Jurisdiction: "XX", CostBasisMethod: "BAD"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertAppErrCode(t, err, apperr.ErrInvalidArgument)
}

func assertAppErrCode(t *testing.T, err error, want apperr.ErrorCode) {
	t.Helper()
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		t.Fatalf("expected app error, got %T: %v", err, err)
	}
	if ae.Code != want {
		t.Fatalf("error code mismatch: got %s want %s", ae.Code, want)
	}
}
