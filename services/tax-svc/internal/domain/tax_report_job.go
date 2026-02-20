package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ReportJobStatus string

const (
	ReportJobStatusQueued     ReportJobStatus = "queued"
	ReportJobStatusProcessing ReportJobStatus = "processing"
	ReportJobStatusCompleted  ReportJobStatus = "completed"
	ReportJobStatusFailed     ReportJobStatus = "failed"
)

type StartReportParams struct {
	TenantID                         uuid.UUID `json:"tenant_id"`
	TaxYear                          int32     `json:"tax_year"`
	Jurisdiction                     string    `json:"jurisdiction"`
	Timezone                         string    `json:"timezone"`
	CostBasisMethod                  string    `json:"cost_basis_method"`
	TreatCryptoToCryptoAsDisposition bool      `json:"treat_crypto_to_crypto_as_disposition"`
}

type TaxReportJob struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	TaxYear          int32           `json:"tax_year"`
	Jurisdiction     string          `json:"jurisdiction"`
	Status           ReportJobStatus `json:"status"`
	RequestedAt      time.Time       `json:"requested_at"`
	StartedAt        *time.Time      `json:"started_at,omitempty"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	Error            *string         `json:"error,omitempty"`
	Params           json.RawMessage `json:"params"`
	Summary          json.RawMessage `json:"summary,omitempty"`
	DatasetObjectKey *string         `json:"dataset_object_key,omitempty"`
	PDFObjectKey     *string         `json:"pdf_object_key,omitempty"`
}

type TaxReportJobPage struct {
	Reports []TaxReportJob `json:"reports"`
	Total   int64          `json:"total"`
}

type ReportStatusView struct {
	Job         TaxReportJob `json:"job"`
	DownloadURL *string      `json:"download_url,omitempty"`
}
