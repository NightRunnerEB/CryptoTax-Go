package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

	aggregationv1 "github.com/NightRunner/CryptoTax-Go/gen/aggregation/v1"
	"github.com/NightRunner/CryptoTax-Go/pkg/logger"
	"github.com/NightRunner/CryptoTax-Go/pkg/postgres"
	"github.com/NightRunner/CryptoTax-Go/pkg/redis"
	"github.com/NightRunner/CryptoTax-Go/pkg/telemetry"
	db "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/clients/ledger"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/clients/price"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/config"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/consumer"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/infra/lock"
	repository "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/infra/repo"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/interceptors"
	grpcserver "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/server"
	usecase "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/usecases"
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
	if telemetryProviders == nil {
		baseLog.Fatal("telemetry init failed: providers are nil")
	}

	otelCore := otelzap.NewCore(cfg.App.Name,
		otelzap.WithLoggerProvider(telemetryProviders.LogProvider),
		otelzap.WithVersion(cfg.App.Version),
	)

	logOptions := []logger.Option{logger.WithCore(otelCore)}

	log, err := logger.NewLogger(cfg.Log.Level, cfg.App.Env, logOptions...)
	if err != nil {
		baseLog.Fatal("logger init failed", zap.Error(err))
	}
	ctx = logger.WithContext(ctx, log)

	pg, err := postgres.New(
		ctx,
		cfg.PG.URL,
		postgres.MaxPoolSize(cfg.PG.PoolMax),
		postgres.ConnTimeout(cfg.PG.ConnectTimeout),
		postgres.ConnAttempts(cfg.PG.ConnAttempts),
		postgres.AttemptTimeout(cfg.PG.AttemptTimeout),
	)
	if err != nil {
		log.Fatal("cannot connect to postgres", zap.Error(err))
	}
	defer pg.Close()

	redisClient, err := redis.New(ctx, cfg.Redis.RedisURL, cfg.Redis.Jitter, redis.WithPoolSize(cfg.Redis.PoolMax))
	if err != nil {
		log.Fatal("cannot connect to redis", zap.Error(err))
	}
	defer redisClient.Close()

	ledgerClient := ledger.NewClient(cfg.Ledger)
	priceClient, err := price.NewClient(ctx, cfg.Price)
	if err != nil {
		log.Fatal("cannot create price client", zap.Error(err))
	}
	defer priceClient.Close()

	store := db.NewStore(pg)

	txRepo := repository.NewAggregatedTransactionRepo(store)
	importStateRepo := repository.NewImportStateRepo(store)
	tenantSettingsRepo := repository.NewTenantSettingsRepo(store)
	lockManager := lock.NewRedisLockManager(redisClient)

	settingsUC := usecase.NewTenantSettingsUC(tenantSettingsRepo)
	aggregationUC := usecase.NewAggregationUC(
		txRepo,
		importStateRepo,
		tenantSettingsRepo,
		ledgerClient,
		priceClient,
		lockManager,
		cfg.Price.BatchSize,
		cfg.Aggregation.ImportLockTTL,
	)

	statsHandler := otelgrpc.NewServerHandler(
		otelgrpc.WithTracerProvider(noop.NewTracerProvider()),
		otelgrpc.WithMeterProvider(telemetryProviders.MeterProvider),
	)

	runGrpcServer(
		ctx,
		waitGroup,
		&cfg.GRPC,
		cfg.App.Name,
		cfg.App.Version,
		cfg.App.Env,
		statsHandler,
		telemetryProviders,
		aggregationUC,
		settingsUC,
	)

	runGateway(
		ctx,
		waitGroup,
		&cfg.HTTP,
		cfg.GRPC.Addr,
	)

	if aggregationUC != nil {
		consumer := consumer.NewImportCompletedConsumer(cfg.RabbitMQ, aggregationUC, log)
		waitGroup.Go(func() error {
			return consumer.Start(ctx)
		})
		waitGroup.Go(func() error {
			<-ctx.Done()
			return consumer.Close()
		})
	} else {
		log.Warn("ImportCompletedConsumer disabled: aggregation usecase is not wired")
	}

	err = waitGroup.Wait()
	if err != nil {
		log.Fatal("error from wait group", zap.Error(err))
	}
}

func runGrpcServer(
	ctx context.Context,
	waitGroup *errgroup.Group,
	cfg *config.GRPC,
	serviceName string,
	serviceVersion string,
	environment string,
	statsHandler stats.Handler,
	telemetryProviders *telemetry.Providers,
	aggregationUC domain.AggregationUseCase,
	settingsUC domain.TenantSettingsUseCase,
) {
	log := logger.FromContext(ctx)
	server := grpcserver.NewAggregationServer(aggregationUC, settingsUC)

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(statsHandler),
		grpc.ChainUnaryInterceptor(
			interceptors.LogInterceptor(log, interceptors.LogConfig{
				ServiceName:    serviceName,
				ServiceVersion: serviceVersion,
				Environment:    environment,
			}),
			interceptors.RecoveryInterceptor(),
			interceptors.ErrorInterceptor(serviceName),
		),
	)

	aggregationv1.RegisterAggregationServer(grpcServer, server)
	reflection.Register(grpcServer)

	hs := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, hs)
	hs.SetServingStatus("aggregation.v1.Aggregation", healthpb.HealthCheckResponse_SERVING)

	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		log.Fatal("cannot create listener", zap.String("addr", cfg.Addr), zap.Error(err))
	}

	waitGroup.Go(func() error {
		log.Info("start gRPC server", zap.String("addr", listener.Addr().String()))

		err = grpcServer.Serve(listener)
		if err != nil {
			if errors.Is(err, grpc.ErrServerStopped) {
				return nil
			}
			log.Error("gRPC server failed to serve", zap.Error(err))
			return err
		}

		return nil
	})

	waitGroup.Go(func() error {
		<-ctx.Done()
		log.Info("graceful shutdown gRPC server")

		grpcServer.GracefulStop()
		log.Info("gRPC server is stopped")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetryProviders.GracefulStop(shutdownCtx); err != nil {
			log.Error("telemetry graceful stop failed", zap.Error(err))
		}

		return nil
	})
}

func runGateway(
	ctx context.Context,
	waitGroup *errgroup.Group,
	cfg *config.HTTP,
	grpcAddr string,
) {
	log := logger.FromContext(ctx)

	grpcAddr = normalizeGRPCAddr(grpcAddr)

	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(incomingHeaderMatcher),
	)

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := aggregationv1.RegisterAggregationHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
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
			log.Error("HTTP gateway failed to serve", zap.Error(err))
			return err
		}
		return nil
	})

	waitGroup.Go(func() error {
		<-ctx.Done()
		log.Info("graceful shutdown HTTP gateway")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Error("HTTP gateway shutdown failed", zap.Error(err))
			return err
		}
		log.Info("HTTP gateway is stopped")
		return nil
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
