package domain

import (
	"time"

	"github.com/google/uuid"
)

type TaxProfile struct {
	TenantID        uuid.UUID `json:"tenant_id"`
	Jurisdiction    string    `json:"jurisdiction"`
	CostBasisMethod string    `json:"cost_basis_method"`
	Timezone        string    `json:"timezone"`

	TreatSwapAsDisposition      bool `json:"treat_swap_as_disposition"`
	TreatCryptoFeeAsDisposition bool `json:"treat_crypto_fee_as_disposition"`
	IncludeIncomeEvents         bool `json:"include_income_events"`
	AllowLossEventsDeduction    bool `json:"allow_loss_events_deduction"`
	FailOnNegativeInventory     bool `json:"fail_on_negative_inventory"`
	FailOnMissingFiat           bool `json:"fail_on_missing_fiat"`

	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
