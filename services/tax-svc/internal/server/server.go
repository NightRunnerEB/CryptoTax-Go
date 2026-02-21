package grpcserver

import (
	"context"
	"strings"
	"time"

	taxv1 "github.com/NightRunner/CryptoTax-Go/gen/tax/v1"
	"github.com/NightRunner/CryptoTax-Go/pkg/logger"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TaxServer struct {
	taxv1.UnimplementedTaxServer

	taxProfileUC      domain.TaxProfileUseCase
	taxpayerProfileUC domain.TaxpayerProfileUseCase
	reportUC          domain.ReportUseCase
}

func NewTaxServer(
	taxProfileUC domain.TaxProfileUseCase,
	taxpayerProfileUC domain.TaxpayerProfileUseCase,
	reportUC domain.ReportUseCase,
) *TaxServer {
	return &TaxServer{
		taxProfileUC:      taxProfileUC,
		taxpayerProfileUC: taxpayerProfileUC,
		reportUC:          reportUC,
	}
}

func (s *TaxServer) GetTaxProfile(ctx context.Context, req *taxv1.GetTaxProfileRequest) (*taxv1.GetTaxProfileResponse, error) {
	if req == nil {
		return nil, invalidRequest()
	}
	tenantID, err := parseUUID(req.GetTenantId())
	if err != nil {
		return nil, invalidField("tenant_id", err)
	}
	if err := requireTenantMatchIfPresent(ctx, tenantID); err != nil {
		return nil, err
	}

	profile, err := s.taxProfileUC.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &taxv1.GetTaxProfileResponse{
		Profile: mapTaxProfile(profile),
	}, nil
}

func (s *TaxServer) UpsertTaxProfile(ctx context.Context, req *taxv1.UpsertTaxProfileRequest) (*taxv1.UpsertTaxProfileResponse, error) {
	if req == nil {
		return nil, invalidRequest()
	}
	tenantID, err := parseUUID(req.GetTenantId())
	if err != nil {
		return nil, invalidField("tenant_id", err)
	}
	if err := requireTenantMatchIfPresent(ctx, tenantID); err != nil {
		return nil, err
	}

	current, err := s.taxProfileUC.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	upsertInput := current
	upsertInput.TenantID = tenantID
	if v := strings.TrimSpace(req.GetJurisdiction()); v != "" {
		upsertInput.Jurisdiction = v
	}
	if v := strings.TrimSpace(req.GetCostBasisMethod()); v != "" {
		upsertInput.CostBasisMethod = v
	}
	if v := strings.TrimSpace(req.GetTimezone()); v != "" {
		upsertInput.Timezone = v
	}
	if req.TreatSwapAsDisposition != nil {
		upsertInput.TreatSwapAsDisposition = req.GetTreatSwapAsDisposition()
	}
	if req.TreatCryptoFeeAsDisposition != nil {
		upsertInput.TreatCryptoFeeAsDisposition = req.GetTreatCryptoFeeAsDisposition()
	}
	if req.IncludeIncomeEvents != nil {
		upsertInput.IncludeIncomeEvents = req.GetIncludeIncomeEvents()
	}
	if req.AllowLossEventsDeduction != nil {
		upsertInput.AllowLossEventsDeduction = req.GetAllowLossEventsDeduction()
	}
	if req.FailOnNegativeInventory != nil {
		upsertInput.FailOnNegativeInventory = req.GetFailOnNegativeInventory()
	}
	if req.FailOnMissingFiat != nil {
		upsertInput.FailOnMissingFiat = req.GetFailOnMissingFiat()
	}

	profile, err := s.taxProfileUC.Upsert(ctx, upsertInput)
	if err != nil {
		return nil, err
	}

	return &taxv1.UpsertTaxProfileResponse{
		Profile: mapTaxProfile(profile),
	}, nil
}

func (s *TaxServer) GetTaxpayerProfile(ctx context.Context, req *taxv1.GetTaxpayerProfileRequest) (*taxv1.GetTaxpayerProfileResponse, error) {
	if req == nil {
		return nil, invalidRequest()
	}
	tenantID, err := parseUUID(req.GetTenantId())
	if err != nil {
		return nil, invalidField("tenant_id", err)
	}
	if err := requireTenantMatchIfPresent(ctx, tenantID); err != nil {
		return nil, err
	}

	profile, err := s.taxpayerProfileUC.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &taxv1.GetTaxpayerProfileResponse{
		Profile: mapTaxpayerProfile(profile),
	}, nil
}

func (s *TaxServer) UpsertTaxpayerProfile(ctx context.Context, req *taxv1.UpsertTaxpayerProfileRequest) (*taxv1.UpsertTaxpayerProfileResponse, error) {
	if req == nil {
		return nil, invalidRequest()
	}
	tenantID, err := parseUUID(req.GetTenantId())
	if err != nil {
		return nil, invalidField("tenant_id", err)
	}
	if err := requireTenantMatchIfPresent(ctx, tenantID); err != nil {
		return nil, err
	}

	var birthDate *time.Time
	if v := strings.TrimSpace(req.GetBirthDate()); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return nil, invalidField("birth_date", err)
		}
		birthDate = &t
	}

	profile, err := s.taxpayerProfileUC.Upsert(ctx, domain.TaxpayerProfile{
		TenantID:           tenantID,
		INN:                nilIfEmpty(req.GetInn()),
		LastName:           nilIfEmpty(req.GetLastName()),
		FirstName:          nilIfEmpty(req.GetFirstName()),
		MiddleName:         nilIfEmpty(req.GetMiddleName()),
		BirthDate:          birthDate,
		DocumentTypeCode:   nilIfEmpty(req.GetDocumentTypeCode()),
		DocumentNumber:     nilIfEmpty(req.GetDocumentNumber()),
		TaxResidencyStatus: nilIfEmpty(req.GetTaxResidencyStatus()),
		Phone:              nilIfEmpty(req.GetPhone()),
	})
	if err != nil {
		return nil, err
	}

	return &taxv1.UpsertTaxpayerProfileResponse{
		Profile: mapTaxpayerProfile(profile),
	}, nil
}

