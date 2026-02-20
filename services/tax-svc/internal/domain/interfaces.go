package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type TaxProfileUseCase interface {
	Get(ctx context.Context, tenantID uuid.UUID) (TaxProfile, error)
	Upsert(ctx context.Context, profile TaxProfile) (TaxProfile, error)
}

type TaxpayerProfileUseCase interface {
	Get(ctx context.Context, tenantID uuid.UUID) (TaxpayerProfile, error)
	Upsert(ctx context.Context, profile TaxpayerProfile) (TaxpayerProfile, error)
}

type ReportUseCase interface {
	StartReport(ctx context.Context, params StartReportParams) (TaxReportJob, error)
	GetReportStatus(ctx context.Context, tenantID, reportID uuid.UUID) (ReportStatusView, error)
	ListReports(ctx context.Context, tenantID uuid.UUID, taxYear, limit, offset int32) (TaxReportJobPage, error)
}

type ReportPipelineUseCase interface {
	ProcessQueuedReport(ctx context.Context, event TaxReportJobRequestedEvent) error
	HandleReportRendered(ctx context.Context, event ReportRenderedEvent) error
	HandleReportRenderFailed(ctx context.Context, event ReportRenderFailedEvent) error
}

type TaxProfileRepo interface {
	Get(ctx context.Context, tenantID uuid.UUID) (TaxProfile, error)
	Upsert(ctx context.Context, profile TaxProfile) (TaxProfile, error)
}

type TaxpayerProfileRepo interface {
	Get(ctx context.Context, tenantID uuid.UUID) (TaxpayerProfile, error)
	Upsert(ctx context.Context, profile TaxpayerProfile) (TaxpayerProfile, error)
}

type TaxReportJobRepo interface {
	Create(ctx context.Context, job TaxReportJob) (TaxReportJob, error)
	Get(ctx context.Context, tenantID, reportID uuid.UUID) (TaxReportJob, error)
	List(ctx context.Context, tenantID uuid.UUID, taxYear, limit, offset int32) (TaxReportJobPage, error)
	MarkProcessing(ctx context.Context, reportID uuid.UUID) (int64, error)
	UpdateDataset(ctx context.Context, reportID uuid.UUID, datasetObjectKey string, summary json.RawMessage) error
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

type AggregationClient interface {
	ListTransactionsByRange(ctx context.Context, tenantID uuid.UUID, fromUTC, toUTC time.Time, limit, offset int32) ([]AggregatedTransaction, error)
}

type ObjectStorage interface {
	UploadJSON(ctx context.Context, objectKey string, payload any) error
	PresignGet(ctx context.Context, objectKey string, ttl time.Duration) (string, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, routingKey string, body []byte) error
}
