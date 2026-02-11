package grpcserver

import (
	"context"

	aggregationv1 "github.com/NightRunner/CryptoTax-Go/gen/aggregation/v1"
	"github.com/NightRunner/CryptoTax-Go/pkg/logger"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AggregationServer struct {
	aggregationv1.UnimplementedAggregationServer
	aggregationUC domain.AggregationUseCase
	settingsUC    domain.TenantSettingsUseCase
}

func NewAggregationServer(aggregationUC domain.AggregationUseCase, settingsUC domain.TenantSettingsUseCase) *AggregationServer {
	return &AggregationServer{
		aggregationUC: aggregationUC,
		settingsUC:    settingsUC,
	}
}

func (s *AggregationServer) ListTransactionsByImport(ctx context.Context, req *aggregationv1.ListTransactionsByImportRequest) (*aggregationv1.ListTransactionsByImportResponse, error) {
	log := logger.FromContext(ctx)
	if err := requireTenantHeader(ctx); err != nil {
		return nil, err
	}

	tenantID, err := parseUUID(req.TenantId)
	if err != nil {
		log.Warn("ListTransactionsByImport: invalid tenant ID", zap.Error(err))
		return nil, apperr.InvalidArgument(
			"invalid tenant id",
			err,
			apperr.FieldViolation{Field: "tenant_id", Description: "invalid format"},
		)
	}
	importID, err := parseUUID(req.ImportId)
	if err != nil {
		log.Warn("ListTransactionsByImport: invalid import ID", zap.Error(err))
		return nil, apperr.InvalidArgument(
			"invalid import id",
			err,
			apperr.FieldViolation{Field: "import_id", Description: "invalid format"},
		)
	}

	if s.aggregationUC == nil {
		return nil, apperr.Internal("aggregation usecase is not configured", nil, map[string]string{
			"tenant_id": tenantID.String(),
			"import_id": importID.String(),
		})
	}

	log.Info("ListTransactionsByImport: not implemented")
	return nil, apperr.Internal("method not implemented", nil, nil)
}

func (s *AggregationServer) GetTenantSettings(ctx context.Context, req *aggregationv1.GetTenantSettingsRequest) (*aggregationv1.GetTenantSettingsResponse, error) {
	log := logger.FromContext(ctx)
	if err := requireTenantHeader(ctx); err != nil {
		return nil, err
	}

	tenantID, err := parseUUID(req.TenantId)
	if err != nil {
		log.Warn("GetTenantSettings: invalid tenant ID", zap.Error(err))
		return nil, apperr.InvalidArgument(
			"invalid tenant id",
			err,
			apperr.FieldViolation{Field: "tenant_id", Description: "invalid format"},
		)
	}

	if s.settingsUC == nil {
		return nil, apperr.Internal("tenant settings usecase is not configured", nil, map[string]string{
			"tenant_id": tenantID.String(),
		})
	}

	log.Info("GetTenantSettings: not implemented")
	return nil, apperr.Internal("method not implemented", nil, nil)
}

func (s *AggregationServer) UpsertTenantSettings(ctx context.Context, req *aggregationv1.UpsertTenantSettingsRequest) (*aggregationv1.UpsertTenantSettingsResponse, error) {
	log := logger.FromContext(ctx)
	if err := requireTenantHeader(ctx); err != nil {
		return nil, err
	}

	tenantID, err := parseUUID(req.TenantId)
	if err != nil {
		log.Warn("UpsertTenantSettings: invalid tenant ID", zap.Error(err))
		return nil, apperr.InvalidArgument(
			"invalid tenant id",
			err,
			apperr.FieldViolation{Field: "tenant_id", Description: "invalid format"},
		)
	}

	if s.settingsUC == nil {
		return nil, apperr.Internal("tenant settings usecase is not configured", nil, map[string]string{
			"tenant_id": tenantID.String(),
		})
	}

	log.Info("UpsertTenantSettings: not implemented")
	return nil, apperr.Internal("method not implemented", nil, nil)
}

func parseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}
