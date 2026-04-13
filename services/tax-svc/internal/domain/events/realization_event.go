package events

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// RealizationEvent = выбытие актива (продажа/обмен/дарение/утрата/сжигание).
// IMPORTANT: fees are NOT stored here (fees are ExpenseEvent).
// CostBasisFiat is computed via RealizationLots (cost-basis allocations).
type RealizationEvent struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	OccurredAt time.Time

	Asset string
	Qty   decimal.Decimal

	ProceedsFiat  decimal.Decimal
	CostBasisFiat decimal.Decimal // SUM(RealizationLot.CostFiat)

	Kind     RealizationKind
	Evidence Evidence
}

type Evidence struct {
	SourceTxID uuid.UUID // AggregatedTransaction.ID
	Source     string    // Bybit/OKX/...
	OrderID    *string
	TxHash     *string
}

type RealizationKind string

const (
	RealizationSellFiat RealizationKind = "SELL_FIAT"
	RealizationSwapOut  RealizationKind = "SWAP_OUT"
	RealizationSpend    RealizationKind = "SPEND"
	RealizationGift     RealizationKind = "GIFT_OUT"
	RealizationBurn     RealizationKind = "BURN"
	RealizationLost     RealizationKind = "LOST"
	RealizationStolen   RealizationKind = "STOLEN"
)

// RealizationLot = FIFO allocation proof (one realization consumes 1..N lots).
type RealizationLot struct {
	RealizationID uuid.UUID
	LotID         uuid.UUID
	Qty           decimal.Decimal
	CostFiat      decimal.Decimal
}
