package grpcserver

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	taxv1 "github.com/NightRunner/CryptoTax-Go/gen/tax/v1"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
)

type TaxServer struct {
	taxv1.UnimplementedTaxServer
	taxProfileUC domain.TaxProfileUseCase
	taxJobUC     domain.TaxJobUseCase
}

func NewTaxServer(taxProfileUC domain.TaxProfileUseCase, taxJobUC domain.TaxJobUseCase) *TaxServer {
	return &TaxServer{
		taxProfileUC: taxProfileUC,
		taxJobUC:     taxJobUC,
	}
}

func (s *TaxServer) GetTaxProfile(ctx context.Context, req *taxv1.GetTaxProfileRequest) (*taxv1.GetTaxProfileResponse, error) {
	if req == nil {
		return nil, apperr.InvalidArgument("invalid request", nil, apperr.FieldViolation{
			Field:       "request",
			Description: "required",
		})
	}

	tenantID, err := parseUUID(req.GetTenantId())
	if err != nil {
		return nil, apperr.InvalidArgument("invalid tenant_id", err, apperr.FieldViolation{
			Field:       "tenant_id",
			Description: "invalid uuid",
		})
	}
	if err := requireTenantMatchIfPresent(ctx, tenantID); err != nil {
		return nil, err
	}

	profile, err := s.taxProfileUC.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &taxv1.GetTaxProfileResponse{
		Profile: toProtoTaxProfile(profile),
	}, nil
}

func (s *TaxServer) UpsertTaxProfile(ctx context.Context, req *taxv1.UpsertTaxProfileRequest) (*taxv1.UpsertTaxProfileResponse, error) {
	if req == nil {
		return nil, apperr.InvalidArgument("invalid request", nil, apperr.FieldViolation{
			Field:       "request",
			Description: "required",
		})
	}

	tenantID, err := parseUUID(req.GetTenantId())
	if err != nil {
		return nil, apperr.InvalidArgument("invalid tenant_id", err, apperr.FieldViolation{
			Field:       "tenant_id",
			Description: "invalid uuid",
		})
	}
	if err := requireTenantMatchIfPresent(ctx, tenantID); err != nil {
		return nil, err
	}
	if req.GetProfile() == nil {
		return nil, apperr.InvalidArgument("invalid profile", nil, apperr.FieldViolation{
			Field:       "profile",
			Description: "required",
		})
	}

	profileReq := req.GetProfile()

	profile := domain.TaxProfile{
		TenantID:           tenantID,
		INN:                profileReq.GetInn(),
		LastName:           profileReq.GetLastName(),
		FirstName:          profileReq.GetFirstName(),
		MiddleName:         profileReq.GetMiddleName(),
		Timezone:           strings.TrimSpace(profileReq.GetTimezone()),
		Phone:              profileReq.GetPhone(),
		Wallets:            toWallets(profileReq.GetWallets()),
		TaxResidencyStatus: domain.TaxResidency(strings.TrimSpace(profileReq.GetTaxResidencyStatus())),
		TaxPayerType:       domain.TaxPayerType(strings.TrimSpace(profileReq.GetTaxpayerType())),
	}

	if err := s.taxProfileUC.Upsert(ctx, profile); err != nil {
		return nil, err
	}

	updated, err := s.taxProfileUC.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return &taxv1.UpsertTaxProfileResponse{
		Profile: toProtoTaxProfile(updated),
	}, nil
}

func (s *TaxServer) StartReport(ctx context.Context, req *taxv1.StartReportRequest) (*taxv1.StartReportResponse, error) {
	if req == nil {
		return nil, apperr.InvalidArgument("invalid request", nil, apperr.FieldViolation{
			Field:       "request",
			Description: "required",
		})
	}

	tenantID, err := parseUUID(req.GetTenantId())
	if err != nil {
		return nil, apperr.InvalidArgument("invalid tenant_id", err, apperr.FieldViolation{
			Field:       "tenant_id",
			Description: "invalid uuid",
		})
	}
	if err := requireTenantMatchIfPresent(ctx, tenantID); err != nil {
		return nil, err
	}
	if req.GetParams() == nil {
		return nil, apperr.InvalidArgument("invalid params", nil, apperr.FieldViolation{
			Field:       "params",
			Description: "required",
		})
	}

	params := req.GetParams()
	if params.GetTaxPolicy() == nil {
		return nil, apperr.InvalidArgument("invalid tax_policy", nil, apperr.FieldViolation{
			Field:       "params.tax_policy",
			Description: "required",
		})
	}
	policy := params.GetTaxPolicy()
	taxPolicy := domain.TaxPolicy{
		TreatCryptoCryptoAsDisposal: policy.GetTreatCryptoCryptoAsDisposal(),
		CostBasisMethod:             domain.CostBasisMethod(policy.GetCostBasisMethod()),
		Jurisdiction:                domain.Jurisdiction(policy.GetJurisdiction()),
	}.Normalize()
	if err := taxPolicy.Validate(); err != nil {
		violations := []apperr.FieldViolation{
			{
				Field:       "params.tax_policy.cost_basis_method",
				Description: "must be FIFO, LIFO or AVG",
			},
			{
				Field:       "params.tax_policy.jurisdiction",
				Description: "must be a supported jurisdiction",
			},
		}
		return nil, apperr.InvalidArgument("invalid tax_policy", err, violations...)
	}

	job, err := s.taxJobUC.Enqueue(
		ctx,
		tenantID,
		int(params.GetTaxYear()),
		taxPolicy,
	)
	if err != nil {
		return nil, err
	}

	return &taxv1.StartReportResponse{
		ReportId: job.ID.String(),
		Status:   string(job.Status),
	}, nil
}

