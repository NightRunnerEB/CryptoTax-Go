package grpcserver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/metadata"

	taxv1 "github.com/NightRunner/CryptoTax-Go/gen/tax/v1"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/mocks"
)

func TestTaxServer_UpsertTaxProfile_TenantMismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	srv := NewTaxServer(
		mocks.NewMockTaxProfileUseCase(ctrl),
		mocks.NewMockTaxJobUseCase(ctrl),
	)

	req := &taxv1.UpsertTaxProfileRequest{
		TenantId: uuid.New().String(),
		Profile: &taxv1.TaxProfileInput{
			Inn:                "123456789012",
			LastName:           "Petrov",
			FirstName:          "Ivan",
			Timezone:           "Europe/Moscow",
			TaxResidencyStatus: "RESIDENT",
			TaxpayerType:       "INDIVIDUAL",
		},
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(headerTenantID, uuid.New().String()))

	_, err := srv.UpsertTaxProfile(ctx, req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServerAppErrCode(t, err, apperr.ErrInvalidArgument)
}

func TestTaxServer_UpsertTaxProfile_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := uuid.New()
	profileUC := mocks.NewMockTaxProfileUseCase(ctrl)
	jobUC := mocks.NewMockTaxJobUseCase(ctrl)
	srv := NewTaxServer(profileUC, jobUC)

	profileUC.EXPECT().Upsert(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got domain.TaxProfile) error {
		if got.TenantID != tenantID {
			t.Fatalf("tenant mismatch: got %s want %s", got.TenantID, tenantID)
		}
		if got.Timezone != "Europe/Moscow" {
			t.Fatalf("timezone mismatch: %q", got.Timezone)
		}
		if got.TaxResidencyStatus != domain.Resident {
			t.Fatalf("tax residency mismatch: %s", got.TaxResidencyStatus)
		}
		if got.TaxPayerType != domain.INDIVIDUAL {
			t.Fatalf("taxpayer type mismatch: %s", got.TaxPayerType)
		}
		return nil
	}).Times(1)

	profileUC.EXPECT().Get(gomock.Any(), tenantID).Return(domain.TaxProfile{
		TenantID:           tenantID,
		INN:                "123456789012",
		LastName:           "Petrov",
		FirstName:          "Ivan",
		Timezone:           "Europe/Moscow",
		TaxResidencyStatus: domain.Resident,
		TaxPayerType:       domain.INDIVIDUAL,
	}, nil).Times(1)

	resp, err := srv.UpsertTaxProfile(context.Background(), &taxv1.UpsertTaxProfileRequest{
		TenantId: tenantID.String(),
		Profile: &taxv1.TaxProfileInput{
			Inn:                "123456789012",
			LastName:           "Petrov",
			FirstName:          "Ivan",
			Timezone:           "Europe/Moscow",
			TaxResidencyStatus: "RESIDENT",
			TaxpayerType:       "INDIVIDUAL",
		},
	})
	if err != nil {
		t.Fatalf("UpsertTaxProfile() unexpected error: %v", err)
	}
	if resp.GetProfile() == nil || resp.GetProfile().GetTenantId() != tenantID.String() {
		t.Fatalf("unexpected profile response: %+v", resp.GetProfile())
	}
}

