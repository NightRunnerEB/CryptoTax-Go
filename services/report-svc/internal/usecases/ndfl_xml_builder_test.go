package usecase

import (
	"bytes"
	"encoding/xml"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain/error"
)

func TestBuildNDFL3XML_BasicMapping(t *testing.T) {
	req := domain.NDFLRenderRequest{
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

	got, err := buildNDFL3XML(req, "report-svc", "5.20")
	if err != nil {
		t.Fatalf("buildNDFL3XML() error = %v", err)
	}

	if len(got) == 0 {
		t.Fatal("buildNDFL3XML() returned empty payload")
	}
	if !strings.Contains(string(got), "<Файл") {
		t.Fatal("xml root <Файл> not found")
	}
	if !strings.Contains(string(got), "КНД=\"1151020\"") {
		t.Fatal("KND mapping missing")
	}
	if !strings.Contains(string(got), "ВидДоход=\"1530\"") {
		t.Fatal("appendix2 income type mapping missing")
	}
	if !strings.Contains(string(got), "РасхПриобрИмущ=\"20.00\"") {
		t.Fatal("appendix6 mapping missing")
	}
	if !strings.Contains(string(got), "ОбщИмущВыч=\"20.00\"") {
		t.Fatal("appendix6 total deduction mapping missing")
	}
	if !strings.Contains(string(got), "<ИмущНалВычПр ОбщИмущВыч=\"20.00\">") {
		t.Fatal("appendix6 total deduction must be on ИмущНалВычПр")
	}
	if strings.Contains(string(got), "<ВычПродИмущИн ОбщИмущВыч=") {
		t.Fatal("appendix6 total deduction must not be on ВычПродИмущИн")
	}

	var parsed struct {
		XMLName xml.Name
	}
	if err := xml.NewDecoder(bytes.NewReader(got)).Decode(&parsed); err != nil {
		t.Fatalf("generated xml is invalid: %v", err)
	}
}

func TestBuildNDFL3XML_InvalidINN(t *testing.T) {
	req := domain.NDFLRenderRequest{
		ReportID: "report-1",
		UserID:   "user-1",
		Header: domain.NDFLHeader{
			TaxYear:          2025,
			INN:              "",
			LastName:         "Petrov",
			FirstName:        "Ivan",
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
		Appendix6: domain.NDFLAppendix6{
			OtherPropertyDeduction:      "0",
			OtherPropertyAcquisitionExp: "20.00",
			TotalPropertyDeduction:      "20.00",
		},
	}

	_, err := buildNDFL3XML(req, "report-svc", "5.20")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.ErrInternal {
		t.Fatalf("expected INTERNAL_ERROR, got %v", err)
	}
}

func TestBuildNDFL3XML_Appendix2RequiresIncomeDate(t *testing.T) {
	req := domain.NDFLRenderRequest{
		ReportID: "report-1",
		UserID:   "user-1",
		Header: domain.NDFLHeader{
			TaxYear:          2025,
			INN:              "123456789047",
			LastName:         "Petrov",
			FirstName:        "Ivan",
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
				IncomeDate:         time.Time{},
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

	_, err := buildNDFL3XML(req, "report-svc", "5.20")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != apperr.ErrInternal {
		t.Fatalf("expected INTERNAL_ERROR, got %v", err)
	}
}
