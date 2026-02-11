package domain

import "github.com/google/uuid"

type ImportCompletedEvent struct {
	TenantID uuid.UUID `json:"tenant_id"`
	ImportID uuid.UUID `json:"import_id"`
	Wallet   string    `json:"wallet"`
	Source   string    `json:"source"`
}
