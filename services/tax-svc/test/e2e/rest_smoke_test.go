//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	taxv1 "github.com/NightRunner/CryptoTax-Go/gen/tax/v1"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/interceptors"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/mocks"
	grpcserver "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/server"
)

func setupRESTGateway(t *testing.T, profileUC domain.TaxProfileUseCase, jobUC domain.TaxJobUseCase) (http.Handler, func()) {
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

	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			switch strings.ToLower(key) {
			case "x-tenant-id", "x-user-id", "x-role", "x-request-id", "authorization":
				return key, true
			default:
				return runtime.DefaultHeaderMatcher(key)
			}
		}),
	)

	if err := taxv1.RegisterTaxHandlerFromEndpoint(
		context.Background(),
		mux,
		"passthrough:///bufnet",
		[]grpc.DialOption{
			grpc.WithContextDialer(dialer),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		},
	); err != nil {
		t.Fatalf("register gateway handler failed: %v", err)
	}

	cleanup := func() {
		grpcSrv.Stop()
		_ = lis.Close()
	}
	return mux, cleanup
}

func TestRESTSmoke_StartReport_Success(t *testing.T) {
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

	handler, cleanup := setupRESTGateway(t, profileUC, jobUC)
	defer cleanup()

	body := map[string]any{
		"taxYear": 2025,
		"taxPolicy": map[string]any{
			"treatCryptoCryptoAsDisposal": true,
			"costBasisMethod":             "FIFO",
			"jurisdiction":                "RU",
		},
	}
	raw, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/tax/reports:start", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-Id", tenantID.String())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%s", rec.Code, rec.Body.String())
	}

	var parsed struct {
		ReportID string `json:"reportId"`
		Status   string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&parsed); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if parsed.ReportID != jobID.String() {
		t.Fatalf("report id mismatch: got %s want %s", parsed.ReportID, jobID.String())
	}
	if parsed.Status != string(domain.JobQueued) {
		t.Fatalf("status mismatch: got %s want %s", parsed.Status, domain.JobQueued)
	}
}

func TestRESTSmoke_StartReport_MissingTenantHeader(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	profileUC := mocks.NewMockTaxProfileUseCase(ctrl)
	jobUC := mocks.NewMockTaxJobUseCase(ctrl)

	handler, cleanup := setupRESTGateway(t, profileUC, jobUC)
	defer cleanup()

	body := map[string]any{
		"taxYear": 2025,
		"taxPolicy": map[string]any{
			"costBasisMethod": "FIFO",
			"jurisdiction":    "RU",
		},
	}
	raw, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/tax/reports:start", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d body=%s", rec.Code, rec.Body.String())
	}

	payload, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read response body failed: %v", err)
	}
	if !strings.Contains(string(payload), "missing tenant header") {
		t.Fatalf("expected missing tenant header in body, got: %s", string(payload))
	}
}
