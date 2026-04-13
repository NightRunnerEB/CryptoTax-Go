//go:build e2e

package e2e

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	taxv1 "github.com/NightRunner/CryptoTax-Go/gen/tax/v1"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/interceptors"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/mocks"
	grpcserver "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/server"
)

const bufSize = 1024 * 1024

func startTestServer(t *testing.T, profileUC domain.TaxProfileUseCase, jobUC domain.TaxJobUseCase) (taxv1.TaxClient, func()) {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	server := grpcserver.NewTaxServer(profileUC, jobUC)

	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.LogInterceptor(zap.NewNop(), interceptors.LogConfig{
				ServiceName:    "tax-svc",
				ServiceVersion: "test",
				Environment:    "test",
			}),
			interceptors.RecoveryInterceptor(),
			interceptors.ErrorInterceptor("tax-svc"),
		),
	)
	taxv1.RegisterTaxServer(grpcSrv, server)

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
	return taxv1.NewTaxClient(conn), cleanup
}

func TestSmoke_StartReport_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tenantID := uuid.New()
	jobID := uuid.New()

	profileUC := mocks.NewMockTaxProfileUseCase(ctrl)
	jobUC := mocks.NewMockTaxJobUseCase(ctrl)
	jobUC.EXPECT().Enqueue(gomock.Any(), tenantID, 2025, domain.TaxPolicy{
		TreatCryptoCryptoAsDisposal: true,
		CostBasisMethod:             domain.FIFO,
		Jurisdiction:                domain.JurisdictionRU,
	}).Return(domain.TaxJob{ID: jobID, Status: domain.JobQueued}, nil).Times(1)

	client, cleanup := startTestServer(t, profileUC, jobUC)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-tenant-id", tenantID.String())

	resp, err := client.StartReport(ctx, &taxv1.StartReportRequest{
		TenantId: tenantID.String(),
		Params: &taxv1.StartReportParams{
			TaxYear: 2025,
			TaxPolicy: &taxv1.TaxPolicy{
				TreatCryptoCryptoAsDisposal: true,
				CostBasisMethod:             "FIFO",
				Jurisdiction:                "RU",
			},
		},
	})
	if err != nil {
		t.Fatalf("StartReport() unexpected error: %v", err)
	}
	if resp.GetReportId() != jobID.String() {
		t.Fatalf("report id mismatch: got %s want %s", resp.GetReportId(), jobID.String())
	}
	if resp.GetStatus() != string(domain.JobQueued) {
		t.Fatalf("status mismatch: got %s want %s", resp.GetStatus(), domain.JobQueued)
	}
}

func TestSmoke_StartReport_MissingTenantHeader(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	profileUC := mocks.NewMockTaxProfileUseCase(ctrl)
	jobUC := mocks.NewMockTaxJobUseCase(ctrl)

	client, cleanup := startTestServer(t, profileUC, jobUC)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := client.StartReport(ctx, &taxv1.StartReportRequest{
		Params: &taxv1.StartReportParams{
			TaxYear: 2025,
			TaxPolicy: &taxv1.TaxPolicy{
				CostBasisMethod: "FIFO",
				Jurisdiction:    "RU",
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
