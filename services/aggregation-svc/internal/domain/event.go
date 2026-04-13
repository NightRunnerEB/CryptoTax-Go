package domain

import "github.com/google/uuid"

type ImportEvent struct {
	EventId  uuid.UUID `json:"event_id"`
	UserID   uuid.UUID `json:"user_id"`
	ImportID uuid.UUID `json:"import_id"`
}
