package grpcserver

import (
	"context"
	"fmt"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	v1 "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/gen/price/v1"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/pkg/logger"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LegKind int

const (
	LegIn LegKind = iota
	LegOut
	LegFee
)

type slot struct {
	txIdx  int
	kind   LegKind
	symbol string
	coinID string
	result **v1.FiatLeg
}

type PriceServer struct {
	v1.UnimplementedPriceServer
	log               *logger.ZeroLogger
	resolver          domain.CoinIdResolver
	historicalPriceUC domain.HistoricalPriceUseCase
	tenantSymbolUC    domain.TenantSymbolUseCase
}

func NewPriceServer(log *logger.ZeroLogger, resolver domain.CoinIdResolver, historicalPriceUC domain.HistoricalPriceUseCase, tenantSymbolUC domain.TenantSymbolUseCase) *PriceServer {
	return &PriceServer{
		log:               log,
		resolver:          resolver,
		historicalPriceUC: historicalPriceUC,
		tenantSymbolUC:    tenantSymbolUC,
	}
}

func (server *PriceServer) ValuateTransactionsBatch(ctx context.Context, req *v1.ValuateTransactionsRequest) (*v1.ValuateTransactionsResponse, error) {
	start := time.Now()
	server.log.Info("ValuateTransactionsBatch: start txs=%d fiat=%s", len(req.Transactions), req.FiatCurrency)

	resp := &v1.ValuateTransactionsResponse{
		Transactions: make([]*v1.ValuatedTx, len(req.Transactions)),
	}

	var slots []slot
	var priceKeys []domain.PriceKey

	for i, tx := range req.Transactions {
		if tx.TimeUtc == nil {
			server.log.Warn("ValuateTransactionsBatch: missing TimeUtc tx_idx=%d", i)
			return nil, status.Errorf(codes.InvalidArgument, "transaction %d: missing TimeUtc", i)
		}

		out := &v1.ValuatedTx{
			TxId:   tx.TxId,
			Errors: nil,
		}
		resp.Transactions[i] = out

		add := func(kind LegKind, m *v1.MoneyLeg, result **v1.FiatLeg) {
			if m == nil {
				return
			}

			coinID, err := server.resolver.Resolve(m.Symbol)
			if err != nil {
				out.Errors = append(out.Errors, &v1.AssetError{
					Symbol:     m.Symbol,
					Code:       v1.AssetErrorCode_ASSET_UNKNOWN,
					Candidates: nil,
					Message:    fmt.Sprintf("symbol to coinID resolution failed: %v", err),
				})
				return
			}

			slots = append(slots, slot{
				txIdx:  i,
				kind:   kind,
				symbol: m.Symbol,
				coinID: coinID,
				result: result,
			})
			priceKeys = append(priceKeys, domain.PriceKey{CoinID: coinID, BucketStartUtc: tx.TimeUtc.AsTime()})
		}
		add(LegIn, tx.InMoney, &out.InFiat)
		add(LegOut, tx.OutMoney, &out.OutFiat)
		add(LegFee, tx.FeeMoney, &out.FeeFiat)
	}

	if len(slots) == 0 {
		server.log.Info("ValuateTransactionsBatch: no slots txs=%d duration=%s", len(req.Transactions), time.Since(start))
		return resp, nil
	}

	fiats, err := server.historicalPriceUC.GetHistoricalPrices(ctx, req.FiatCurrency, priceKeys)
	if err != nil {
		server.log.Error("ValuateTransactionsBatch: GetHistoricalPrices failed: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to get historical prices: %v", err)
	}

	if len(fiats) != len(priceKeys) {
		server.log.Error("ValuateTransactionsBatch: pricing invariant violated got=%d expected=%d", len(fiats), len(priceKeys))
		return nil, status.Errorf(codes.Internal, "pricing invariant violated: got %d results for %d keys", len(fiats), len(priceKeys))
	}

	for i, fiat := range fiats {
		s := slots[i]
		*s.result = &v1.FiatLeg{
			Fiat: fiat.String(),
		}
	}

	server.log.Info("ValuateTransactionsBatch: done txs=%d duration=%s", len(req.Transactions), time.Since(start))
	return resp, nil
}

func (server PriceServer) UpsertTenantSymbol(ctx context.Context, req *v1.UpsertTenantSymbolRequest) (*v1.UpsertTenantSymbolResponse, error) {
	tenantId, err := parseUUID(req.TenantId)
	if err != nil {
		server.log.Warn("UpsertTenantSymbol: invalid tenant ID: %v", err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant ID: %v", err)
	}

	tenantSymbol := domain.TenantSymbol{
		TenantID: tenantId,
		Source:   req.Source,
		Symbol:   req.Symbol,
		CoinID:   req.CoinId,
	}

	server.tenantSymbolUC.Upsert(ctx, tenantSymbol)

	server.log.Info("UpsertTenantSymbol: upserted tenant_id=%s source=%s symbol=%s", tenantId.String(), req.Source, req.Symbol)
	return nil, status.Error(codes.Unimplemented, "method UpsertTenantSymbol not implemented")
}

func parseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}
