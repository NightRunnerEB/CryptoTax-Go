package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	db "github.com/NightRunner/CryptoTax-Go/services/report-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain/error"
)

type renderUC struct {
	store db.Store

	inboxRepo  domain.InboxRepo
	renderRepo domain.RenderJobRepo
	storage    domain.ObjectStorage
	pdf        domain.PDFGenerator

	templateVersion string
	maxPreviewRows  int
}

func NewRenderUC(
	store db.Store,
	inboxRepo domain.InboxRepo,
	renderRepo domain.RenderJobRepo,
	storage domain.ObjectStorage,
	pdf domain.PDFGenerator,
	templateVersion string,
	maxPreviewRows int,
) domain.RenderPipelineUseCase {
	if strings.TrimSpace(templateVersion) == "" {
		templateVersion = "v1"
	}
	if maxPreviewRows <= 0 {
		maxPreviewRows = 20
	}
	return &renderUC{
		store:           store,
		inboxRepo:       inboxRepo,
		renderRepo:      renderRepo,
		storage:         storage,
		pdf:             pdf,
		templateVersion: templateVersion,
		maxPreviewRows:  maxPreviewRows,
	}
}

func (u *renderUC) ProcessRenderRequested(ctx context.Context, event domain.ReportRenderRequestedEvent) error {
	if event.EventID == uuid.Nil || event.ReportID == uuid.Nil || event.UserID == uuid.Nil {
		return apperr.InvalidArgument("invalid render request event", nil, apperr.FieldViolation{
			Field:       "event",
			Description: "event_id/report_id/user_id are required",
		})
	}
	if strings.TrimSpace(event.DatasetObjectKey) == "" {
		return apperr.InvalidArgument("invalid dataset object key", nil, apperr.FieldViolation{
			Field:       "dataset_object_key",
			Description: "required",
		})
	}

	inserted, err := u.inboxRepo.Register(ctx, event.EventID)
	if err != nil {
		return err
	}
	if !inserted {
		return nil
	}

	if err := u.renderRepo.UpsertProcessing(ctx, domain.RenderJob{
		ReportID:         event.ReportID,
		UserID:           event.UserID,
		Status:           domain.RenderJobStatusProcessing,
		DatasetObjectKey: event.DatasetObjectKey,
	}); err != nil {
		return err
	}

	var dataset domain.ReportDataset
	if err := u.storage.DownloadJSON(ctx, event.DatasetObjectKey, &dataset); err != nil {
		return u.persistFailed(ctx, event.ReportID, err.Error())
	}

	pdfData, err := u.pdf.Generate(dataset, domain.PDFOptions{
		TemplateVersion: normalizeTemplateVersion(event.TemplateVersion, u.templateVersion),
		MaxPreviewRows:  u.maxPreviewRows,
	})
	if err != nil {
		return u.persistFailed(ctx, event.ReportID, err.Error())
	}

	pdfObjectKey := fmt.Sprintf("reports/%s/%s.pdf", event.UserID.String(), event.ReportID.String())
	if err := u.storage.UploadPDF(ctx, pdfObjectKey, pdfData); err != nil {
		return u.persistFailed(ctx, event.ReportID, err.Error())
	}

	successEvent := domain.ReportRenderedEvent{
		EventID:      uuid.New(),
		ReportID:     event.ReportID,
		PDFObjectKey: pdfObjectKey,
	}
	payload, err := json.Marshal(successEvent)
	if err != nil {
		return u.persistFailed(ctx, event.ReportID, "marshal rendered event failed: "+err.Error())
	}

	if err := u.store.ExecTx(ctx, func(q *db.Queries) error {
		if err := q.MarkRenderJobCompleted(ctx, db.MarkRenderJobCompletedParams{
			ReportID:     event.ReportID,
			PdfObjectKey: &pdfObjectKey,
		}); err != nil {
			return err
		}
		return q.InsertOutboxEvent(ctx, db.InsertOutboxEventParams{
			ID:            uuid.New(),
			AggregateType: "render_job",
			AggregateID:   event.ReportID,
			EventType:     domain.EventTypeReportRendered,
			Payload:       payload,
			Status:        string(domain.OutboxStatusPending),
		})
	}); err != nil {
		return apperr.Internal("persist render success failed", err, map[string]string{
			"report_id": event.ReportID.String(),
		})
	}

	return nil
}

func (u *renderUC) persistFailed(ctx context.Context, reportID uuid.UUID, failure string) error {
	if strings.TrimSpace(failure) == "" {
		failure = "render failed"
	}
	event := domain.ReportRenderFailedEvent{
		EventID:  uuid.New(),
		ReportID: reportID,
		Error:    failure,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return apperr.Internal("marshal failed event failed", err, map[string]string{
			"report_id": reportID.String(),
		})
	}

	if err := u.store.ExecTx(ctx, func(q *db.Queries) error {
		if err := q.MarkRenderJobFailed(ctx, db.MarkRenderJobFailedParams{
			ReportID: reportID,
			Error:    &failure,
		}); err != nil {
			return err
		}
		return q.InsertOutboxEvent(ctx, db.InsertOutboxEventParams{
			ID:            uuid.New(),
			AggregateType: "render_job",
			AggregateID:   reportID,
			EventType:     domain.EventTypeReportRenderFailed,
			Payload:       payload,
			Status:        string(domain.OutboxStatusPending),
		})
	}); err != nil {
		return apperr.Internal("persist render failure failed", err, map[string]string{
			"report_id": reportID.String(),
		})
	}
	return nil
}

func normalizeTemplateVersion(candidate, fallback string) string {
	candidate = strings.TrimSpace(candidate)
	if candidate != "" {
		return candidate
	}
	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		return "v1"
	}
	return fallback
}

var _ domain.RenderPipelineUseCase = (*renderUC)(nil)
