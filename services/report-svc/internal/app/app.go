package app

import (
	"context"
	"errors"
	"net"
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

	"github.com/NightRunner/CryptoTax-Go/pkg/logger"
	"github.com/NightRunner/CryptoTax-Go/pkg/postgres"
	"github.com/NightRunner/CryptoTax-Go/pkg/telemetry"
	db "github.com/NightRunner/CryptoTax-Go/services/report-svc/db/sqlc"
	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/clients/rabbit"
	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/config"
	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/consumer"
	repository "github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/infra/repo"
	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/infra/storage"
	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/interceptors"
	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/pdf"
	usecase "github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/usecases"
	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/worker"
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

	objectStorage, err := storage.NewMinIOStorage(cfg.MinIO)
	if err != nil {
		log.Fatal("cannot create storage client", zap.Error(err))
	}

	eventPublisher, err := rabbit.NewPublisher(cfg.Rabbit)
	if err != nil {
		log.Fatal("cannot create rabbit publisher", zap.Error(err))
	}
	defer eventPublisher.Close()

	inboxRepo := repository.NewInboxRepo(store)
	outboxRepo := repository.NewOutboxRepo(store)
	renderJobRepo := repository.NewRenderJobRepo(store)
	pdfGenerator := pdf.NewSimpleGenerator()

	renderUC := usecase.NewRenderUC(
		store,
		inboxRepo,
		renderJobRepo,
		objectStorage,
		pdfGenerator,
		cfg.Worker.TemplateVersion,
		cfg.Worker.MaxPreviewEvents,
	)

	renderConsumer := consumer.NewReportRenderRequestedConsumer(cfg.Rabbit, renderUC, log)
	outboxDispatcher := worker.NewOutboxDispatcher(cfg.Rabbit, outboxRepo, eventPublisher, log)

	statsHandler := otelgrpc.NewServerHandler(
		otelgrpc.WithTracerProvider(noop.NewTracerProvider()),
		otelgrpc.WithMeterProvider(telemetryProviders.MeterProvider),
	)
	runGrpcServer(ctx, waitGroup, cfg, statsHandler, telemetryProviders)

	waitGroup.Go(func() error {
		return outboxDispatcher.Start(ctx)
	})
	waitGroup.Go(func() error {
		return renderConsumer.Start(ctx)
	})
	waitGroup.Go(func() error {
		<-ctx.Done()
		var joinErr error
		joinErr = errors.Join(joinErr, renderConsumer.Close())
		joinErr = errors.Join(joinErr, eventPublisher.Close())
		return joinErr
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

	reflection.Register(grpcServer)

	hs := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, hs)
	hs.SetServingStatus("report.v1.Renderer", healthpb.HealthCheckResponse_SERVING)

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
