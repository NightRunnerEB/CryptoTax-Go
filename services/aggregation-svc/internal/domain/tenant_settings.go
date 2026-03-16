package domain

import (
	"time"

	"github.com/google/uuid"
)

type TenantSettings struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	FiatCurrency string    `json:"fiat_currency"`
	Timezone     string    `json:"timezone"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type SupportedFiatCurrency struct {
	Code        string `json:"code"`
	DisplayName string `json:"display_name"`
}
