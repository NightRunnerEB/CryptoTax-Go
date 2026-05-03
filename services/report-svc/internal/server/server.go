package grpcserver

import (
	"context"
	"strings"

	reportv1 "github.com/NightRunner/CryptoTax-Go/gen/report/v1"
	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain"
)

type ReportServer struct {
	reportv1.UnimplementedReportServer
	ndfl domain.NDFLRenderer
}

func NewReportServer(ndfl domain.NDFLRenderer) *ReportServer {
	return &ReportServer{ndfl: ndfl}
}

func (s *ReportServer) RenderNDFL(ctx context.Context, req *reportv1.RenderNDFLRequest) (*reportv1.RenderNDFLResponse, error) {
	renderReq := mapRenderNDFLRequest(req)
	objectKey, err := s.ndfl.Render(ctx, renderReq)
	if err != nil {
		return nil, err
	}

	return &reportv1.RenderNDFLResponse{ObjectKey: objectKey}, nil
}

func mapRenderNDFLRequest(req *reportv1.RenderNDFLRequest) domain.NDFLRenderRequest {
	header := req.GetHeader()
	section1 := req.GetSection1()
	section2 := req.GetSection2()
	appendix6 := req.GetAppendix6()

	appendix2 := make([]domain.NDFLAppendix2Line, 0, len(req.GetAppendix2Lines()))
	for _, line := range req.GetAppendix2Lines() {
		mapped := domain.NDFLAppendix2Line{
			SourceCountryCode:  strings.TrimSpace(line.GetSourceCountryCode()),
			PaymentCountryCode: strings.TrimSpace(line.GetPaymentCountryCode()),
			SourceName:         strings.TrimSpace(line.GetSourceName()),
			CurrencyCode:       strings.TrimSpace(line.GetCurrencyCode()),
			IncomeTypeCode:     strings.TrimSpace(line.GetIncomeTypeCode()),
			FXRate:             strings.TrimSpace(line.GetFxRate()),
			IncomeForeign:      strings.TrimSpace(line.GetIncomeForeign()),
			IncomeRub:          strings.TrimSpace(line.GetIncomeRub()),
		}
		if ts := line.GetIncomeDate(); ts != nil {
			mapped.IncomeDate = ts.AsTime().UTC()
		}
		appendix2 = append(appendix2, mapped)
	}

	return domain.NDFLRenderRequest{
		ReportID: strings.TrimSpace(req.GetReportId()),
		UserID:   strings.TrimSpace(req.GetUserId()),
		Header: domain.NDFLHeader{
			TaxYear:          header.GetTaxYear(),
			INN:              strings.TrimSpace(header.GetInn()),
			LastName:         strings.TrimSpace(header.GetLastName()),
			FirstName:        strings.TrimSpace(header.GetFirstName()),
			MiddleName:       strings.TrimSpace(header.GetMiddleName()),
			Phone:            strings.TrimSpace(header.GetPhone()),
			OKTMO:            strings.TrimSpace(header.GetOktmo()),
			TaxResidency:     strings.TrimSpace(header.GetTaxResidency()),
			TaxPayerType:     strings.TrimSpace(header.GetTaxPayerType()),
			CorrectionNumber: strings.TrimSpace(header.GetCorrectionNumber()),
			TaxPeriodCode:    strings.TrimSpace(header.GetTaxPeriodCode()),
			TaxOfficeCode:    strings.TrimSpace(header.GetTaxOfficeCode()),
		},
		Section1: domain.NDFLSection1{
			KBK:         strings.TrimSpace(section1.GetKbk()),
			OKTMO:       strings.TrimSpace(section1.GetOktmo()),
			TaxToPay:    strings.TrimSpace(section1.GetTaxToPay()),
			TaxToRefund: strings.TrimSpace(section1.GetTaxToRefund()),
		},
		Section2: domain.NDFLSection2{
			IncomeGroupCode:           strings.TrimSpace(section2.GetIncomeGroupCode()),
			TotalIncome:               strings.TrimSpace(section2.GetTotalIncome()),
			NonTaxableIncome:          strings.TrimSpace(section2.GetNonTaxableIncome()),
			TaxableIncome:             strings.TrimSpace(section2.GetTaxableIncome()),
			Deductions:                strings.TrimSpace(section2.GetDeductions()),
			RecognizedExpenses:        strings.TrimSpace(section2.GetRecognizedExpenses()),
			TaxBase:                   strings.TrimSpace(section2.GetTaxBase()),
			CalculatedTax:             strings.TrimSpace(section2.GetCalculatedTax()),
			WithheldAtSource:          strings.TrimSpace(section2.GetWithheldAtSource()),
			MaterialBenefitTax:        strings.TrimSpace(section2.GetMaterialBenefitTax()),
			TradingFeeCredit:          strings.TrimSpace(section2.GetTradingFeeCredit()),
			FixedAdvanceCredit:        strings.TrimSpace(section2.GetFixedAdvanceCredit()),
			ForeignTaxCredit:          strings.TrimSpace(section2.GetForeignTaxCredit()),
			PatentTaxCredit:           strings.TrimSpace(section2.GetPatentTaxCredit()),
			TaxToPay:                  strings.TrimSpace(section2.GetTaxToPay()),
			TaxToRefund:               strings.TrimSpace(section2.GetTaxToRefund()),
			SimplifiedDeductionRefund: strings.TrimSpace(section2.GetSimplifiedDeductionRefund()),
		},
		Appendix2: appendix2,
		Appendix6: domain.NDFLAppendix6{
			OtherPropertyDeduction:      strings.TrimSpace(appendix6.GetOtherPropertyDeduction()),
			OtherPropertyAcquisitionExp: strings.TrimSpace(appendix6.GetOtherPropertyAcquisitionExp()),
			TotalPropertyDeduction:      strings.TrimSpace(appendix6.GetTotalPropertyDeduction()),
		},
	}
}
