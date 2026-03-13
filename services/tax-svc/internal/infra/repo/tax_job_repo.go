package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/NightRunner/CryptoTax-Go/services/tax-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
)

type TaxJobRepo struct {
	store db.Store
}

func NewTaxJobRepo(store db.Store) *TaxJobRepo {
	return &TaxJobRepo{store: store}
}

func (r *TaxJobRepo) Create(ctx context.Context, job domain.TaxJob) (domain.TaxJob, error) {
	policySnapshot, err := json.Marshal(job.PolicySnapshot)
	if err != nil {
		return domain.TaxJob{}, apperr.Internal("marshal policy snapshot failed", err, nil)
	}

	row, err := r.store.CreateTaxJob(ctx, db.CreateTaxJobParams{
		ID:             job.ID,
		TenantID:       job.TenantID,
		TaxYear:        int32(job.TaxYear),
		PolicySnapshot: policySnapshot,
		Status:         string(job.Status),
		Attempts:       int32(job.Attempts),
	})
	if err != nil {
		return domain.TaxJob{}, apperr.Internal("create tax job failed", err, map[string]string{
			"tenant_id": job.TenantID.String(),
			"job_id":    job.ID.String(),
		})
	}

	return mapTaxJobRow(row)
}

func (r *TaxJobRepo) Get(ctx context.Context, tenantID, jobID uuid.UUID) (domain.TaxJob, error) {
	row, err := r.store.GetTaxJob(ctx, db.GetTaxJobParams{
		TenantID: tenantID,
		ID:       jobID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.TaxJob{}, apperr.NotFound("tax job not found", apperr.Resource{
				Type: "tax_job",
				Name: jobID.String(),
			}, err)
		}
		return domain.TaxJob{}, apperr.Internal("get tax job failed", err, map[string]string{
			"tenant_id": tenantID.String(),
			"job_id":    jobID.String(),
		})
	}
	return mapTaxJobRow(row)
}

func (r *TaxJobRepo) List(ctx context.Context, tenantID uuid.UUID, limit, offset int32) ([]domain.TaxJob, int64, error) {
	total, err := r.store.CountTaxJobs(ctx, tenantID)
	if err != nil {
		return nil, 0, apperr.Internal("count tax jobs failed", err, map[string]string{
			"tenant_id": tenantID.String(),
		})
	}

	rows, err := r.store.ListTaxJobs(ctx, db.ListTaxJobsParams{
		TenantID: tenantID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, 0, apperr.Internal("list tax jobs failed", err, map[string]string{
			"tenant_id": tenantID.String(),
		})
	}

	out := make([]domain.TaxJob, 0, len(rows))
	for _, row := range rows {
		job, mapErr := mapTaxJobRow(row)
		if mapErr != nil {
			return nil, 0, mapErr
		}
		out = append(out, job)
	}
	return out, total, nil
}

func (r *TaxJobRepo) ClaimNextQueued(ctx context.Context) (*domain.TaxJob, error) {
	row, err := r.store.ClaimNextQueuedTaxJob(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, apperr.Internal("claim queued tax job failed", err, nil)
	}
	job, err := mapTaxJobRow(row)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *TaxJobRepo) SaveResult(
	ctx context.Context,
	jobID uuid.UUID,
	summary domain.TaxSummary,
	auditObjectKey *string,
	reportObjectKey *string,
) error {
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return apperr.Internal("marshal summary failed", err, map[string]string{
			"job_id": jobID.String(),
		})
	}

	if err := r.store.SaveTaxJobResult(ctx, db.SaveTaxJobResultParams{
		ID:              jobID,
		Summary:         summaryJSON,
		AuditObjectKey:  auditObjectKey,
		ReportObjectKey: reportObjectKey,
	}); err != nil {
		return apperr.Internal("save tax job result failed", err, map[string]string{
			"job_id": jobID.String(),
		})
	}

	return nil
}

func (r *TaxJobRepo) Requeue(
	ctx context.Context,
	jobID uuid.UUID,
	retryAt time.Time,
	errCode string,
	errMsg string,
) error {
	errorCode := errCode
	errorMessage := errMsg
	if err := r.store.RequeueTaxJob(ctx, db.RequeueTaxJobParams{
		ID:               jobID,
		RetryAt:          pgtype.Timestamptz{Time: retryAt, Valid: true},
		LastErrorCode:    &errorCode,
		LastErrorMessage: &errorMessage,
	}); err != nil {
		return apperr.Internal("requeue tax job failed", err, map[string]string{
			"job_id": jobID.String(),
		})
	}
	return nil
}

func (r *TaxJobRepo) MarkFailed(ctx context.Context, jobID uuid.UUID, errCode, errMsg string) error {
	errorCode := errCode
	errorMessage := errMsg
	if err := r.store.MarkTaxJobFailed(ctx, db.MarkTaxJobFailedParams{
		ID:               jobID,
		LastErrorCode:    &errorCode,
		LastErrorMessage: &errorMessage,
	}); err != nil {
		return apperr.Internal("mark tax job failed failed", err, map[string]string{
			"job_id": jobID.String(),
		})
	}
	return nil
}

func (r *TaxJobRepo) MarkCanceled(ctx context.Context, jobID uuid.UUID) error {
	if err := r.store.MarkTaxJobCanceled(ctx, jobID); err != nil {
		return apperr.Internal("mark tax job canceled failed", err, map[string]string{
			"job_id": jobID.String(),
		})
	}
	return nil
}

func mapTaxJobRow(row db.TaxJob) (domain.TaxJob, error) {
	var policy domain.TaxPolicy
	if err := json.Unmarshal(row.PolicySnapshot, &policy); err != nil {
		return domain.TaxJob{}, apperr.Internal("unmarshal policy snapshot failed", err, map[string]string{
			"job_id": row.ID.String(),
		})
	}
	if policy.CostBasisMethod == "" {
		return domain.TaxJob{}, apperr.Internal("invalid policy snapshot in tax job", nil, map[string]string{
			"job_id": row.ID.String(),
		})
	}

	var summary *domain.TaxSummary
	if len(row.Summary) > 0 {
		var decoded domain.TaxSummary
		if err := json.Unmarshal(row.Summary, &decoded); err != nil {
			return domain.TaxJob{}, apperr.Internal("unmarshal summary failed", err, nil)
		}
		summary = &decoded
	}

	var startedAt *time.Time
	if row.StartedAt.Valid {
		t := row.StartedAt.Time
		startedAt = &t
	}

	var finishedAt *time.Time
	if row.FinishedAt.Valid {
		t := row.FinishedAt.Time
		finishedAt = &t
	}

	var createdAt time.Time
	if row.CreatedAt.Valid {
		createdAt = row.CreatedAt.Time
	}

	return domain.TaxJob{
		ID:               row.ID,
		TenantID:         row.TenantID,
		PolicySnapshot:   policy,
		TaxYear:          int(row.TaxYear),
		Status:           domain.TaxJobStatus(row.Status),
		Attempts:         int(row.Attempts),
		Summary:          summary,
		AuditObjectKey:   row.AuditObjectKey,
		ReportObjectKey:  row.ReportObjectKey,
		CreatedAt:        createdAt,
		StartedAt:        startedAt,
		FinishedAt:       finishedAt,
		LastErrorCode:    row.LastErrorCode,
		LastErrorMessage: row.LastErrorMessage,
	}, nil
}

var _ domain.TaxJobRepo = (*TaxJobRepo)(nil)