func (s *TaxServer) StartReport(ctx context.Context, req *taxv1.StartReportRequest) (*taxv1.StartReportResponse, error) {
	log := logger.FromContext(ctx)
	if req == nil {
		return nil, invalidRequest()
	}
	tenantID, err := parseUUID(req.GetTenantId())
	if err != nil {
		return nil, invalidField("tenant_id", err)
	}
	if err := requireTenantMatchIfPresent(ctx, tenantID); err != nil {
		return nil, err
	}

	job, err := s.reportUC.StartReport(ctx, domain.StartReportParams{
		TenantID:                         tenantID,
		TaxYear:                          req.GetTaxYear(),
		Jurisdiction:                     req.GetJurisdiction(),
		Timezone:                         req.GetTimezone(),
		CostBasisMethod:                  req.GetCostBasisMethod(),
		TreatCryptoToCryptoAsDisposition: req.GetTreatCryptoToCryptoAsDisposition(),
	})
	if err != nil {
		return nil, err
	}

	log.Info("StartReport: queued", zap.String("report_id", job.ID.String()), zap.String("tenant_id", tenantID.String()))
	_ = grpc.SetHeader(ctx, metadata.Pairs("x-http-code", "202"))
	return &taxv1.StartReportResponse{
		ReportId: job.ID.String(),
	}, nil
}

func (s *TaxServer) GetReportStatus(ctx context.Context, req *taxv1.GetReportStatusRequest) (*taxv1.GetReportStatusResponse, error) {
	if req == nil {
		return nil, invalidRequest()
	}
	tenantID, err := parseUUID(req.GetTenantId())
	if err != nil {
		return nil, invalidField("tenant_id", err)
	}
	if err := requireTenantMatchIfPresent(ctx, tenantID); err != nil {
		return nil, err
	}
	reportID, err := parseUUID(req.GetReportId())
	if err != nil {
		return nil, invalidField("report_id", err)
	}

	view, err := s.reportUC.GetReportStatus(ctx, tenantID, reportID)
	if err != nil {
		return nil, err
	}

	resp := &taxv1.GetReportStatusResponse{
		ReportId:     view.Job.ID.String(),
		Status:       string(view.Job.Status),
		Error:        valueOrEmpty(view.Job.Error),
		RequestedAt:  timestamppb.New(view.Job.RequestedAt),
		StartedAt:    toTimestamp(view.Job.StartedAt),
		CompletedAt:  toTimestamp(view.Job.CompletedAt),
		PdfObjectKey: valueOrEmpty(view.Job.PDFObjectKey),
		DownloadUrl:  valueOrEmpty(view.DownloadURL),
	}
	return resp, nil
}

