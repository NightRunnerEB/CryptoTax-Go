package usecases

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
)

type TaxJobUC struct {
	jobRepo     domain.TaxJobRepo
	profileRepo domain.TaxProfileRepo
	storage     domain.ObjectStorage
}

func NewTaxJobUC(
	jobRepo domain.TaxJobRepo,
	profileRepo domain.TaxProfileRepo,
	storage domain.ObjectStorage,
) *TaxJobUC {
	return &TaxJobUC{
		jobRepo:     jobRepo,
		profileRepo: profileRepo,
		storage:     storage,
	}
}

func (uc *TaxJobUC) Enqueue(ctx context.Context, tenantID uuid.UUID, taxYear int, taxPolicy domain.TaxPolicy) (domain.TaxJob, error) {
	if tenantID == uuid.Nil {
		return domain.TaxJob{}, apperr.InvalidArgument("invalid tenant id", nil, apperr.FieldViolation{
			Field:       "tenant_id",
			Description: "required",
		})
	}
	if taxYear < 2000 || taxYear > time.Now().Year()+1 {
		return domain.TaxJob{}, apperr.InvalidArgument("invalid tax year", nil, apperr.FieldViolation{
			Field:       "tax_year",
			Description: "out of supported range",
		})
	}

	// TaxProfile must exist before enqueuing a calculation job.
	if _, err := uc.profileRepo.Get(ctx, tenantID); err != nil {
		return domain.TaxJob{}, err
	}

	taxPolicy = taxPolicy.Normalize()
	if err := taxPolicy.Validate(); err != nil {
		return domain.TaxJob{}, apperr.InvalidArgument("invalid cost basis method", err, apperr.FieldViolation{
			Field:       "tax_policy.cost_basis_method",
			Description: "must be FIFO, LIFO or AVG",
		})
	}

	job := domain.TaxJob{
		ID:             uuid.New(),
		TenantID:       tenantID,
		PolicySnapshot: taxPolicy,
		TaxYear:        taxYear,
		Status:         domain.JobQueued,
		Attempts:       0,
	}

	created, err := uc.jobRepo.Create(ctx, job)
	if err != nil {
		return domain.TaxJob{}, err
	}

	return created, nil
}

func (uc *TaxJobUC) GetStatus(ctx context.Context, tenantID, jobID uuid.UUID) (domain.TaxJob, error) {
	if tenantID == uuid.Nil || jobID == uuid.Nil {
		return domain.TaxJob{}, apperr.InvalidArgument("invalid ids", nil, apperr.FieldViolation{
			Field:       "tenant_id/report_id",
			Description: "required",
		})
	}
	job, err := uc.jobRepo.Get(ctx, tenantID, jobID)
	if err != nil {
		return domain.TaxJob{}, err
	}

	uc.attachPresignedURLs(ctx, &job)
	return job, nil
}

func (uc *TaxJobUC) List(ctx context.Context, tenantID uuid.UUID, limit, offset int32) ([]domain.TaxJob, int64, error) {
	if tenantID == uuid.Nil {
		return nil, 0, apperr.InvalidArgument("invalid tenant id", nil, apperr.FieldViolation{
			Field:       "tenant_id",
			Description: "required",
		})
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	jobs, total, err := uc.jobRepo.List(ctx, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	for i := range jobs {
		uc.attachPresignedURLs(ctx, &jobs[i])
	}
	return jobs, total, nil
}

var _ domain.TaxJobUseCase = (*TaxJobUC)(nil)

func (uc *TaxJobUC) attachPresignedURLs(ctx context.Context, job *domain.TaxJob) {
	if job.AuditObjectKey != nil {
		if url, err := uc.storage.PresignGet(ctx, *job.AuditObjectKey); err == nil && strings.TrimSpace(url) != "" {
			job.AuditZipURL = &url
		}
	}
	if job.ReportObjectKey != nil {
		if url, err := uc.storage.PresignGet(ctx, *job.ReportObjectKey); err == nil && strings.TrimSpace(url) != "" {
			job.ReportURL = &url
		}
	}
}
