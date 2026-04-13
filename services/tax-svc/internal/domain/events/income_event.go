package events

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// IncomeEvent = доход без продажи (airdrop/reward/gift-in/positive derivative pnl).
// If Asset/Qty present -> you MUST create a Lot with CostFiat = AmountFiat.
type IncomeEvent struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	OccurredAt time.Time

	AmountFiat decimal.Decimal

	Asset string
	Qty   decimal.Decimal

	Kind     IncomeKind
	Evidence Evidence
}

type IncomeKind string

const (
	IncomeAirdrop       IncomeKind = "AIRDROP"
	IncomeStakingReward IncomeKind = "STAKING_REWARD"
	IncomeGiftIn        IncomeKind = "GIFT_IN"
	IncomeDerivativePnL IncomeKind = "DERIVATIVE_PNL_POSITIVE"
)
