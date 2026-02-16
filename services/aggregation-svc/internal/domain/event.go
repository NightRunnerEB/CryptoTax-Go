package domain

import "github.com/google/uuid"

type ImportEvent struct {
	EventId  uuid.UUID `json:"event_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	ImportID uuid.UUID `json:"import_id"`
}
