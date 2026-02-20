package domain

import (
	"encoding/json"

	"github.com/google/uuid"
)

const (
	EventTypeTaxReportJobRequested = "TaxReportJobRequested"
	EventTypeReportRenderRequested = "ReportRenderRequested"
	EventTypeReportRendered        = "ReportRendered"
	EventTypeReportRenderFailed    = "ReportRenderFailed"
)

type BrokerEnvelope struct {
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
}

type TaxReportJobRequestedEvent struct {
	EventID  uuid.UUID `json:"event_id"`
	ReportID uuid.UUID `json:"report_id"`
	TenantID uuid.UUID `json:"tenant_id"`
}

type ReportRenderRequestedEvent struct {
	EventID          uuid.UUID `json:"event_id"`
	ReportID         uuid.UUID `json:"report_id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	Jurisdiction     string    `json:"jurisdiction"`
	TaxYear          int32     `json:"tax_year"`
	DatasetObjectKey string    `json:"dataset_object_key"`
	TemplateVersion  string    `json:"template_version,omitempty"`
}

type ReportRenderedEvent struct {
	EventID      uuid.UUID `json:"event_id"`
	ReportID     uuid.UUID `json:"report_id"`
	PDFObjectKey string    `json:"pdf_object_key"`
}

type ReportRenderFailedEvent struct {
	EventID  uuid.UUID `json:"event_id"`
	ReportID uuid.UUID `json:"report_id"`
	Error    string    `json:"error"`
}
