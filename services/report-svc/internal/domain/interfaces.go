package domain

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type RenderPipelineUseCase interface {
	ProcessRenderRequested(ctx context.Context, event ReportRenderRequestedEvent) error
}

type RenderJobRepo interface {
	UpsertProcessing(ctx context.Context, job RenderJob) error
	MarkCompleted(ctx context.Context, reportID uuid.UUID, pdfObjectKey string) error
	MarkFailed(ctx context.Context, reportID uuid.UUID, errMsg string) error
}

type OutboxRepo interface {
	Insert(ctx context.Context, event OutboxEvent) error
	ListPending(ctx context.Context, maxAttempts, limit int32) ([]OutboxEvent, error)
	MarkPublished(ctx context.Context, id uuid.UUID) error
	MarkPublishFailed(ctx context.Context, id uuid.UUID, maxAttempts int32, lastError string) error
}

type InboxRepo interface {
	Register(ctx context.Context, eventID uuid.UUID) (bool, error)
}

type ObjectStorage interface {
	DownloadJSON(ctx context.Context, objectKey string, out any) error
	UploadPDF(ctx context.Context, objectKey string, pdf []byte) error
}

type PDFGenerator interface {
	Generate(dataset ReportDataset, opts PDFOptions) ([]byte, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, routingKey string, body []byte) error
}

type PDFOptions struct {
	TemplateVersion string
	MaxPreviewRows  int
}

type RenderResult struct {
	PDFObjectKey string
	Payload      json.RawMessage
}
