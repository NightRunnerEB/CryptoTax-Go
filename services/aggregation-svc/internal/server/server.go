package grpcserver

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	aggregationv1 "github.com/NightRunner/CryptoTax-Go/gen/aggregation/v1"
	"github.com/NightRunner/CryptoTax-Go/pkg/logger"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
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
	if req == nil {
		return nil, apperr.InvalidArgument(
			"invalid request",
			nil,
			apperr.FieldViolation{Field: "request", Description: "required"},
		)
	}
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
	if err := requireTenantHeaderMatch(ctx, tenantID); err != nil {
		return nil, err
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

	page, err := s.aggregationUC.ListTransactionsByImport(ctx, tenantID, importID, req.GetLimit(), req.GetOffset())
	if err != nil {
		return nil, err
	}

	items := make([]*aggregationv1.AggregatedTx, 0, len(page.Transactions))
	for idx, tx := range page.Transactions {
		inMoney, err := toProtoMoneyLeg(tx.InMoney)
		if err != nil {
			return nil, apperr.Internal("failed to map transaction leg", err, map[string]string{
				"field": "in_money",
				"tx_id": tx.ID.String(),
				"idx":   strconv.Itoa(idx),
			})
		}
		outMoney, err := toProtoMoneyLeg(tx.OutMoney)
		if err != nil {
			return nil, apperr.Internal("failed to map transaction leg", err, map[string]string{
				"field": "out_money",
				"tx_id": tx.ID.String(),
				"idx":   strconv.Itoa(idx),
			})
		}
		feeMoney, err := toProtoMoneyLeg(tx.FeeMoney)
		if err != nil {
			return nil, apperr.Internal("failed to map transaction leg", err, map[string]string{
				"field": "fee_money",
				"tx_id": tx.ID.String(),
				"idx":   strconv.Itoa(idx),
			})
		}

		items = append(items, &aggregationv1.AggregatedTx{
			TxId:           tx.ID.String(),
			TenantId:       tx.TenantID.String(),
			Source:         tx.Source,
			ImportId:       tx.ImportID.String(),
			TimeUtc:        timestamppb.New(tx.TimeUTC),
			Kind:           tx.Kind,
			InMoney:        inMoney,
			OutMoney:       outMoney,
			FeeMoney:       feeMoney,
			TxHash:         tx.TxHash,
			Note:           optionalString(tx.Note),
			ContractSymbol: tx.ContractSymbol,
			DerivativeKind: optionalString(tx.DerivativeKind),
			PositionId:     tx.PositionID,
			OrderId:        tx.OrderID,
			TxFingerprint:  tx.TxFingerprint,
		})
	}

	log.Info(
		"ListTransactionsByImport: success",
		zap.String("tenant_id", tenantID.String()),
		zap.String("import_id", importID.String()),
		zap.Int("count", len(items)),
		zap.Int64("total", page.Total),
	)

	return &aggregationv1.ListTransactionsByImportResponse{
		Transactions: items,
		Total:        page.Total,
	}, nil
}

