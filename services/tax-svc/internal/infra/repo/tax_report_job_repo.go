package repository

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	db "github.com/NightRunner/CryptoTax-Go/services/tax-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/google/uuid"
)

type taxReportJobRepo struct {
	store db.Store
}

func NewTaxReportJobRepo(store db.Store) domain.TaxReportJobRepo {
	return &taxReportJobRepo{store: store}
}

func (r *taxReportJobRepo) Create(ctx context.Context, job domain.TaxReportJob) (domain.TaxReportJob, error) {
	row, err := r.store.CreateTaxReportJob(ctx, db.CreateTaxReportJobParams{
		ID:           job.ID,
		TenantID:     job.TenantID,
		TaxYear:      job.TaxYear,
		Jurisdiction: job.Jurisdiction,
		Status:       string(job.Status),
		Params:       job.Params,
	})
	if err != nil {
		return domain.TaxReportJob{}, apperr.Internal("create tax report job failed", err, map[string]string{
			"tenant_id": job.TenantID.String(),
		})
	}

	return mapTaxReportJobModel(row), nil
}

func (r *taxReportJobRepo) Get(ctx context.Context, tenantID, reportID uuid.UUID) (domain.TaxReportJob, error) {
	row, err := r.store.GetTaxReportJob(ctx, db.GetTaxReportJobParams{
		TenantID: tenantID,
		ID:       reportID,
	})
	if err != nil {
		return domain.TaxReportJob{}, apperr.Internal("get tax report job failed", err, map[string]string{
			"tenant_id": tenantID.String(),
			"report_id": reportID.String(),
		})
	}

	return mapTaxReportJobModel(row), nil
}

func (r *taxReportJobRepo) List(ctx context.Context, tenantID uuid.UUID, taxYear, limit, offset int32) (domain.TaxReportJobPage, error) {
	total, err := r.store.CountTaxReportJobs(ctx, db.CountTaxReportJobsParams{
		TenantID: tenantID,
		Column2:  taxYear,
	})
	if err != nil {
		return domain.TaxReportJobPage{}, apperr.Internal("count report jobs failed", err, map[string]string{
			"tenant_id": tenantID.String(),
			"tax_year":  strconv.Itoa(int(taxYear)),
		})
	}

	rows, err := r.store.ListTaxReportJobs(ctx, db.ListTaxReportJobsParams{
		TenantID: tenantID,
		Column2:  taxYear,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return domain.TaxReportJobPage{}, apperr.Internal("list report jobs failed", err, map[string]string{
			"tenant_id": tenantID.String(),
			"tax_year":  strconv.Itoa(int(taxYear)),
		})
	}

	out := make([]domain.TaxReportJob, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapTaxReportJobModel(row))
	}

	return domain.TaxReportJobPage{Reports: out, Total: total}, nil
}

func (r *taxReportJobRepo) MarkProcessing(ctx context.Context, reportID uuid.UUID) (int64, error) {
	rows, err := r.store.MarkTaxReportJobProcessing(ctx, reportID)
	if err != nil {
		return 0, apperr.Internal("mark processing failed", err, map[string]string{
			"report_id": reportID.String(),
		})
	}
	return rows, nil
}

func (r *taxReportJobRepo) UpdateDataset(ctx context.Context, reportID uuid.UUID, datasetObjectKey string, summary json.RawMessage) error {
	if err := r.store.UpdateTaxReportJobDataset(ctx, db.UpdateTaxReportJobDatasetParams{
		ID:               reportID,
		DatasetObjectKey: &datasetObjectKey,
		Summary:          summary,
	}); err != nil {
		return apperr.Internal("update dataset failed", err, map[string]string{
			"report_id": reportID.String(),
		})
	}
	return nil
}

func (r *taxReportJobRepo) MarkCompleted(ctx context.Context, reportID uuid.UUID, pdfObjectKey string) error {
	if err := r.store.MarkTaxReportJobCompleted(ctx, db.MarkTaxReportJobCompletedParams{
		ID:           reportID,
		PdfObjectKey: &pdfObjectKey,
	}); err != nil {
		return apperr.Internal("mark completed failed", err, map[string]string{
			"report_id": reportID.String(),
		})
	}
	return nil
}

func (r *taxReportJobRepo) MarkFailed(ctx context.Context, reportID uuid.UUID, errMsg string) error {
	if errMsg == "" {
		errMsg = "unknown error"
	}
	if err := r.store.MarkTaxReportJobFailed(ctx, db.MarkTaxReportJobFailedParams{
		ID:    reportID,
		Error: &errMsg,
	}); err != nil {
		return apperr.Internal("mark failed failed", err, map[string]string{
			"report_id": reportID.String(),
		})
	}
	return nil
}

func mapTaxReportJobModel(row db.TaxReportJob) domain.TaxReportJob {
	var startedAt *time.Time
	if row.StartedAt.Valid {
		v := fromTimestamptz(row.StartedAt)
		startedAt = &v
	}
	var completedAt *time.Time
	if row.CompletedAt.Valid {
		v := fromTimestamptz(row.CompletedAt)
		completedAt = &v
	}

	return domain.TaxReportJob{
		ID:               row.ID,
		TenantID:         row.TenantID,
		TaxYear:          row.TaxYear,
		Jurisdiction:     row.Jurisdiction,
		Status:           domain.ReportJobStatus(row.Status),
		RequestedAt:      fromTimestamptz(row.RequestedAt),
		StartedAt:        startedAt,
		CompletedAt:      completedAt,
		Error:            row.Error,
		Params:           row.Params,
		Summary:          row.Summary,
		DatasetObjectKey: row.DatasetObjectKey,
		PDFObjectKey:     row.PdfObjectKey,
	}
}

var _ domain.TaxReportJobRepo = (*taxReportJobRepo)(nil)
