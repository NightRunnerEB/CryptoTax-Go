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

	db "github.com/NightRunner/CryptoTax-Go/services/price-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/coingecko"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/config"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/fiatfx"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/gen/price/v1"
	inmemory "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/infra/in-memory"
	repository "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/infra/repo"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/resolver"
	grpcserver "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/server"
	usecase "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/usecases"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/pkg/logger"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/pkg/postgres"
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/pkg/redis"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
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

	log := logger.New(cfg.Log.Level)
	pg, err := postgres.New(
		ctx,
		cfg.PG.URL,
		postgres.MaxPoolSize(cfg.PG.PoolMax),
		postgres.ConnTimeout(cfg.PG.ConnectTimeout),
		postgres.ConnAttempts(cfg.PG.ConnAttempts),
		postgres.AttemptTimeout(cfg.PG.AttemptTimeout),
	)
	if err != nil {
		log.Fatal("cannot connect to postgres: %v", err)
	}
	defer pg.Close()

	db := db.NewStore(pg)

	redis, err := redis.New(ctx, cfg.Redis.RedisURL, cfg.Redis.Jitter, redis.WithPoolSize(cfg.Redis.PoolMax))
	if err != nil {
		log.Fatal("cannot connect to redis: %v", err)
	}
	defer redis.Close()

	tenantSymbolRepo := repository.NewTenantSymbolRepo(db)
	historicalPriceRepo := repository.NewHistoricalPriceRepo(db)

	tenantSymbolUC := usecase.NewTenantSymbolUC(tenantSymbolRepo, time.Second*5)

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	fxSourceRegistry := fiatfx.NewFXRegistry()
	fxSourceRegistry.Register(fiatfx.NewRUBSource(httpClient))
	fxSourceRegistry.Register(fiatfx.NewKZTSource(httpClient))
	fxProvider := fiatfx.NewFXProvider(fxSourceRegistry)
	// НАСТРОИТЬ CONTEXT - сейчас поставил от waitGroup
	if err := fxProvider.Start(ctx); err != nil {
		log.Fatal("cannot connect to redis: %v", err)
	}

	cgClient, err := coingecko.NewCGClient(cfg.CG)
	if err != nil {
		log.Fatal("cannot create coingecko client: %v", err)
	}

	historicalPriceUC := usecase.NewHistoricalPriceUC(log, historicalPriceRepo, fxProvider, cgClient, time.Second*5)

	coinIdCache, err := inmemory.NewCoinIdCache(cfg.Resolver.Path)
	if err != nil {
		log.Fatal("cannot create coinIdCache: %v", err)
	}
	resolver := resolver.NewCoinIdResolver(tenantSymbolRepo, coinIdCache)

	runGrpcServer(ctx, waitGroup, &cfg.GRPC, log, resolver, historicalPriceUC, tenantSymbolUC)

	err = waitGroup.Wait()
	if err != nil {
		log.Fatal("error from wait group: %v", err)
	}
}

func runGrpcServer(
	ctx context.Context,
	waitGroup *errgroup.Group,
	config *config.GRPC,
	log *logger.ZeroLogger,
	resolver domain.CoinIdResolver,
	historicalPriceUC domain.HistoricalPriceUseCase,
	tenantSymbolUC domain.TenantSymbolUseCase,
) {
	server := grpcserver.NewPriceServer(log, resolver, historicalPriceUC, tenantSymbolUC)

	// Place for middleware injection
	// grpcLogger := grpc.UnaryInterceptor(gapi.GrpcLogger)
	grpcServer := grpc.NewServer()
	pricev1.RegisterPriceServer(grpcServer, server)
	reflection.Register(grpcServer)

	hs := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, hs)
	hs.SetServingStatus("price.v1.Price", healthpb.HealthCheckResponse_SERVING)

	listener, err := net.Listen("tcp", config.Addr)
	if err != nil {
		log.Fatal("cannot create listener on %s: %v", config.Addr, err)
	}

	waitGroup.Go(func() error {
		log.Info("start gRPC server at %s", listener.Addr().String())

		err = grpcServer.Serve(listener)
		if err != nil {
			if errors.Is(err, grpc.ErrServerStopped) {
				return nil
			}
			log.Error("gRPC server failed to serve: %v", err)
			return err
		}

		return nil
	})

	waitGroup.Go(func() error {
		<-ctx.Done()
		log.Info("graceful shutdown gRPC server")

		grpcServer.GracefulStop()
		log.Info("gRPC server is stopped")

		return nil
	})
}
