package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/stats"

	pricev1 "github.com/NightRunner/CryptoTax-Go/gen/price/v1"
	"github.com/NightRunner/CryptoTax-Go/pkg/logger"
	"github.com/NightRunner/CryptoTax-Go/pkg/postgres"
	"github.com/NightRunner/CryptoTax-Go/pkg/redis"
	"github.com/NightRunner/CryptoTax-Go/pkg/telemetry"
	db "github.com/NightRunner/CryptoTax-Go/services/price-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/coingecko"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/config"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/fiatfx"
	inmemory "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/infra/in-memory"
	repository "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/infra/repo"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/interceptors"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/resolver"
	grpcserver "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/server"
	usecase "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/usecases"
)

func Run(cfg *config.Config) {
	interruptSignals := []os.Signal{
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
	}

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

	db := db.NewStore(pg)

	redis, err := redis.New(ctx, cfg.Redis.RedisURL, cfg.Redis.Jitter, redis.WithPoolSize(cfg.Redis.PoolMax))
	if err != nil {
		log.Fatal("cannot connect to redis", zap.Error(err))
	}
	defer redis.Close()

	userSymbolRepo := repository.NewUserSymbolRepo(db)
	historicalPriceRepo := repository.NewHistoricalPriceRepo(db)
	fxRateRepo := repository.NewFXRateRepo(db)

	userSymbolUC := usecase.NewUserSymbolUC(userSymbolRepo, time.Second*5)

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	fxLocker := fiatfx.NewPGAdvisoryLocker(pg.ConnPool())

	KZTSource, err := fiatfx.NewKZTSource(ctx, httpClient, fxRateRepo, fxLocker)
	if err != nil {
		log.Fatal("new kzt source failed", zap.Error(err))
	}
	RUBSource, err := fiatfx.NewRUBSource(ctx, httpClient, fxRateRepo, fxLocker)
	if err != nil {
		log.Fatal("new rub source failed", zap.Error(err))
	}

	fxSourceRegistry := fiatfx.NewFXRegistry()
	fxSourceRegistry.Register(RUBSource)
	fxSourceRegistry.Register(KZTSource)
	fxProvider := fiatfx.NewFXProvider(fxSourceRegistry)
	// НАСТРОИТЬ CONTEXT - сейчас поставил от waitGroup
	if err := fxProvider.Start(ctx); err != nil {
		log.Fatal("fx provider start failed", zap.Error(err))
	}

	cgClient, err := coingecko.NewCGClient(cfg.CG)
	if err != nil {
		log.Fatal("cannot create coingecko client", zap.Error(err))
	}

	historicalPriceUC := usecase.NewHistoricalPriceUC(historicalPriceRepo, fxProvider, cgClient, time.Second*5)

	coinIdCache, err := inmemory.NewCoinIdCache(cfg.Resolver.Path)
	if err != nil {
		log.Fatal("cannot create coin id cache", zap.Error(err))
	}
	resolver := resolver.NewCoinIdResolver(userSymbolRepo, coinIdCache)

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
		fxProvider,
		resolver,
		historicalPriceUC,
		userSymbolUC,
	)

	err = waitGroup.Wait()
	if err != nil {
		log.Fatal("error from wait group", zap.Error(err))
	}
}

func runGrpcServer(
	ctx context.Context,
	waitGroup *errgroup.Group,
	config *config.Config,
	statsHandler stats.Handler,
	telemetryProviders *telemetry.Providers,
	fxProvider domain.FXProvider,
	resolver domain.CoinIdResolver,
	historicalPriceUC domain.HistoricalPriceUseCase,
	userSymbolUC domain.UserSymbolUseCase,
) {
	log := logger.FromContext(ctx)
	server := grpcserver.NewPriceServer(resolver, historicalPriceUC, userSymbolUC)

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(statsHandler),
		grpc.ChainUnaryInterceptor(
			interceptors.AccessLogInterceptor(log, interceptors.AccessLogConfig{
				ServiceName:    config.App.Name,
				ServiceVersion: config.App.Version,
				Environment:    config.App.Env,
			}),
			interceptors.RecoveryInterceptor(),
			interceptors.ErrorInterceptor(config.App.Name),
		),
	)
	pricev1.RegisterPriceServer(grpcServer, server)
	reflection.Register(grpcServer)

	hs := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, hs)
	hs.SetServingStatus("price.v1.Price", healthpb.HealthCheckResponse_SERVING)

	listener, err := net.Listen("tcp", config.GRPC.Addr)
	if err != nil {
		log.Fatal("cannot create listener", zap.String("addr", config.GRPC.Addr), zap.Error(err))
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

		grpcServer.GracefulStop()
		log.Info("gRPC server graceful shutdown completed")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := fxProvider.Stop(shutdownCtx); err != nil {
			log.Error("fx provider graceful stop failed", zap.Error(err))
		}

		if err := telemetryProviders.GracefulStop(shutdownCtx); err != nil {
			log.Error("telemetry graceful stop failed", zap.Error(err))
		}
		log.Info("Telemetry graceful shutdown completed")

		return nil
	})
}