func (s *TaxServer) ListReports(ctx context.Context, req *taxv1.ListReportsRequest) (*taxv1.ListReportsResponse, error) {
	if req == nil {
		return nil, invalidRequest()
	}
	tenantID, err := parseUUID(req.GetTenantId())
	if err != nil {
		return nil, invalidField("tenant_id", err)
	}
	if err := requireTenantMatchIfPresent(ctx, tenantID); err != nil {
		return nil, err
	}

	page, err := s.reportUC.ListReports(ctx, tenantID, req.GetTaxYear(), req.GetLimit(), req.GetOffset())
	if err != nil {
		return nil, err
	}

	items := make([]*taxv1.ReportListItem, 0, len(page.Reports))
	for _, job := range page.Reports {
		items = append(items, &taxv1.ReportListItem{
			ReportId:     job.ID.String(),
			TaxYear:      job.TaxYear,
			Jurisdiction: job.Jurisdiction,
			Status:       string(job.Status),
			Error:        valueOrEmpty(job.Error),
			RequestedAt:  timestamppb.New(job.RequestedAt),
			StartedAt:    toTimestamp(job.StartedAt),
			CompletedAt:  toTimestamp(job.CompletedAt),
			PdfObjectKey: valueOrEmpty(job.PDFObjectKey),
		})
	}

	return &taxv1.ListReportsResponse{
		Reports: items,
		Total:   page.Total,
	}, nil
}

func mapTaxProfile(p domain.TaxProfile) *taxv1.TaxProfile {
	return &taxv1.TaxProfile{
		TenantId:                    p.TenantID.String(),
		Jurisdiction:                p.Jurisdiction,
		CostBasisMethod:             p.CostBasisMethod,
		Timezone:                    p.Timezone,
		TreatSwapAsDisposition:      p.TreatSwapAsDisposition,
		TreatCryptoFeeAsDisposition: p.TreatCryptoFeeAsDisposition,
		IncludeIncomeEvents:         p.IncludeIncomeEvents,
		AllowLossEventsDeduction:    p.AllowLossEventsDeduction,
		FailOnNegativeInventory:     p.FailOnNegativeInventory,
		FailOnMissingFiat:           p.FailOnMissingFiat,
	}
}

func mapTaxpayerProfile(p domain.TaxpayerProfile) *taxv1.TaxpayerProfile {
	out := &taxv1.TaxpayerProfile{
		TenantId:           p.TenantID.String(),
		Inn:                valueOrEmpty(p.INN),
		LastName:           valueOrEmpty(p.LastName),
		FirstName:          valueOrEmpty(p.FirstName),
		MiddleName:         valueOrEmpty(p.MiddleName),
		DocumentTypeCode:   valueOrEmpty(p.DocumentTypeCode),
		DocumentNumber:     valueOrEmpty(p.DocumentNumber),
		TaxResidencyStatus: valueOrEmpty(p.TaxResidencyStatus),
		Phone:              valueOrEmpty(p.Phone),
	}
	if p.BirthDate != nil {
		out.BirthDate = p.BirthDate.Format("2006-01-02")
	}
	return out
}

func parseUUID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func invalidRequest() error {
	return apperr.InvalidArgument("invalid request", nil, apperr.FieldViolation{
		Field:       "request",
		Description: "required",
	})
}

func invalidField(field string, cause error) error {
	return apperr.InvalidArgument("invalid "+field, cause, apperr.FieldViolation{
		Field:       field,
		Description: "invalid format",
	})
}

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func nilIfEmpty(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	out := v
	return &out
}

func toTimestamp(v *time.Time) *timestamppb.Timestamp {
	if v == nil {
		return nil
	}
	return timestamppb.New(*v)
}
