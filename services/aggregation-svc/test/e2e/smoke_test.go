//go:build e2e

package e2e

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"

	aggregationv1 "github.com/NightRunner/CryptoTax-Go/gen/aggregation/v1"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/interceptors"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/mocks"
	grpcserver "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/server"
)

const bufSize = 1024 * 1024

func startTestServer(
	t *testing.T,
	aggUC domain.AggregationUseCase,
	settingsUC domain.UserSettingsUseCase,
) (aggregationv1.AggregationClient, func()) {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	server := grpcserver.NewAggregationServer(aggUC, settingsUC)

	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.RecoveryInterceptor(),
			interceptors.ErrorInterceptor("aggregation-svc"),
		),
	)
	aggregationv1.RegisterAggregationServer(grpcSrv, server)

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
	return aggregationv1.NewAggregationClient(conn), cleanup
}

func TestSmoke_ListSupportedFiatCurrencies_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	settingsUC := mocks.NewMockUserSettingsUseCase(ctrl)
	settingsUC.EXPECT().
		ListSupportedFiatCurrencies(gomock.Any()).
		Return([]domain.SupportedFiatCurrency{
			{Code: "USD", DisplayName: "US Dollar"},
			{Code: "RUB", DisplayName: "Russian Ruble"},
		}, nil)

	client, cleanup := startTestServer(t, nil, settingsUC)
	defer cleanup()

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("x-user-id", uuid.NewString()))
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp, err := client.ListSupportedFiatCurrencies(ctx, &aggregationv1.ListSupportedFiatCurrenciesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetCurrencies()) != 2 {
		t.Fatalf("expected 2 currencies, got %d", len(resp.GetCurrencies()))
	}
	if resp.GetCurrencies()[0].GetCode() != "USD" || resp.GetCurrencies()[1].GetCode() != "RUB" {
		t.Fatalf("unexpected response: %+v", resp.GetCurrencies())
	}
}

func TestSmoke_ListTransactionsByRange_SucceedsWithoutUserHeader(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	aggUC := mocks.NewMockAggregationUseCase(ctrl)
	userID := uuid.New()
	fromUTC := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	toUTC := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	aggUC.EXPECT().
		ListTransactionsByRange(gomock.Any(), userID, fromUTC, toUTC, int32(0), int32(0), "").
		Return(domain.AggregatedTxPage{}, nil)
	client, cleanup := startTestServer(t, aggUC, nil)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := client.ListTransactionsByRange(ctx, &aggregationv1.ListTransactionsByRangeRequest{
		UserId:  userID.String(),
		FromUtc: timestamppb.New(fromUTC),
		ToUtc:   timestamppb.New(toUTC),
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}
