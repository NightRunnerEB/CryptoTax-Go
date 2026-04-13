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

func (uc *TaxJobUC) Enqueue(ctx context.Context, userID uuid.UUID, taxYear int, taxPolicy domain.TaxPolicy) (domain.TaxJob, error) {
	if userID == uuid.Nil {
		return domain.TaxJob{}, apperr.InvalidArgument("invalid user id", nil, apperr.FieldViolation{
			Field:       "user_id",
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
	if _, err := uc.profileRepo.Get(ctx, userID); err != nil {
		return domain.TaxJob{}, err
	}

	taxPolicy = taxPolicy.Normalize()
	if err := taxPolicy.Validate(); err != nil {
		return domain.TaxJob{}, apperr.InvalidArgument(
			"invalid tax_policy",
			err,
			apperr.FieldViolation{
				Field:       "tax_policy.cost_basis_method",
				Description: "must be FIFO, LIFO or AVG",
			},
			apperr.FieldViolation{
				Field:       "tax_policy.jurisdiction",
				Description: "must be a supported jurisdiction",
			},
		)
	}

	job := domain.TaxJob{
		ID:             uuid.New(),
		UserID:         userID,
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

func (uc *TaxJobUC) GetStatus(ctx context.Context, userID, jobID uuid.UUID) (domain.TaxJob, error) {
	if userID == uuid.Nil || jobID == uuid.Nil {
		return domain.TaxJob{}, apperr.InvalidArgument("invalid ids", nil, apperr.FieldViolation{
			Field:       "user_id/report_id",
			Description: "required",
		})
	}
	job, err := uc.jobRepo.Get(ctx, userID, jobID)
	if err != nil {
		return domain.TaxJob{}, err
	}

	uc.attachPresignedURLs(ctx, &job)
	return job, nil
}

func (uc *TaxJobUC) List(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]domain.TaxJob, int64, error) {
	if userID == uuid.Nil {
		return nil, 0, apperr.InvalidArgument("invalid user id", nil, apperr.FieldViolation{
			Field:       "user_id",
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
	jobs, total, err := uc.jobRepo.List(ctx, userID, limit, offset)
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
