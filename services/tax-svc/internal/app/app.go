package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/stats"
	"google.golang.org/protobuf/proto"

	taxv1 "github.com/NightRunner/CryptoTax-Go/gen/tax/v1"
	"github.com/NightRunner/CryptoTax-Go/pkg/logger"
	"github.com/NightRunner/CryptoTax-Go/pkg/postgres"
	"github.com/NightRunner/CryptoTax-Go/pkg/telemetry"
	db "github.com/NightRunner/CryptoTax-Go/services/tax-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/clients/aggregation"
	reportclient "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/clients/report"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/config"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/engines"
	engineskz "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/engines/kz"
	enginesru "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/engines/ru"
	repository "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/infra/repo"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/infra/storage"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/interceptors"
	grpcserver "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/server"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/usecases"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/worker"
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

	pg, err := postgres.New(
		ctx,
		cfg.PG.URL,
		postgres.MaxPoolSize(cfg.PG.MaxConns),
		postgres.ConnTimeout(cfg.PG.ConnTimeout),
		postgres.ConnAttempts(cfg.PG.ConnAttempts),
		postgres.AttemptTimeout(cfg.PG.AttemptTimeout),
	)
	if err != nil {
		log.Fatal("cannot connect to postgres", zap.Error(err))
	}
	defer pg.Close()

	store := db.NewStore(pg)

	aggregationClient, err := aggregation.NewClient(ctx, cfg.Aggregation)
	if err != nil {
		log.Fatal("cannot create aggregation client", zap.Error(err))
	}
	defer aggregationClient.Close()

	reportClient, err := reportclient.NewClient(ctx, cfg.Report.Addr, cfg.Report.Timeout)
	if err != nil {
		log.Fatal("cannot create report client", zap.Error(err))
	}
	defer reportClient.Close()

	objectStorage, err := storage.NewMinIOStorage(ctx, cfg.MinIO)
	if err != nil {
		log.Fatal("cannot create storage client", zap.Error(err))
	}

	taxProfileRepo := repository.NewTaxProfileRepo(store)
	taxJobRepo := repository.NewTaxJobRepo(store)
	engineRegistry, err := engines.NewRegistry(
		enginesru.New(),
		engineskz.New(),
	)
	if err != nil {
		log.Fatal("cannot create engines registry", zap.Error(err))
	}

	taxProfileUC := usecases.NewTaxProfileUC(taxProfileRepo, engineRegistry)
	taxJobUC := usecases.NewTaxJobUC(taxJobRepo, taxProfileRepo, objectStorage)
	taxJobWorkerUC := usecases.NewTaxJobWorkerUC(
		taxJobRepo,
		taxProfileRepo,
		aggregationClient,
		reportClient,
		objectStorage,
		engineRegistry,
		cfg.Worker.RetryMaxAttempts,
		cfg.Worker.RetryBaseDelay,
		cfg.Worker.RetryMaxDelay,
	)

	statsHandler := otelgrpc.NewServerHandler(
		otelgrpc.WithTracerProvider(noop.NewTracerProvider()),
		otelgrpc.WithMeterProvider(telemetryProviders.MeterProvider),
	)

	runGrpcServer(
		ctx,
		waitGroup,
		cfg,
		statsHandler,
		telemetryProviders,
		taxProfileUC,
		taxJobUC,
	)

	runGateway(ctx, waitGroup, &cfg.HTTP, cfg.GRPC.Addr)

	taxJobWorker := worker.NewTaxJobWorker(
		taxJobWorkerUC,
		log,
		cfg.Worker.PollInterval,
		cfg.Worker.IdleSleep,
	)
	waitGroup.Go(func() error {
		return taxJobWorker.Start(ctx)
	})

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
	taxProfileUC *usecases.TaxProfileUC,
	taxJobUC *usecases.TaxJobUC,
) {
	log := logger.FromContext(ctx)
	server := grpcserver.NewTaxServer(taxProfileUC, taxJobUC)

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

	taxv1.RegisterTaxServer(grpcServer, server)
	reflection.Register(grpcServer)

	hs := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, hs)
	hs.SetServingStatus("tax.v1.Tax", healthpb.HealthCheckResponse_SERVING)

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

func runGateway(ctx context.Context, waitGroup *errgroup.Group, cfg *config.HTTPConfig, grpcAddr string) {
	log := logger.FromContext(ctx)

	grpcAddr = normalizeGRPCAddr(grpcAddr)
	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(incomingHeaderMatcher),
		runtime.WithForwardResponseOption(httpStatusFromMetadata),
	)

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := taxv1.RegisterTaxHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		log.Fatal("failed to register gateway", zap.Error(err))
	}

	handler := otelhttp.NewHandler(mux, "grpc-gateway-api")
	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
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

func incomingHeaderMatcher(key string) (string, bool) {
	switch strings.ToLower(key) {
	case "authorization", "x-tenant-id", "x-user-id", "x-roles", "x-request-id":
		return key, true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}

func normalizeGRPCAddr(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		return net.JoinHostPort("127.0.0.1", port)
	}
	return addr
}

func httpStatusFromMetadata(ctx context.Context, w http.ResponseWriter, _ proto.Message) error {
	md, ok := runtime.ServerMetadataFromContext(ctx)
	if !ok {
		return nil
	}
	vals := md.HeaderMD.Get("x-http-code")
	if len(vals) == 0 {
		return nil
	}
	code, err := strconv.Atoi(vals[0])
	if err != nil {
		return nil
	}
	delete(md.HeaderMD, "x-http-code")
	delete(w.Header(), "Grpc-Metadata-X-Http-Code")
	w.WriteHeader(code)
	return nil
}
