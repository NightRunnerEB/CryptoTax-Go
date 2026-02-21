package aggregation

import (
	"context"
	"fmt"
	"strings"
	"time"

	aggregationv1 "github.com/NightRunner/CryptoTax-Go/gen/aggregation/v1"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/config"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Client struct {
	addr    string
	timeout time.Duration
	conn    *grpc.ClientConn
	client  aggregationv1.AggregationClient
}

func NewClient(ctx context.Context, cfg config.AggregationConfig) (*Client, error) {
	conn, err := grpc.NewClient(cfg.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, apperr.AggregationUnavailable("aggregation grpc dial failed", err, map[string]string{
			"addr": cfg.Addr,
		})
	}

	return &Client{
		addr:    cfg.Addr,
		timeout: cfg.Timeout,
		conn:    conn,
		client:  aggregationv1.NewAggregationClient(conn),
	}, nil
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) ListTransactionsByRange(
	ctx context.Context,
	tenantID uuid.UUID,
	fromUTC, toUTC time.Time,
	limit, offset int32,
) ([]domain.AggregatedTransaction, error) {
	if c == nil || c.client == nil {
		return nil, apperr.Internal("aggregation client is not initialized", nil, nil)
	}
	if tenantID == uuid.Nil {
		return nil, apperr.InvalidArgument("invalid tenant id", nil, apperr.FieldViolation{
			Field:       "tenant_id",
			Description: "required",
		})
	}
	if !fromUTC.Before(toUTC) {
		return nil, apperr.InvalidArgument("invalid range", nil, apperr.FieldViolation{
			Field:       "from_utc/to_utc",
			Description: "from_utc must be before to_utc",
		})
	}

	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	resp, err := c.client.ListTransactionsByRange(ctx, &aggregationv1.ListTransactionsByRangeRequest{
		TenantId: tenantID.String(),
		FromUtc:  timestamppb.New(fromUTC),
		ToUtc:    timestamppb.New(toUTC),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, apperr.AggregationUnavailable("aggregation ListTransactionsByRange failed", err, map[string]string{
			"tenant_id": tenantID.String(),
		})
	}

	out := make([]domain.AggregatedTransaction, 0, len(resp.GetTransactions()))
	for _, tx := range resp.GetTransactions() {
		if tx == nil {
			continue
		}
		txID, err := uuid.Parse(strings.TrimSpace(tx.GetTxId()))
		if err != nil {
			return nil, apperr.AggregationBadResponse("invalid tx_id in aggregation response", err, nil)
		}
		importID, err := uuid.Parse(strings.TrimSpace(tx.GetImportId()))
		if err != nil {
			return nil, apperr.AggregationBadResponse("invalid import_id in aggregation response", err, nil)
		}

		item := domain.AggregatedTransaction{
			ID:             txID,
			TenantID:       tenantID,
			Source:         tx.GetSource(),
			ImportID:       importID,
			TimeUTC:        tx.GetTimeUtc().AsTime().UTC(),
			Kind:           tx.GetKind(),
			InMoney:        parseMoneyLeg(structMap(tx.GetInMoney())),
			OutMoney:       parseMoneyLeg(structMap(tx.GetOutMoney())),
			FeeMoney:       parseMoneyLeg(structMap(tx.GetFeeMoney())),
			ContractSymbol: nilIfEmpty(tx.GetContractSymbol()),
			DerivativeKind: nilIfEmpty(tx.GetDerivativeKind()),
			PositionID:     nilIfEmpty(tx.GetPositionId()),
			OrderID:        nilIfEmpty(tx.GetOrderId()),
			TxHash:         nilIfEmpty(tx.GetTxHash()),
			Note:           nilIfEmpty(tx.GetNote()),
			TxFingerprint:  tx.GetTxFingerprint(),
		}
		out = append(out, item)
	}
	return out, nil
}

var _ domain.AggregationClient = (*Client)(nil)

func parseMoneyLeg(raw map[string]any) *domain.MoneyLeg {
	if len(raw) == 0 {
		return nil
	}

	leg := &domain.MoneyLeg{
		Symbol:       asString(raw["symbol"]),
		CryptoAmount: asString(raw["crypto_amount"]),
	}
	if fiat := strings.TrimSpace(asString(raw["fiat_amount"])); fiat != "" {
		leg.FiatAmount = &fiat
	}
	if rawErr, ok := raw["error"].(map[string]any); ok {
		mappedErr := &domain.FiatLegError{
			Code: asString(rawErr["code"]),
		}
		if rawCandidates, ok := rawErr["candidates"].([]any); ok {
			for _, candidate := range rawCandidates {
				cMap, ok := candidate.(map[string]any)
				if !ok {
					continue
				}
				mappedErr.Candidates = append(mappedErr.Candidates, domain.FiatLegCandidate{
					CoinID: asString(cMap["coin_id"]),
					Name:   asString(cMap["name"]),
				})
			}
		}
		leg.Error = mappedErr
	}
	return leg
}

func structMap(s *structpb.Struct) map[string]any {
	if s == nil {
		return nil
	}
	return s.AsMap()
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		return fmt.Sprint(t)
	}
}

func nilIfEmpty(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	out := v
	return &out
}
