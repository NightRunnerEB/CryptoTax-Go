package aggregation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	aggregationv1 "github.com/NightRunner/CryptoTax-Go/gen/aggregation/v1"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/config"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
)

type Client struct {
	addr    string
	timeout time.Duration
	conn    *grpc.ClientConn
	client  aggregationv1.AggregationClient
}

const headerTenantID = "x-tenant-id"

const aggregationDataNotReadyReason = "DATA_NOT_READY"

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
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) ListTransactionsByRange(
	ctx context.Context,
	tenantID uuid.UUID,
	fromUTC, toUTC time.Time,
) ([]domain.AggregatedTransaction, error) {
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
	ctx = metadata.AppendToOutgoingContext(ctx, headerTenantID, tenantID.String())
	resp, err := c.client.ListTransactionsByRange(ctx, &aggregationv1.ListTransactionsByRangeRequest{
		TenantId: tenantID.String(),
		FromUtc:  timestamppb.New(fromUTC),
		ToUtc:    timestamppb.New(toUTC),
		Limit:    1_000_000, // unlimited for now, can add pagination later if needed
		Offset:   0,
	})
	if err != nil {
		meta := map[string]string{
			"tenant_id": tenantID.String(),
		}
		if st, ok := status.FromError(err); ok {
			meta["grpc_code"] = st.Code().String()
			if info := aggregationErrorInfo(st); info != nil {
				meta["aggregation_reason"] = info.GetReason()
				for k, v := range info.GetMetadata() {
					meta["aggregation_"+k] = v
				}
			}
			switch st.Code() {
			case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
				return nil, apperr.AggregationUnavailable("aggregation ListTransactionsByRange failed", err, meta)
			case codes.FailedPrecondition:
				if info := aggregationErrorInfo(st); info != nil && info.GetReason() == aggregationDataNotReadyReason {
					return nil, apperr.NeedsPriceResolution(
						"aggregation data is not ready, resolve fiat valuations and retry",
						err,
						meta,
					)
				}
				return nil, apperr.AggregationBadResponse("aggregation ListTransactionsByRange returned non-retryable status", err, meta)
			case codes.InvalidArgument, codes.NotFound, codes.PermissionDenied, codes.Unauthenticated:
				return nil, apperr.AggregationBadResponse("aggregation ListTransactionsByRange returned non-retryable status", err, meta)
			default:
				return nil, apperr.AggregationFetchFailed("aggregation ListTransactionsByRange failed", err, meta)
			}
		}
		return nil, apperr.AggregationFetchFailed("aggregation ListTransactionsByRange failed", err, meta)
	}

	batch := resp.GetTransactions()
	out := make([]domain.AggregatedTransaction, 0, len(batch))
	for _, tx := range batch {
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
			Kind:           domain.Kind(tx.GetKind()),
			InMoney:        parseMoneyLeg(structMap(tx.GetInMoney())),
			OutMoney:       parseMoneyLeg(structMap(tx.GetOutMoney())),
			FeeMoney:       parseMoneyLeg(structMap(tx.GetFeeMoney())),
			ContractSymbol: tx.ContractSymbol,
			PositionID:     tx.PositionId,
			OrderID:        tx.OrderId,
			TxHash:         tx.TxHash,
		}
		out = append(out, item)
	}

	return out, nil
}

var _ domain.AggregatedTxProvider = (*Client)(nil)

func parseMoneyLeg(raw map[string]any) *domain.MoneyLeg {
	if len(raw) == 0 {
		return nil
	}

	leg := &domain.MoneyLeg{
		Symbol:       asString(raw["symbol"]),
		CryptoAmount: asString(raw["crypto_amount"]),
	}
	if fiat := strings.TrimSpace(asString(raw["fiat_amount"])); fiat != "" {
		leg.FiatAmount = fiat
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

func aggregationErrorInfo(st *status.Status) *errdetails.ErrorInfo {
	if st == nil {
		return nil
	}
	for _, detail := range st.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if ok {
			return info
		}
	}
	return nil
}
