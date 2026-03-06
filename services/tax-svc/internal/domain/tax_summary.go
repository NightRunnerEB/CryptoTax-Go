package domain

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TaxSummary struct {
	TenantID     uuid.UUID
	TaxYear      int
	TotalIncome  decimal.Decimal
	TotalExpense decimal.Decimal
	TaxBase      decimal.Decimal
	TaxDue       decimal.Decimal
}
