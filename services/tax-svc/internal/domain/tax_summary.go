package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type P2PIncome struct {
	OccurredAt time.Time
	Qty        decimal.Decimal
	GainFiat   decimal.Decimal
}

type TaxSummary struct {
	UserID       uuid.UUID
	TaxYear      int
	TotalIncome  decimal.Decimal
	TotalTrade   decimal.Decimal
	TotalP2P     []P2PIncome
	TotalExpense decimal.Decimal
	TaxBase      decimal.Decimal
	TaxDue       decimal.Decimal
}
