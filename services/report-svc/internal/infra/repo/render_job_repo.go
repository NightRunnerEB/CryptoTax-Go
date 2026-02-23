package repository

import (
	"context"

	db "github.com/NightRunner/CryptoTax-Go/services/report-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain/error"
	"github.com/google/uuid"
)

type renderJobRepo struct {
	store db.Store
}

func NewRenderJobRepo(store db.Store) domain.RenderJobRepo {
	return &renderJobRepo{store: store}
}

func (r *renderJobRepo) UpsertProcessing(ctx context.Context, job domain.RenderJob) error {
	if err := r.store.UpsertRenderJobProcessing(ctx, db.UpsertRenderJobProcessingParams{
		ReportID:         job.ReportID,
		TenantID:         job.TenantID,
		DatasetObjectKey: job.DatasetObjectKey,
	}); err != nil {
		return apperr.Internal("upsert render job processing failed", err, map[string]string{
			"report_id": job.ReportID.String(),
		})
	}
	return nil
}

func (r *renderJobRepo) MarkCompleted(ctx context.Context, reportID uuid.UUID, pdfObjectKey string) error {
	if err := r.store.MarkRenderJobCompleted(ctx, db.MarkRenderJobCompletedParams{
		ReportID:     reportID,
		PdfObjectKey: stringPtr(pdfObjectKey),
	}); err != nil {
		return apperr.Internal("mark render job completed failed", err, map[string]string{
			"report_id": reportID.String(),
		})
	}
	return nil
}

func (r *renderJobRepo) MarkFailed(ctx context.Context, reportID uuid.UUID, errMsg string) error {
	if errMsg == "" {
		errMsg = "unknown error"
	}
	if err := r.store.MarkRenderJobFailed(ctx, db.MarkRenderJobFailedParams{
		ReportID: reportID,
		Error:    stringPtr(errMsg),
	}); err != nil {
		return apperr.Internal("mark render job failed failed", err, map[string]string{
			"report_id": reportID.String(),
		})
	}
	return nil
}

var _ domain.RenderJobRepo = (*renderJobRepo)(nil)
