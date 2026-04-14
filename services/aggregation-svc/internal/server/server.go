package grpcserver

import (
	"context"
	"strconv"
	"time"

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
	settingsUC    domain.UserSettingsUseCase
}

func NewAggregationServer(aggregationUC domain.AggregationUseCase, settingsUC domain.UserSettingsUseCase) *AggregationServer {
	return &AggregationServer{
		aggregationUC: aggregationUC,
		settingsUC:    settingsUC,
	}
}

func (s *AggregationServer) ListTransactions(
	ctx context.Context,
	req *aggregationv1.ListTransactionsRequest,
) (*aggregationv1.ListTransactionsResponse, error) {
	log := logger.FromContext(ctx)
	if req == nil {
		return nil, apperr.InvalidArgument(
			"invalid request",
			nil,
			apperr.FieldViolation{Field: "request", Description: "required"},
		)
	}

	userID, err := userIDFromHeader(ctx)
	if err != nil {
		return nil, err
	}

	var importID *uuid.UUID
	if rawImportID := req.GetImportId(); rawImportID != "" {
		parsedImportID, err := parseUUID(rawImportID)
		if err != nil {
			return nil, apperr.InvalidArgument(
				"invalid import id",
				err,
				apperr.FieldViolation{Field: "import_id", Description: "invalid format"},
			)
		}
		importID = &parsedImportID
	}

	var dateFrom *time.Time
	if req.GetDateFrom() != nil {
		value := req.GetDateFrom().AsTime().UTC()
		dateFrom = &value
	}
	var dateTo *time.Time
	if req.GetDateTo() != nil {
		value := req.GetDateTo().AsTime().UTC()
		dateTo = &value
	}
	if s.aggregationUC == nil {
		return nil, apperr.Internal("aggregation usecase is not configured", nil, map[string]string{
			"user_id": userID.String(),
		})
	}

	page, err := s.aggregationUC.ListTransactions(
		ctx,
		userID,
		domain.ListTransactionsFilter{
			DateFrom: dateFrom,
			DateTo:   dateTo,
			ImportID: importID,
			Source:   req.GetSource(),
			Kind:     req.GetKind(),
		},
		req.GetPageSize(),
		req.GetPageToken(),
		req.GetTargetFiat(),
	)
	if err != nil {
		return nil, err
	}

	items, err := toProtoTransactions(page.Items)
	if err != nil {
		return nil, err
	}

	log.Info(
		"ListTransactions: success",
		zap.String("user_id", userID.String()),
		zap.Int("count", len(items)),
		zap.Bool("has_next", page.NextPageToken != ""),
	)

	return &aggregationv1.ListTransactionsResponse{
		Items:         items,
		NextPageToken: page.NextPageToken,
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
	userID, err := parseUUID(req.UserId)
	if err != nil {
		log.Warn("ListTransactionsByRange: invalid user ID", zap.Error(err))
		return nil, apperr.InvalidArgument(
			"invalid user id",
			err,
			apperr.FieldViolation{Field: "user_id", Description: "invalid format"},
		)
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
			"user_id": userID.String(),
		})
	}

	page, err := s.aggregationUC.ListTransactionsByRange(
		ctx,
		userID,
		fromUTC,
		toUTC,
		req.GetLimit(),
		req.GetOffset(),
		req.GetTargetFiat(),
	)
	if err != nil {
		return nil, err
	}

	items, err := toProtoTransactions(page.Transactions)
	if err != nil {
		return nil, err
	}

	return &aggregationv1.ListTransactionsByRangeResponse{
		Transactions: items,
		Total:        page.Total,
	}, nil
}

func (s *AggregationServer) GetUserSettings(ctx context.Context, req *aggregationv1.GetUserSettingsRequest) (*aggregationv1.GetUserSettingsResponse, error) {
	log := logger.FromContext(ctx)
	if req == nil {
		return nil, apperr.InvalidArgument(
			"invalid request",
			nil,
			apperr.FieldViolation{Field: "request", Description: "required"},
		)
	}
	userID, err := userIDFromHeader(ctx)
	if err != nil {
		return nil, err
	}

	if s.settingsUC == nil {
		return nil, apperr.Internal("user settings usecase is not configured", nil, map[string]string{
			"user_id": userID.String(),
		})
	}

	settings, err := s.settingsUC.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	log.Info("GetUserSettings: success", zap.String("user_id", userID.String()))
	return &aggregationv1.GetUserSettingsResponse{
		Settings: toProtoSettings(settings),
	}, nil
}

func (s *AggregationServer) UpsertUserSettings(ctx context.Context, req *aggregationv1.UpsertUserSettingsRequest) (*aggregationv1.UpsertUserSettingsResponse, error) {
	log := logger.FromContext(ctx)
	if req == nil {
		return nil, apperr.InvalidArgument(
			"invalid request",
			nil,
			apperr.FieldViolation{Field: "request", Description: "required"},
		)
	}
	userID, err := userIDFromHeader(ctx)
	if err != nil {
		return nil, err
	}

	if s.settingsUC == nil {
		return nil, apperr.Internal("user settings usecase is not configured", nil, map[string]string{
			"user_id": userID.String(),
		})
	}

	settings, err := s.settingsUC.Upsert(ctx, domain.UserSettings{
		UserID:       userID,
		FiatCurrency: req.GetFiatCurrency(),
		Timezone:     req.GetTimezone(),
	})
	if err != nil {
		return nil, err
	}

	log.Info("UpsertUserSettings: success", zap.String("user_id", userID.String()))
	return &aggregationv1.UpsertUserSettingsResponse{
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
	if err := requireUserHeader(ctx); err != nil {
		return nil, err
	}

	if s.settingsUC == nil {
		return nil, apperr.Internal("user settings usecase is not configured", nil, nil)
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

func toProtoSettings(settings domain.UserSettings) *aggregationv1.UserSettings {
	return &aggregationv1.UserSettings{
		UserId:       settings.UserID.String(),
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

func toProtoTransactions(txs []domain.AggregatedTransaction) ([]*aggregationv1.AggregatedTx, error) {
	items := make([]*aggregationv1.AggregatedTx, 0, len(txs))
	for idx, tx := range txs {
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
			UserId:         tx.UserID.String(),
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
	return items, nil
}
