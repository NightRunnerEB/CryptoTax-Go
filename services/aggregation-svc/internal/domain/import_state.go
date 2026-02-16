package domain

import (
	"time"

	"github.com/google/uuid"
)

type ImportStatus string

const (
	ImportStatusProcessing ImportStatus = "processing"
	ImportStatusCompleted  ImportStatus = "completed"
	ImportStatusFailed     ImportStatus = "failed"
)

type AggregationImportState struct {
	TenantID    uuid.UUID    `json:"tenant_id"`
	ImportID    uuid.UUID    `json:"import_id"`
	EventId     uuid.UUID    `json:"event_id"`
	Status      ImportStatus `json:"status"`
	StartedAt   time.Time    `json:"started_at"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
	Error       *string      `json:"error,omitempty"`
}
