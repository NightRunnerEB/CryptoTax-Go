package domain

import (
	"time"

	"github.com/google/uuid"
)

type TaxpayerProfile struct {
	TenantID           uuid.UUID  `json:"tenant_id"`
	INN                *string    `json:"inn,omitempty"`
	LastName           *string    `json:"last_name,omitempty"`
	FirstName          *string    `json:"first_name,omitempty"`
	MiddleName         *string    `json:"middle_name,omitempty"`
	BirthDate          *time.Time `json:"birth_date,omitempty"`
	DocumentTypeCode   *string    `json:"document_type_code,omitempty"`
	DocumentNumber     *string    `json:"document_number,omitempty"`
	TaxResidencyStatus *string    `json:"tax_residency_status,omitempty"`
	Phone              *string    `json:"phone,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}
