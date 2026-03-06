package domain

import (
	"fmt"

	"github.com/shopspring/decimal"
)

type Jurisdiction string

const (
	JurisdictionRU Jurisdiction = "RU"
	JurisdictionKZ Jurisdiction = "KZ"
)

func (j Jurisdiction) FiatCurrency() string {
	switch j {
	case JurisdictionRU:
		return "RUB"
	case JurisdictionKZ:
		return "KZT"
	default:
		return ""
	}
}

func (j Jurisdiction) Validate() error {
	if j.FiatCurrency() == "" {
		return fmt.Errorf("unsupported jurisdiction: %s", j)
	}
	return nil
}

type NdflPayload struct {
	TaxYear          int
	Income           decimal.Decimal
	Expense          decimal.Decimal
	TaxBase          decimal.Decimal
	TaxDue           decimal.Decimal
	Appendix6Expense decimal.Decimal
	IncomeLines      []NdflIncomeLine
}
type NdflIncomeLine struct {
	Source      string // биржа
	Income      decimal.Decimal
	TaxWithheld decimal.Decimal
}