func TestTaxServer_StartReport_InvalidPolicy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	srv := NewTaxServer(
		mocks.NewMockTaxProfileUseCase(ctrl),
		mocks.NewMockTaxJobUseCase(ctrl),
	)

	_, err := srv.StartReport(context.Background(), &taxv1.StartReportRequest{
		TenantId: uuid.New().String(),
		Params: &taxv1.StartReportParams{
			TaxYear: 2025,
			TaxPolicy: &taxv1.TaxPolicy{
				Jurisdiction:    "XX",
				CostBasisMethod: "BAD",
			},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServerAppErrCode(t, err, apperr.ErrInvalidArgument)
}

func TestTaxServer_StartReport_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := uuid.New()
	jobID := uuid.New()
	jobUC := mocks.NewMockTaxJobUseCase(ctrl)
	srv := NewTaxServer(mocks.NewMockTaxProfileUseCase(ctrl), jobUC)

	jobUC.EXPECT().Enqueue(gomock.Any(), tenantID, 2025, domain.TaxPolicy{
		TreatCryptoCryptoAsDisposal: true,
		CostBasisMethod:             domain.FIFO,
		Jurisdiction:                domain.JurisdictionRU,
	}).Return(domain.TaxJob{
		ID:     jobID,
		Status: domain.JobQueued,
	}, nil).Times(1)

	resp, err := srv.StartReport(context.Background(), &taxv1.StartReportRequest{
		TenantId: tenantID.String(),
		Params: &taxv1.StartReportParams{
			TaxYear: 2025,
			TaxPolicy: &taxv1.TaxPolicy{
				TreatCryptoCryptoAsDisposal: true,
				CostBasisMethod:             "fifo",
				Jurisdiction:                "ru",
			},
		},
	})
	if err != nil {
		t.Fatalf("StartReport() unexpected error: %v", err)
	}
	if resp.GetReportId() != jobID.String() {
		t.Fatalf("report id mismatch: got %s want %s", resp.GetReportId(), jobID.String())
	}
	if resp.GetStatus() != string(domain.JobQueued) {
		t.Fatalf("status mismatch: got %s want %s", resp.GetStatus(), domain.JobQueued)
	}
}

func TestTaxServer_GetReportStatus_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := uuid.New()
	jobID := uuid.New()
	auditURL := "https://minio.local/audit"
	startedAt := time.Now().UTC().Add(-time.Minute)
	finishedAt := time.Now().UTC()

	jobUC := mocks.NewMockTaxJobUseCase(ctrl)
	srv := NewTaxServer(mocks.NewMockTaxProfileUseCase(ctrl), jobUC)

	jobUC.EXPECT().GetStatus(gomock.Any(), tenantID, jobID).Return(domain.TaxJob{
		ID:             jobID,
		TenantID:       tenantID,
		TaxYear:        2025,
		Status:         domain.JobSuccess,
		PolicySnapshot: domain.TaxPolicy{Jurisdiction: domain.JurisdictionRU, CostBasisMethod: domain.FIFO},
		CreatedAt:      time.Now().UTC().Add(-2 * time.Minute),
		StartedAt:      &startedAt,
		FinishedAt:     &finishedAt,
		AuditZipURL:    &auditURL,
	}, nil).Times(1)

	resp, err := srv.GetReportStatus(context.Background(), &taxv1.GetReportStatusRequest{
		TenantId: tenantID.String(),
		ReportId: jobID.String(),
	})
	if err != nil {
		t.Fatalf("GetReportStatus() unexpected error: %v", err)
	}
	if resp.GetJob() == nil || resp.GetJob().GetReportId() != jobID.String() {
		t.Fatalf("unexpected job response: %+v", resp.GetJob())
	}
	if resp.GetJob().GetAuditZipUrl() != auditURL {
		t.Fatalf("audit url mismatch: got %s want %s", resp.GetJob().GetAuditZipUrl(), auditURL)
	}
}

func TestTaxServer_ListReports_InvalidTenantID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	srv := NewTaxServer(
		mocks.NewMockTaxProfileUseCase(ctrl),
		mocks.NewMockTaxJobUseCase(ctrl),
	)

	_, err := srv.ListReports(context.Background(), &taxv1.ListReportsRequest{TenantId: "bad-uuid"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertServerAppErrCode(t, err, apperr.ErrInvalidArgument)
}

func assertServerAppErrCode(t *testing.T, err error, want apperr.ErrorCode) {
	t.Helper()
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		t.Fatalf("expected app error, got %T: %v", err, err)
	}
	if ae.Code != want {
		t.Fatalf("error code mismatch: got %s want %s", ae.Code, want)
	}
}
