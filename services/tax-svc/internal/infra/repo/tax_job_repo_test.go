package repository

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/mock/gomock"

	db "github.com/NightRunner/CryptoTax-Go/services/tax-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/mocks"
)

func TestTaxJobRepo_Create_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockStore(ctrl)
	repo := NewTaxJobRepo(store)

	jobID := uuid.New()
	userID := uuid.New()
	job := domain.TaxJob{
		ID:       jobID,
		UserID:   userID,
		TaxYear:  2025,
		Status:   domain.JobQueued,
		Attempts: 0,
		PolicySnapshot: domain.TaxPolicy{
			Jurisdiction:    domain.JurisdictionRU,
			CostBasisMethod: domain.FIFO,
		},
	}

	store.EXPECT().CreateTaxJob(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, arg db.CreateTaxJobParams) (db.TaxJob, error) {
		if arg.ID != jobID || arg.UserID != userID {
			t.Fatalf("unexpected create args: %+v", arg)
		}
		var policy domain.TaxPolicy
		if err := json.Unmarshal(arg.PolicySnapshot, &policy); err != nil {
			t.Fatalf("policy json is invalid: %v", err)
		}
		if policy.Jurisdiction != domain.JurisdictionRU || policy.CostBasisMethod != domain.FIFO {
			t.Fatalf("unexpected policy in sql args: %+v", policy)
		}
		return sampleTaxJobRow(jobID, userID), nil
	}).Times(1)

	created, err := repo.Create(context.Background(), job)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	if created.ID != jobID || created.UserID != userID {
		t.Fatalf("unexpected created job: %+v", created)
	}
	if created.PolicySnapshot.Jurisdiction != domain.JurisdictionRU {
		t.Fatalf("unexpected policy snapshot: %+v", created.PolicySnapshot)
	}
}

func TestTaxJobRepo_Get_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockStore(ctrl)
	repo := NewTaxJobRepo(store)
	jobID := uuid.New()
	userID := uuid.New()

	store.EXPECT().GetTaxJob(gomock.Any(), db.GetTaxJobParams{UserID: userID, ID: jobID}).Return(db.TaxJob{}, pgx.ErrNoRows).Times(1)

	_, err := repo.Get(context.Background(), userID, jobID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertRepoErrCode(t, err, apperr.ErrNotFound)
}

func TestTaxJobRepo_List_MapsRows(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockStore(ctrl)
	repo := NewTaxJobRepo(store)
	userID := uuid.New()
	jobID := uuid.New()

	row := sampleTaxJobRow(jobID, userID)
	store.EXPECT().CountTaxJobs(gomock.Any(), userID).Return(int64(1), nil).Times(1)
	store.EXPECT().ListTaxJobs(gomock.Any(), db.ListTaxJobsParams{UserID: userID, Limit: 20, Offset: 0}).Return([]db.TaxJob{row}, nil).Times(1)

	jobs, total, err := repo.List(context.Background(), userID, 20, 0)
	if err != nil {
		t.Fatalf("List() unexpected error: %v", err)
	}
	if total != 1 || len(jobs) != 1 {
		t.Fatalf("unexpected list response: total=%d len=%d", total, len(jobs))
	}
	if jobs[0].ID != jobID {
		t.Fatalf("job id mismatch: got %s want %s", jobs[0].ID, jobID)
	}
	if jobs[0].Summary == nil {
		t.Fatal("summary must be mapped")
	}
}

func TestMapTaxJobRow_InvalidPolicyJSON(t *testing.T) {
	row := sampleTaxJobRow(uuid.New(), uuid.New())
	row.PolicySnapshot = []byte("not-json")

	_, err := mapTaxJobRow(row)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertRepoErrCode(t, err, apperr.ErrInternal)
}

func TestTaxJobRepo_Requeue_SetsErrorAndRetryAt(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockStore(ctrl)
	repo := NewTaxJobRepo(store)
	jobID := uuid.New()
	retryAt := time.Now().UTC().Add(30 * time.Second)

	store.EXPECT().RequeueTaxJob(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, arg db.RequeueTaxJobParams) error {
		if arg.ID != jobID {
			t.Fatalf("job id mismatch: got %s want %s", arg.ID, jobID)
		}
		if !arg.RetryAt.Valid {
			t.Fatal("retry_at should be valid")
		}
		if arg.LastErrorCode == nil || *arg.LastErrorCode == "" {
			t.Fatal("last_error_code must be set")
		}
		if arg.LastErrorMessage == nil || *arg.LastErrorMessage == "" {
			t.Fatal("last_error_message must be set")
		}
		return nil
	}).Times(1)

	err := repo.Requeue(context.Background(), jobID, retryAt, "AGGREGATION_UNAVAILABLE", "upstream down")
	if err != nil {
		t.Fatalf("Requeue() unexpected error: %v", err)
	}
}

func sampleTaxJobRow(jobID, userID uuid.UUID) db.TaxJob {
	policyJSON, _ := json.Marshal(domain.TaxPolicy{Jurisdiction: domain.JurisdictionRU, CostBasisMethod: domain.FIFO})
	summaryJSON, _ := json.Marshal(domain.TaxSummary{UserID: userID, TaxYear: 2025})
	now := time.Now().UTC()
	errCode := "NONE"
	errMessage := ""
	auditKey := "audits/user/report.json"
	return db.TaxJob{
		ID:               jobID,
		UserID:           userID,
		TaxYear:          2025,
		PolicySnapshot:   policyJSON,
		Status:           string(domain.JobQueued),
		Attempts:         1,
		Summary:          summaryJSON,
		AuditObjectKey:   &auditKey,
		CreatedAt:        pgtype.Timestamptz{Time: now, Valid: true},
		StartedAt:        pgtype.Timestamptz{Time: now, Valid: true},
		FinishedAt:       pgtype.Timestamptz{Time: now, Valid: true},
		LastErrorCode:    &errCode,
		LastErrorMessage: &errMessage,
	}
}

func assertRepoErrCode(t *testing.T, err error, want apperr.ErrorCode) {
	t.Helper()
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		t.Fatalf("expected app error, got %T: %v", err, err)
	}
	if ae.Code != want {
		t.Fatalf("error code mismatch: got %s want %s", ae.Code, want)
	}
}
