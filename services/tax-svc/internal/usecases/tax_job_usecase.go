package usecases

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
)

type TaxJobUC struct {
	jobRepo     domain.TaxJobRepo
	profileRepo domain.TaxProfileRepo
}

func NewTaxJobUC(jobRepo domain.TaxJobRepo, profileRepo domain.TaxProfileRepo) *TaxJobUC {
	return &TaxJobUC{
		jobRepo:     jobRepo,
		profileRepo: profileRepo,
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
	return uc.jobRepo.Get(ctx, tenantID, jobID)
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
	return uc.jobRepo.List(ctx, tenantID, limit, offset)
}

var _ domain.TaxJobUseCase = (*TaxJobUC)(nil)
