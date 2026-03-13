package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type TaxJobStatus string

const (
	JobQueued   TaxJobStatus = "queued"
	JobRunning  TaxJobStatus = "running"
	JobSuccess  TaxJobStatus = "success"
	JobFailed   TaxJobStatus = "failed"
	JobCanceled TaxJobStatus = "canceled"
)

type TaxJob struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	PolicySnapshot TaxPolicy
	TaxYear        int

	Status   TaxJobStatus
	Attempts int

	Summary *TaxSummary

	AuditObjectKey  *string
	ReportObjectKey *string
	AuditZipURL     *string
	ReportURL       *string

	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time

	LastErrorCode    *string
	LastErrorMessage *string
}

type TaxJobUseCase interface {
	Enqueue(ctx context.Context, tenantID uuid.UUID, taxYear int, taxPolicy TaxPolicy) (TaxJob, error)
	GetStatus(ctx context.Context, tenantID, jobID uuid.UUID) (TaxJob, error)
	List(ctx context.Context, tenantID uuid.UUID, limit, offset int32) ([]TaxJob, int64, error)
}

type TaxJobRepo interface {
	Create(ctx context.Context, job TaxJob) (TaxJob, error)
	Get(ctx context.Context, tenantID, jobID uuid.UUID) (TaxJob, error)
	List(ctx context.Context, tenantID uuid.UUID, limit, offset int32) ([]TaxJob, int64, error)
	ClaimNextQueued(ctx context.Context) (*TaxJob, error)
	Requeue(ctx context.Context, jobID uuid.UUID, retryAt time.Time, errCode, errMsg string) error
	SaveResult(ctx context.Context, jobID uuid.UUID, summary TaxSummary, auditObjectKey *string, reportObjectKey *string) error
	MarkFailed(ctx context.Context, jobID uuid.UUID, errCode, errMsg string) error
	MarkCanceled(ctx context.Context, jobID uuid.UUID) error
}

type TaxJobWorkerUseCase interface {
	ProcessNextQueuedJob(ctx context.Context) (bool, error)
}
