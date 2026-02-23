package domain

import (
	"time"

	"github.com/google/uuid"
)

type RenderJobStatus string

const (
	RenderJobStatusProcessing RenderJobStatus = "processing"
	RenderJobStatusCompleted  RenderJobStatus = "completed"
	RenderJobStatusFailed     RenderJobStatus = "failed"
)

type RenderJob struct {
	ReportID         uuid.UUID       `json:"report_id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	Status           RenderJobStatus `json:"status"`
	StartedAt        time.Time       `json:"started_at"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	Error            *string         `json:"error,omitempty"`
	DatasetObjectKey string          `json:"dataset_object_key"`
	PDFObjectKey     *string         `json:"pdf_object_key,omitempty"`
	UpdatedAt        time.Time       `json:"updated_at"`
}
