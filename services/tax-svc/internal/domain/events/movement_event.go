package events

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type MovementEvent struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	OccurredAt time.Time

	Asset string
	Qty   decimal.Decimal

	Direction MovementDirection
	Evidence  Evidence
}

type MovementDirection string

const (
	MovementIn       MovementDirection = "IN"
	MovementOut      MovementDirection = "OUT"
	MovementInternal MovementDirection = "INTERNAL"
)