func (s *AggregationServer) ListTransactionsByRange(ctx context.Context, req *aggregationv1.ListTransactionsByRangeRequest) (*aggregationv1.ListTransactionsByRangeResponse, error) {
	log := logger.FromContext(ctx)
	if req == nil {
		return nil, apperr.InvalidArgument(
			"invalid request",
			nil,
			apperr.FieldViolation{Field: "request", Description: "required"},
		)
	}
	if err := requireTenantHeader(ctx); err != nil {
		return nil, err
	}

	tenantID, err := parseUUID(req.TenantId)
	if err != nil {
		log.Warn("ListTransactionsByRange: invalid tenant ID", zap.Error(err))
		return nil, apperr.InvalidArgument(
			"invalid tenant id",
			err,
			apperr.FieldViolation{Field: "tenant_id", Description: "invalid format"},
		)
	}
	if err := requireTenantHeaderMatch(ctx, tenantID); err != nil {
		return nil, err
	}
	if req.GetFromUtc() == nil || req.GetToUtc() == nil {
		return nil, apperr.InvalidArgument(
			"invalid range",
			nil,
			apperr.FieldViolation{Field: "from_utc/to_utc", Description: "required"},
		)
	}
	fromUTC := req.GetFromUtc().AsTime().UTC()
	toUTC := req.GetToUtc().AsTime().UTC()
	if !fromUTC.Before(toUTC) {
		return nil, apperr.InvalidArgument(
			"invalid range",
			nil,
			apperr.FieldViolation{Field: "from_utc/to_utc", Description: "from_utc must be before to_utc"},
		)
	}

	if s.aggregationUC == nil {
		return nil, apperr.Internal("aggregation usecase is not configured", nil, map[string]string{
			"tenant_id": tenantID.String(),
		})
	}

	page, err := s.aggregationUC.ListTransactionsByRange(
		ctx,
		tenantID,
		fromUTC,
		toUTC,
		req.GetLimit(),
		req.GetOffset(),
		req.GetTargetFiat(),
	)
	if err != nil {
		return nil, err
	}

	items := make([]*aggregationv1.AggregatedTx, 0, len(page.Transactions))
	for idx, tx := range page.Transactions {
		inMoney, err := toProtoMoneyLeg(tx.InMoney)
		if err != nil {
			return nil, apperr.Internal("failed to map transaction leg", err, map[string]string{
				"field": "in_money",
				"tx_id": tx.ID.String(),
				"idx":   strconv.Itoa(idx),
			})
		}
		outMoney, err := toProtoMoneyLeg(tx.OutMoney)
		if err != nil {
			return nil, apperr.Internal("failed to map transaction leg", err, map[string]string{
				"field": "out_money",
				"tx_id": tx.ID.String(),
				"idx":   strconv.Itoa(idx),
			})
		}
		feeMoney, err := toProtoMoneyLeg(tx.FeeMoney)
		if err != nil {
			return nil, apperr.Internal("failed to map transaction leg", err, map[string]string{
				"field": "fee_money",
				"tx_id": tx.ID.String(),
				"idx":   strconv.Itoa(idx),
			})
		}

		items = append(items, &aggregationv1.AggregatedTx{
			TxId:           tx.ID.String(),
			TenantId:       tx.TenantID.String(),
			Source:         tx.Source,
			ImportId:       tx.ImportID.String(),
			TimeUtc:        timestamppb.New(tx.TimeUTC),
			Kind:           tx.Kind,
			InMoney:        inMoney,
			OutMoney:       outMoney,
			FeeMoney:       feeMoney,
			TxHash:         tx.TxHash,
			Note:           optionalString(tx.Note),
			ContractSymbol: tx.ContractSymbol,
			DerivativeKind: optionalString(tx.DerivativeKind),
			PositionId:     tx.PositionID,
			OrderId:        tx.OrderID,
			TxFingerprint:  tx.TxFingerprint,
		})
	}

	return &aggregationv1.ListTransactionsByRangeResponse{
		Transactions: items,
		Total:        page.Total,
	}, nil
}

func (s *AggregationServer) GetTenantSettings(ctx context.Context, req *aggregationv1.GetTenantSettingsRequest) (*aggregationv1.GetTenantSettingsResponse, error) {
	log := logger.FromContext(ctx)
	if req == nil {
		return nil, apperr.InvalidArgument(
			"invalid request",
			nil,
			apperr.FieldViolation{Field: "request", Description: "required"},
		)
	}
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
	if err := requireTenantHeaderMatch(ctx, tenantID); err != nil {
		return nil, err
	}

	if s.settingsUC == nil {
		return nil, apperr.Internal("tenant settings usecase is not configured", nil, map[string]string{
			"tenant_id": tenantID.String(),
		})
	}

	settings, err := s.settingsUC.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	log.Info("GetTenantSettings: success", zap.String("tenant_id", tenantID.String()))
	return &aggregationv1.GetTenantSettingsResponse{
		Settings: toProtoSettings(settings),
	}, nil
}

