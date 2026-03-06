package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AggregatedTxProvider interface {
	ListTransactionsByRange(ctx context.Context, tenantID uuid.UUID, fromUTC, toUTC time.Time) ([]AggregatedTransaction, error)
}

type ObjectStorage interface {
	UploadJSON(ctx context.Context, objectKey string, payload any) error
	PresignGet(ctx context.Context, objectKey string, ttl time.Duration) (string, error)
}

type ReportRenderRequest struct {
	ReportID         uuid.UUID
	TenantID         uuid.UUID
	Jurisdiction     string
	TaxYear          int32
	DatasetObjectKey string
	TemplateVersion  string
}

type ReportClient interface {
	RequestRender(ctx context.Context, req ReportRenderRequest) error
}
