package report

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	reportv1 "github.com/NightRunner/CryptoTax-Go/gen/report/v1"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
)

type Client struct {
	addr    string
	timeout time.Duration
	conn    *grpc.ClientConn
	client  reportv1.ReportClient
}

func NewClient(ctx context.Context, addr string, timeout time.Duration) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, apperr.Internal("report grpc dial failed", err, map[string]string{
			"addr": addr,
		})
	}

	return &Client{
		addr:    addr,
		timeout: timeout,
		conn:    conn,
		client:  reportv1.NewReportClient(conn),
	}, nil
}

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) RequestRender(ctx context.Context, req domain.ReportRenderRequest) (string, error) {
	callCtx := ctx
	var cancel context.CancelFunc
	if c.timeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	pbReq := &reportv1.RenderNDFLRequest{
		ReportId: req.ReportID.String(),
		UserId:   req.UserID.String(),
		Header: &reportv1.NdflHeader{
			TaxYear:          int32(req.NDFL.Header.TaxYear),
			Inn:              req.NDFL.Header.INN,
			LastName:         req.NDFL.Header.LastName,
			FirstName:        req.NDFL.Header.FirstName,
			MiddleName:       req.NDFL.Header.MiddleName,
			Phone:            req.NDFL.Header.Phone,
			Oktmo:            req.NDFL.Header.OKTMO,
			TaxResidency:     req.NDFL.Header.TaxResidency,
			TaxPayerType:     req.NDFL.Header.TaxPayerType,
			CorrectionNumber: req.NDFL.Header.CorrectionNumber,
			TaxPeriodCode:    req.NDFL.Header.TaxPeriodCode,
			TaxOfficeCode:    req.NDFL.Header.TaxOfficeCode,
		},
		Section1: &reportv1.NdflSection1{
			Kbk:         req.NDFL.Section1.KBK,
			Oktmo:       req.NDFL.Section1.OKTMO,
			TaxToPay:    req.NDFL.Section1.TaxToPay.String(),
			TaxToRefund: req.NDFL.Section1.TaxToRefund.String(),
		},
		Section2: &reportv1.NdflSection2{
			IncomeGroupCode:           req.NDFL.Section2.IncomeGroupCode,
			TotalIncome:               req.NDFL.Section2.TotalIncome.String(),
			NonTaxableIncome:          req.NDFL.Section2.NonTaxableIncome.String(),
			TaxableIncome:             req.NDFL.Section2.TaxableIncome.String(),
			Deductions:                req.NDFL.Section2.Deductions.String(),
			RecognizedExpenses:        req.NDFL.Section2.RecognizedExpenses.String(),
			TaxBase:                   req.NDFL.Section2.TaxBase.String(),
			CalculatedTax:             req.NDFL.Section2.CalculatedTax.String(),
			WithheldAtSource:          req.NDFL.Section2.WithheldAtSource.String(),
			MaterialBenefitTax:        req.NDFL.Section2.MaterialBenefitTax.String(),
			TradingFeeCredit:          req.NDFL.Section2.TradingFeeCredit.String(),
			FixedAdvanceCredit:        req.NDFL.Section2.FixedAdvanceCredit.String(),
			ForeignTaxCredit:          req.NDFL.Section2.ForeignTaxCredit.String(),
			PatentTaxCredit:           req.NDFL.Section2.PatentTaxCredit.String(),
			TaxToPay:                  req.NDFL.Section2.TaxToPay.String(),
			TaxToRefund:               req.NDFL.Section2.TaxToRefund.String(),
			SimplifiedDeductionRefund: req.NDFL.Section2.SimplifiedDeductionRefund.String(),
		},
		Appendix6: &reportv1.NdflAppendix6{
			OtherPropertyDeduction:      req.NDFL.Appendix6.OtherPropertyDeduction.String(),
			OtherPropertyAcquisitionExp: req.NDFL.Appendix6.OtherPropertyAcquisitionExp.String(),
			TotalPropertyDeduction:      req.NDFL.Appendix6.TotalPropertyDeduction.String(),
		},
	}

	pbReq.Appendix2Lines = make([]*reportv1.NdflAppendix2Line, 0, len(req.NDFL.Appendix2))
	for _, line := range req.NDFL.Appendix2 {
		item := &reportv1.NdflAppendix2Line{
			SourceCountryCode:  line.SourceCountryCode,
			PaymentCountryCode: line.PaymentCountryCode,
			SourceName:         line.SourceName,
			CurrencyCode:       line.CurrencyCode,
			IncomeTypeCode:     line.IncomeTypeCode,
			FxRate:             line.FXRate.String(),
			IncomeForeign:      line.IncomeForeign.String(),
			IncomeRub:          line.IncomeRub.String(),
		}
		if !line.IncomeDate.IsZero() {
			item.IncomeDate = timestamppb.New(line.IncomeDate)
		}
		pbReq.Appendix2Lines = append(pbReq.Appendix2Lines, item)
	}

	resp, err := c.client.RenderNDFL(callCtx, pbReq)
	if err != nil {
		return "", apperr.Internal("render ndfl request failed", err, map[string]string{
			"addr":      c.addr,
			"report_id": req.ReportID.String(),
		})
	}

	return resp.GetObjectKey(), nil
}

var _ domain.ReportClient = (*Client)(nil)