func (s *AggregationServer) UpsertTenantSettings(ctx context.Context, req *aggregationv1.UpsertTenantSettingsRequest) (*aggregationv1.UpsertTenantSettingsResponse, error) {
	log := logger.FromContext(ctx)
	if req == nil {
		return nil, apperr.InvalidArgument(
			"invalid request",
			nil,
			apperr.FieldViolation{Field: "request", Description: "required"},
		)
	}
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
	if err := requireTenantHeaderMatch(ctx, tenantID); err != nil {
		return nil, err
	}

	if s.settingsUC == nil {
		return nil, apperr.Internal("tenant settings usecase is not configured", nil, map[string]string{
			"tenant_id": tenantID.String(),
		})
	}

	settings, err := s.settingsUC.Upsert(ctx, domain.TenantSettings{
		TenantID:     tenantID,
		FiatCurrency: req.GetFiatCurrency(),
		Timezone:     req.GetTimezone(),
	})
	if err != nil {
		return nil, err
	}

	log.Info("UpsertTenantSettings: success", zap.String("tenant_id", tenantID.String()))
	return &aggregationv1.UpsertTenantSettingsResponse{
		Settings: toProtoSettings(settings),
	}, nil
}

func (s *AggregationServer) ListSupportedFiatCurrencies(
	ctx context.Context,
	req *aggregationv1.ListSupportedFiatCurrenciesRequest,
) (*aggregationv1.ListSupportedFiatCurrenciesResponse, error) {
	log := logger.FromContext(ctx)
	if req == nil {
		return nil, apperr.InvalidArgument(
			"invalid request",
			nil,
			apperr.FieldViolation{Field: "request", Description: "required"},
		)
	}
	if err := requireTenantHeader(ctx); err != nil {
		return nil, err
	}

	if s.settingsUC == nil {
		return nil, apperr.Internal("tenant settings usecase is not configured", nil, nil)
	}

	currencies, err := s.settingsUC.ListSupportedFiatCurrencies(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*aggregationv1.SupportedFiatCurrency, 0, len(currencies))
	for _, currency := range currencies {
		items = append(items, &aggregationv1.SupportedFiatCurrency{
			Code:        currency.Code,
			DisplayName: currency.DisplayName,
		})
	}

	log.Debug("ListSupportedFiatCurrencies: success", zap.Int("count", len(items)))
	return &aggregationv1.ListSupportedFiatCurrenciesResponse{
		Currencies: items,
	}, nil
}

func parseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func toProtoSettings(settings domain.TenantSettings) *aggregationv1.TenantSettings {
	return &aggregationv1.TenantSettings{
		TenantId:     settings.TenantID.String(),
		FiatCurrency: settings.FiatCurrency,
		Timezone:     settings.Timezone,
	}
}

func toProtoMoneyLeg(leg *domain.MoneyLeg) (*structpb.Struct, error) {
	if leg == nil {
		return nil, nil
	}

	data := map[string]any{
		"symbol":        leg.Symbol,
		"crypto_amount": leg.CryptoAmount,
	}
	if leg.FiatAmount != nil {
		data["fiat_amount"] = *leg.FiatAmount
	}
	if leg.Error != nil {
		errPayload := map[string]any{
			"code": leg.Error.Code,
		}

		if len(leg.Error.Candidates) > 0 {
			candidates := make([]any, 0, len(leg.Error.Candidates))
			for _, candidate := range leg.Error.Candidates {
				candidates = append(candidates, map[string]any{
					"coin_id": candidate.CoinID,
					"name":    candidate.Name,
				})
			}
			errPayload["candidates"] = candidates
		}
		data["error"] = errPayload
	}

	return structpb.NewStruct(data)
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
