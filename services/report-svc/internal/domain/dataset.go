package domain

import "time"

type DatasetEvent struct {
	EventType     string            `json:"event_type"`
	TxID          string            `json:"tx_id"`
	TimeUTC       time.Time         `json:"time_utc"`
	Source        string            `json:"source"`
	Kind          string            `json:"kind"`
	TxFingerprint string            `json:"tx_fingerprint"`
	AssetSymbol   string            `json:"asset_symbol,omitempty"`
	CryptoAmount  string            `json:"crypto_amount,omitempty"`
	FiatAmount    *string           `json:"fiat_amount,omitempty"`
	FeeFiatAmount *string           `json:"fee_fiat_amount,omitempty"`
	Meta          map[string]string `json:"meta,omitempty"`
}

type RealizationLine struct {
	DisposalTxID      string    `json:"disposal_tx_id"`
	AssetSymbol       string    `json:"asset_symbol"`
	QtyDisposed       string    `json:"qty_disposed"`
	ProceedsFiatAlloc string    `json:"proceeds_fiat_alloc"`
	CostFiatAlloc     string    `json:"cost_fiat_alloc"`
	FeesFiatAlloc     string    `json:"fees_fiat_alloc"`
	GainFiatAlloc     string    `json:"gain_fiat_alloc"`
	LotAcquiredAt     time.Time `json:"lot_acquired_at"`
	LotSourceTxID     string    `json:"lot_source_tx_id"`
}

type TaxProfile struct {
	UserID          string `json:"user_id"`
	Jurisdiction    string `json:"jurisdiction"`
	CostBasisMethod string `json:"cost_basis_method"`
	Timezone        string `json:"timezone"`
}

type TaxpayerProfile struct {
	UserID             string `json:"user_id"`
	INN                string `json:"inn,omitempty"`
	LastName           string `json:"last_name,omitempty"`
	FirstName          string `json:"first_name,omitempty"`
	MiddleName         string `json:"middle_name,omitempty"`
	BirthDate          string `json:"birth_date,omitempty"`
	DocumentTypeCode   string `json:"document_type_code,omitempty"`
	DocumentNumber     string `json:"document_number,omitempty"`
	TaxResidencyStatus string `json:"tax_residency_status,omitempty"`
	Phone              string `json:"phone,omitempty"`
}

type ReportDataset struct {
	ReportID         string            `json:"report_id"`
	UserID           string            `json:"user_id"`
	TaxYear          int32             `json:"tax_year"`
	Jurisdiction     string            `json:"jurisdiction"`
	TaxpayerProfile  TaxpayerProfile   `json:"taxpayer_profile"`
	TaxProfile       TaxProfile        `json:"tax_profile"`
	Summary          map[string]any    `json:"summary"`
	Events           []DatasetEvent    `json:"events"`
	RealizationLines []RealizationLine `json:"realization_lines"`
}
