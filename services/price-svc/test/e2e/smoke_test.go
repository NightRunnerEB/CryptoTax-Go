//go:build e2e

package e2e

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/NightRunner/CryptoTax-Go/gen/price/v1"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/interceptors"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/mocks"
	grpcserver "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/server"
)

const bufSize = 1024 * 1024

func startTestServer(
	t *testing.T,
	resolver domain.CoinIdResolver,
	huc domain.HistoricalPriceUseCase,
	tenantSymbolUC domain.TenantSymbolUseCase,
) (v1.PriceClient, func()) {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	baseLog := zap.NewNop()

	server := grpcserver.NewPriceServer(resolver, huc, tenantSymbolUC)

	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.AccessLogInterceptor(baseLog, interceptors.AccessLogConfig{
				ServiceName:    "price-svc",
				ServiceVersion: "test",
				Environment:    "test",
			}),
			interceptors.RecoveryInterceptor(),
			interceptors.ErrorInterceptor("price-svc"),
		),
	)
	v1.RegisterPriceServer(grpcSrv, server)

	go func() {
		_ = grpcSrv.Serve(lis)
	}()

	dialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("grpc dial failed: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		grpcSrv.Stop()
		_ = lis.Close()
	}
	return v1.NewPriceClient(conn), cleanup
}

func TestSmoke_SupportedFiatUSD(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	resolver := mocks.NewMockCoinIdResolver(ctrl)
	resolver.EXPECT().Resolve("BTC").Return("bitcoin", nil).AnyTimes()

	tenantSymbolUC := mocks.NewMockTenantSymbolUseCase(ctrl)

	huc := mocks.NewMockHistoricalPriceUseCase(ctrl)
	huc.EXPECT().GetHistoricalPrices(gomock.Any(), "USD", gomock.Any()).
		Return([]domain.Fiat{decimal.NewFromInt(100)}, nil).
		Times(1)

	client, cleanup := startTestServer(t, resolver, huc, tenantSymbolUC)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := client.ValuateTransactionsBatch(ctx, &v1.ValuateTransactionsRequest{
		TenantId:     "550e8400-e29b-41d4-a716-446655440000",
		Source:       "MEXC",
		FiatCurrency: "USD",
		Transactions: []*v1.TxToValuate{
			{
				TxId:    "tx-1",
				TimeUtc: timestamppb.New(time.Now().UTC().Add(-time.Hour)),
				InMoney: &v1.MoneyLeg{
					Symbol: "BTC",
					Amount: "2",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(resp.Transactions))
	}
	if resp.Transactions[0].GetInFiat() == nil || resp.Transactions[0].GetInFiat().GetFiat() == "" {
		t.Fatalf("expected non-empty in_fiat, got %+v", resp.Transactions[0].GetInFiat())
	}
}

func TestSmoke_UnsupportedFiatABC(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	resolver := mocks.NewMockCoinIdResolver(ctrl)
	resolver.EXPECT().Resolve("BTC").Return("bitcoin", nil).AnyTimes()

	tenantSymbolUC := mocks.NewMockTenantSymbolUseCase(ctrl)

	huc := mocks.NewMockHistoricalPriceUseCase(ctrl)
	huc.EXPECT().GetHistoricalPrices(gomock.Any(), "ABC", gomock.Any()).
		Return(nil, apperr.UnsupportedFiat("unsupported fiat currency", "ABC")).
		Times(1)

	client, cleanup := startTestServer(t, resolver, huc, tenantSymbolUC)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := client.ValuateTransactionsBatch(ctx, &v1.ValuateTransactionsRequest{
		TenantId:     "550e8400-e29b-41d4-a716-446655440000",
		Source:       "MEXC",
		FiatCurrency: "ABC",
		Transactions: []*v1.TxToValuate{
			{
				TxId:    "tx-2",
				TimeUtc: timestamppb.New(time.Now().UTC().Add(-time.Hour)),
				InMoney: &v1.MoneyLeg{
					Symbol: "BTC",
					Amount: "1",
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %T", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %s", st.Code())
	}
}
