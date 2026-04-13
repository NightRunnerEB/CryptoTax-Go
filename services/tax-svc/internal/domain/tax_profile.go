package domain

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type TaxResidency string

const (
	Resident    TaxResidency = "RESIDENT"
	NonResident TaxResidency = "NON_RESIDENT"
)

func (r TaxResidency) Validate() error {
	switch r {
	case Resident, NonResident:
		return nil
	default:
		return fmt.Errorf("unsupported tax residency status: %s", r)
	}
}

type TaxPayerType string

const (
	INDIVIDUAL      TaxPayerType = "INDIVIDUAL"
	SOLE_PROPRIETOR TaxPayerType = "SOLE_PROPRIETOR"
	LEGAL_ENTITY    TaxPayerType = "LEGAL_ENTITY"
)

func (t TaxPayerType) Validate() error {
	switch t {
	case INDIVIDUAL, SOLE_PROPRIETOR, LEGAL_ENTITY:
		return nil
	default:
		return fmt.Errorf("unsupported taxpayer type: %s", t)
	}
}

type Wallet string

type TaxProfile struct {
	UserID             uuid.UUID    `json:"user_id"`
	INN                string       `json:"inn"`
	LastName           string       `json:"last_name"`
	FirstName          string       `json:"first_name"`
	MiddleName         string       `json:"middle_name"`
	Timezone           string       `json:"timezone"` // IANA timezone
	Phone              string       `json:"phone"`
	Wallets            []Wallet     `json:"wallets"`
	TaxResidencyStatus TaxResidency `json:"tax_residency_status"`
	TaxPayerType       TaxPayerType `json:"taxpayer_type"`
}

type TaxProfileUseCase interface {
	Upsert(ctx context.Context, p TaxProfile) error
	Get(ctx context.Context, userID uuid.UUID) (TaxProfile, error)
	Delete(ctx context.Context, userID uuid.UUID) error
}

type TaxProfileRepo interface {
	Upsert(ctx context.Context, p TaxProfile) error
	Get(ctx context.Context, userID uuid.UUID) (TaxProfile, error)
	Delete(ctx context.Context, userID uuid.UUID) error
}