func (s *TaxServer) GetReportStatus(ctx context.Context, req *taxv1.GetReportStatusRequest) (*taxv1.GetReportStatusResponse, error) {
	if req == nil {
		return nil, apperr.InvalidArgument("invalid request", nil, apperr.FieldViolation{
			Field:       "request",
			Description: "required",
		})
	}

	tenantID, err := parseUUID(req.GetTenantId())
	if err != nil {
		return nil, apperr.InvalidArgument("invalid tenant_id", err, apperr.FieldViolation{
			Field:       "tenant_id",
			Description: "invalid uuid",
		})
	}
	if err := requireTenantMatchIfPresent(ctx, tenantID); err != nil {
		return nil, err
	}

	jobID, err := parseUUID(req.GetReportId())
	if err != nil {
		return nil, apperr.InvalidArgument("invalid report_id", err, apperr.FieldViolation{
			Field:       "report_id",
			Description: "invalid uuid",
		})
	}

	job, err := s.taxJobUC.GetStatus(ctx, tenantID, jobID)
	if err != nil {
		return nil, err
	}

	return &taxv1.GetReportStatusResponse{
		Job: toProtoTaxJob(job),
	}, nil
}

func (s *TaxServer) ListReports(ctx context.Context, req *taxv1.ListReportsRequest) (*taxv1.ListReportsResponse, error) {
	if req == nil {
		return nil, apperr.InvalidArgument("invalid request", nil, apperr.FieldViolation{
			Field:       "request",
			Description: "required",
		})
	}

	tenantID, err := parseUUID(req.GetTenantId())
	if err != nil {
		return nil, apperr.InvalidArgument("invalid tenant_id", err, apperr.FieldViolation{
			Field:       "tenant_id",
			Description: "invalid uuid",
		})
	}
	if err := requireTenantMatchIfPresent(ctx, tenantID); err != nil {
		return nil, err
	}

	jobs, total, err := s.taxJobUC.List(ctx, tenantID, req.GetLimit(), req.GetOffset())
	if err != nil {
		return nil, err
	}

	out := make([]*taxv1.TaxJob, 0, len(jobs))
	for _, job := range jobs {
		out = append(out, toProtoTaxJob(job))
	}

	return &taxv1.ListReportsResponse{
		Jobs:  out,
		Total: total,
	}, nil
}

func parseUUID(value string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(value))
}

func toWallets(in []string) []domain.Wallet {
	if len(in) == 0 {
		return nil
	}
	out := make([]domain.Wallet, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		out = append(out, domain.Wallet(v))
	}
	return out
}

func toProtoTaxProfile(profile domain.TaxProfile) *taxv1.TaxProfile {
	if profile.TenantID == uuid.Nil {
		return nil
	}

	wallets := make([]string, 0, len(profile.Wallets))
	for _, w := range profile.Wallets {
		wallets = append(wallets, string(w))
	}

	return &taxv1.TaxProfile{
		TenantId:           profile.TenantID.String(),
		Inn:                profile.INN,
		LastName:           profile.LastName,
		FirstName:          profile.FirstName,
		MiddleName:         profile.MiddleName,
		Timezone:           profile.Timezone,
		Phone:              profile.Phone,
		Wallets:            wallets,
		TaxResidencyStatus: string(profile.TaxResidencyStatus),
		TaxpayerType:       string(profile.TaxPayerType),
	}
}

func toProtoTaxJob(job domain.TaxJob) *taxv1.TaxJob {
	out := &taxv1.TaxJob{
		ReportId:         job.ID.String(),
		TenantId:         job.TenantID.String(),
		TaxYear:          int32(job.TaxYear),
		PolicySnapshot:   toProtoPolicy(job.PolicySnapshot),
		Status:           string(job.Status),
		Attempts:         int32(job.Attempts),
		LastErrorCode:    optionalString(job.LastErrorCode),
		LastErrorMessage: optionalString(job.LastErrorMessage),
		AuditZipUrl:      optionalString(job.AuditZipURL),
		ReportUrl:        optionalString(job.ReportURL),
		Summary:          toProtoTaxSummary(job.Summary),
	}
	if !job.CreatedAt.IsZero() {
		out.CreatedAt = timestamppb.New(job.CreatedAt)
	}
	if job.StartedAt != nil {
		out.StartedAt = timestamppb.New(*job.StartedAt)
	}
	if job.FinishedAt != nil {
		out.FinishedAt = timestamppb.New(*job.FinishedAt)
	}
	return out
}

func toProtoPolicy(policy domain.TaxPolicy) *taxv1.TaxPolicy {
	return &taxv1.TaxPolicy{
		TreatCryptoCryptoAsDisposal: policy.TreatCryptoCryptoAsDisposal,
		CostBasisMethod:             string(policy.CostBasisMethod),
		Jurisdiction:                string(policy.Jurisdiction),
	}
}

func toProtoTaxSummary(summary *domain.TaxSummary) *taxv1.TaxSummary {
	if summary == nil {
		return nil
	}
	return &taxv1.TaxSummary{
		TotalIncomeFiat:  summary.TotalIncome.String(),
		TotalExpenseFiat: summary.TotalExpense.String(),
		TaxBaseFiat:      summary.TaxBase.String(),
		TaxDueFiat:       summary.TaxDue.String(),
	}
}

func optionalString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
