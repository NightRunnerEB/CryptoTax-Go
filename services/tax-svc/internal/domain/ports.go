package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AggregatedTxProvider interface {
	ListTransactionsByRange(ctx context.Context, userID uuid.UUID, fromUTC, toUTC time.Time, targetFiat string) ([]AggregatedTransaction, error)
}

type ObjectStorage interface {
	UploadJSON(ctx context.Context, objectKey string, payload any) error
	PresignGet(ctx context.Context, objectKey string) (string, error)
}

type ReportRenderRequest struct {
	ReportID     uuid.UUID
	UserID       uuid.UUID
	Jurisdiction string
	NDFL         NDFLReportPayload
}

type ReportClient interface {
	RequestRender(ctx context.Context, req ReportRenderRequest) (string, error)
}

type NDFLReportPayload struct {
	Header    NDFLHeader
	Section1  NDFLSection1
	Section2  NDFLSection2
	Appendix2 []NDFLAppendix2Line
	Appendix6 NDFLAppendix6
}

type NDFLHeader struct {
	TaxYear int

	INN        string
	LastName   string
	FirstName  string
	MiddleName string
	Phone      string
	OKTMO      string

	TaxResidency string
	TaxPayerType string

	CorrectionNumber string
	TaxPeriodCode    string
	TaxOfficeCode    string
}

type NDFLSection1 struct {
	KBK         string
	OKTMO       string
	TaxToPay    decimal.Decimal
	TaxToRefund decimal.Decimal
}

type NDFLSection2 struct {
	IncomeGroupCode string

	TotalIncome        decimal.Decimal
	NonTaxableIncome   decimal.Decimal
	TaxableIncome      decimal.Decimal
	Deductions         decimal.Decimal
	RecognizedExpenses decimal.Decimal
	TaxBase            decimal.Decimal

	CalculatedTax      decimal.Decimal
	WithheldAtSource   decimal.Decimal
	MaterialBenefitTax decimal.Decimal
	TradingFeeCredit   decimal.Decimal
	FixedAdvanceCredit decimal.Decimal
	ForeignTaxCredit   decimal.Decimal
	PatentTaxCredit    decimal.Decimal

	TaxToPay    decimal.Decimal
	TaxToRefund decimal.Decimal

	SimplifiedDeductionRefund decimal.Decimal
}

type NDFLAppendix2Line struct {
	SourceCountryCode  string
	PaymentCountryCode string
	SourceName         string
	CurrencyCode       string
	IncomeTypeCode     string
	IncomeDate         time.Time
	FXRate             decimal.Decimal
	IncomeForeign      decimal.Decimal
	IncomeRub          decimal.Decimal
}

type NDFLAppendix6 struct {
	OtherPropertyDeduction      decimal.Decimal
	OtherPropertyAcquisitionExp decimal.Decimal
	TotalPropertyDeduction      decimal.Decimal
}
