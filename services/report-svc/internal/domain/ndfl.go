package domain

import "time"

type NDFLRenderRequest struct {
	ReportID string
	UserID   string

	Header    NDFLHeader
	Section1  NDFLSection1
	Section2  NDFLSection2
	Appendix2 []NDFLAppendix2Line
	Appendix6 NDFLAppendix6
}

type NDFLHeader struct {
	TaxYear int32

	INN        string
	LastName   string
	FirstName  string
	MiddleName string

	Phone string
	OKTMO string

	TaxResidency string
	TaxPayerType string

	CorrectionNumber string
	TaxPeriodCode    string
	TaxOfficeCode    string
}

type NDFLSection1 struct {
	KBK         string
	OKTMO       string
	TaxToPay    string
	TaxToRefund string
}

type NDFLSection2 struct {
	IncomeGroupCode string

	TotalIncome        string
	NonTaxableIncome   string
	TaxableIncome      string
	Deductions         string
	RecognizedExpenses string
	TaxBase            string

	CalculatedTax      string
	WithheldAtSource   string
	MaterialBenefitTax string
	TradingFeeCredit   string
	FixedAdvanceCredit string
	ForeignTaxCredit   string
	PatentTaxCredit    string

	TaxToPay    string
	TaxToRefund string

	SimplifiedDeductionRefund string
}

type NDFLAppendix2Line struct {
	SourceCountryCode  string
	PaymentCountryCode string
	SourceName         string

	CurrencyCode   string
	IncomeTypeCode string

	IncomeDate time.Time
	FXRate     string

	IncomeForeign string
	IncomeRub     string
}

type NDFLAppendix6 struct {
	OtherPropertyDeduction      string
	OtherPropertyAcquisitionExp string
	TotalPropertyDeduction      string
}
