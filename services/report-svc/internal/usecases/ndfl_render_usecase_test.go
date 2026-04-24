package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain"
)

func TestNDFLRenderUC_Render_Success(t *testing.T) {
	storage := &testObjectStorage{}
	validator := &testNDFLValidator{}
	uc := NewNDFLRenderUCWithValidator(storage, "report-svc", validator)

	objectKey, err := uc.Render(context.Background(), validNDFLRenderRequest())
	if err != nil {
		t.Fatalf("Render() unexpected error: %v", err)
	}
	if objectKey == "" {
		t.Fatal("Render() returned empty object key")
	}
	if storage.uploadCount != 1 {
		t.Fatalf("UploadXML call count mismatch: got %d want %d", storage.uploadCount, 1)
	}
	if validator.callCount != 1 {
		t.Fatalf("validator call count mismatch: got %d want %d", validator.callCount, 1)
	}
}

func TestNDFLRenderUC_Render_ValidationFailed(t *testing.T) {
	storage := &testObjectStorage{}
	validator := &testNDFLValidator{
		err: errors.New("schema mismatch"),
	}
	uc := NewNDFLRenderUCWithValidator(storage, "report-svc", validator)

	_, err := uc.Render(context.Background(), validNDFLRenderRequest())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if storage.uploadCount != 0 {
		t.Fatalf("UploadXML must not be called on validation failure, got %d", storage.uploadCount)
	}
}

type testNDFLValidator struct {
	callCount int
	err       error
}

func (v *testNDFLValidator) Validate(_ context.Context, _ []byte) error {
	v.callCount++
	return v.err
}

type testObjectStorage struct {
	uploadCount int
}

func (s *testObjectStorage) UploadXML(_ context.Context, _ string, _ []byte) error {
	s.uploadCount++
	return nil
}

func validNDFLRenderRequest() domain.NDFLRenderRequest {
	return domain.NDFLRenderRequest{
		ReportID: "report-1",
		UserID:   "user-1",
		Header: domain.NDFLHeader{
			TaxYear:          2025,
			INN:              "123456789047",
			LastName:         "Petrov",
			FirstName:        "Ivan",
			MiddleName:       "Ivanovich",
			Phone:            "+79990000000",
			OKTMO:            "12345678",
			TaxResidency:     "RESIDENT",
			TaxPayerType:     "INDIVIDUAL",
			CorrectionNumber: "0",
			TaxPeriodCode:    "34",
			TaxOfficeCode:    "1234",
		},
		Section1: domain.NDFLSection1{
			KBK:         "18210102010011000110",
			OKTMO:       "12345678",
			TaxToPay:    "100",
			TaxToRefund: "0",
		},
		Section2: domain.NDFLSection2{
			IncomeGroupCode:           "13",
			TotalIncome:               "100.00",
			NonTaxableIncome:          "0",
			TaxableIncome:             "100.00",
			Deductions:                "0",
			RecognizedExpenses:        "20.00",
			TaxBase:                   "80.00",
			CalculatedTax:             "10",
			WithheldAtSource:          "0",
			MaterialBenefitTax:        "0",
			TradingFeeCredit:          "0",
			FixedAdvanceCredit:        "0",
			ForeignTaxCredit:          "0",
			PatentTaxCredit:           "0",
			TaxToPay:                  "10",
			TaxToRefund:               "0",
			SimplifiedDeductionRefund: "0",
		},
		Appendix2: []domain.NDFLAppendix2Line{
			{
				SourceCountryCode:  "999",
				PaymentCountryCode: "643",
				SourceName:         "CRYPTO",
				CurrencyCode:       "643",
				IncomeTypeCode:     "1530",
				IncomeDate:         time.Date(2025, time.January, 2, 0, 0, 0, 0, time.UTC),
				FXRate:             "1",
				IncomeForeign:      "30.00",
				IncomeRub:          "30.00",
			},
		},
		Appendix6: domain.NDFLAppendix6{
			OtherPropertyDeduction:      "0",
			OtherPropertyAcquisitionExp: "20.00",
			TotalPropertyDeduction:      "20.00",
		},
	}
}
