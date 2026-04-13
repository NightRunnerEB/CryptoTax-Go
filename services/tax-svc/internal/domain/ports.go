package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AggregatedTxProvider interface {
	ListTransactionsByRange(ctx context.Context, userID uuid.UUID, fromUTC, toUTC time.Time, targetFiat string) ([]AggregatedTransaction, error)
}

type ObjectStorage interface {
	UploadJSON(ctx context.Context, objectKey string, payload any) error
	PresignGet(ctx context.Context, objectKey string) (string, error)
}

type ReportRenderRequest struct {
	ReportID         uuid.UUID
	UserID           uuid.UUID
	Jurisdiction     string
	TaxYear          int32
	DatasetObjectKey string
	TemplateVersion  string
}

type ReportClient interface {
	RequestRender(ctx context.Context, req ReportRenderRequest) error
}
