package events

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ExpenseEvent = расходы.
type ExpenseEvent struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	OccurredAt time.Time

	AmountFiat decimal.Decimal // valuation snapshot in jurisdiction fiat

	Asset string
	Qty   decimal.Decimal

	LinkedRealizationID *uuid.UUID // optional: link fee to a specific realization

	Kind     ExpenseKind
	Evidence Evidence
}

type ExpenseKind string

const (
	ExpenseTradeFee       ExpenseKind = "TRADE_FEE"
	ExpenseFundingFee     ExpenseKind = "FUNDING_FEE"
	ExpenseNetworkFee     ExpenseKind = "NETWORK_FEE"
	ExpenseManual         ExpenseKind = "MANUAL"
	ExpenseDerivativeLoss ExpenseKind = "DERIVATIVE_LOSS" // negative derivative pnl if modeled as expense
)

type ExpenseLot struct {
	ExpenseID uuid.UUID
	LotID     uuid.UUID
	Qty       decimal.Decimal
	CostFiat  decimal.Decimal
}
