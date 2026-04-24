package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"syscall"
	"time"

	reportv1 "github.com/NightRunner/CryptoTax-Go/gen/report/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/stats"

	"github.com/NightRunner/CryptoTax-Go/pkg/logger"
	"github.com/NightRunner/CryptoTax-Go/pkg/telemetry"
	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/config"
	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/infra/storage"
	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/interceptors"
	grpcserver "github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/server"
	usecase "github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/usecases"
)

var interruptSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGINT,
}

func Run(cfg *config.Config) {
	ctx, stop := signal.NotifyContext(context.Background(), interruptSignals...)
	defer stop()

	waitGroup, ctx := errgroup.WithContext(ctx)

	baseLog := zap.NewExample()
	telemetryProviders, err := telemetry.Init(ctx, telemetry.Config{
		ServiceName:                 cfg.App.Name,
		ServiceVersion:              cfg.App.Version,
		Environment:                 cfg.App.Env,
		OTLPEndpoint:                cfg.OTel.Endpoint,
		Insecure:                    cfg.OTel.Insecure,
		MetricsExportInterval:       cfg.OTel.MetricsExportInterval,
		RuntimeReadMemStatsInterval: cfg.OTel.RuntimeReadMemStatsInterval,
	})
	if err != nil {
		baseLog.Fatal("telemetry init failed", zap.Error(err))
	}

	otelCore := otelzap.NewCore(cfg.App.Name,
		otelzap.WithLoggerProvider(telemetryProviders.LogProvider),
		otelzap.WithVersion(cfg.App.Version),
	)

	log, err := logger.NewLogger(cfg.Log.Level, cfg.App.Env, logger.WithCore(otelCore))
	if err != nil {
		baseLog.Fatal("logger init failed", zap.Error(err))
	}
	ctx = logger.WithContext(ctx, log)

	objectStorage, err := storage.NewMinIOStorage(cfg.MinIO)
	if err != nil {
		log.Fatal("cannot create storage client", zap.Error(err))
	}

	ndflUC, err := usecase.NewNDFLRenderUC(objectStorage, cfg.App.Version)
	if err != nil {
		log.Fatal("cannot create ndfl renderer", zap.Error(err))
	}
	reportServer := grpcserver.NewReportServer(ndflUC)

	statsHandler := otelgrpc.NewServerHandler(
		otelgrpc.WithTracerProvider(noop.NewTracerProvider()),
		otelgrpc.WithMeterProvider(telemetryProviders.MeterProvider),
	)

	runGrpcServer(ctx, waitGroup, cfg, statsHandler, telemetryProviders, reportServer)
	runGateway(ctx, waitGroup, &cfg.HTTP, cfg.GRPC.Addr)

	if err := waitGroup.Wait(); err != nil {
		log.Fatal("error from wait group", zap.Error(err))
	}
}

func runGrpcServer(
	ctx context.Context,
	waitGroup *errgroup.Group,
	cfg *config.Config,
	statsHandler stats.Handler,
	telemetryProviders *telemetry.Providers,
	reportServer *grpcserver.ReportServer,
) {
	log := logger.FromContext(ctx)

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(statsHandler),
		grpc.ChainUnaryInterceptor(
			interceptors.LogInterceptor(log, interceptors.LogConfig{
				ServiceName:    cfg.App.Name,
				ServiceVersion: cfg.App.Version,
				Environment:    cfg.App.Env,
			}),
			interceptors.RecoveryInterceptor(),
			interceptors.ErrorInterceptor(cfg.App.Name),
		),
	)

	reportv1.RegisterReportServer(grpcServer, reportServer)
	reflection.Register(grpcServer)

	hs := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, hs)
	hs.SetServingStatus("report.v1.Report", healthpb.HealthCheckResponse_SERVING)

	listener, err := net.Listen("tcp", cfg.GRPC.Addr)
	if err != nil {
		log.Fatal("cannot create listener", zap.String("addr", cfg.GRPC.Addr), zap.Error(err))
	}

	waitGroup.Go(func() error {
		log.Info("start gRPC server", zap.String("addr", listener.Addr().String()))
		if err := grpcServer.Serve(listener); err != nil {
			if errors.Is(err, grpc.ErrServerStopped) {
				return nil
			}
			return err
		}
		return nil
	})

	waitGroup.Go(func() error {
		<-ctx.Done()
		grpcServer.GracefulStop()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return telemetryProviders.GracefulStop(shutdownCtx)
	})
}

func runGateway(
	ctx context.Context,
	waitGroup *errgroup.Group,
	cfg *config.HTTPConfig,
	grpcAddr string,
) {
	log := logger.FromContext(ctx)
	grpcAddr = normalizeGRPCAddr(grpcAddr)

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := reportv1.RegisterReportHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		log.Fatal("failed to register gateway", zap.Error(err))
	}

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	waitGroup.Go(func() error {
		log.Info("start HTTP gateway", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return err
		}
		return nil
	})

	waitGroup.Go(func() error {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		return httpServer.Shutdown(shutdownCtx)
	})
}

func normalizeGRPCAddr(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		return net.JoinHostPort("127.0.0.1", port)
	}
	if parsed, err := netip.ParseAddr(host); err == nil && parsed.IsUnspecified() {
		return net.JoinHostPort("127.0.0.1", port)
	}
	return addr
}
