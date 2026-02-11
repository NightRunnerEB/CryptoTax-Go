package telemetry

import (
	"context"
	"errors"
	"time"

	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"
)

type Config struct {
	ServiceName                 string
	ServiceVersion              string
	Environment                 string
	OTLPEndpoint                string
	Insecure                    bool
	MetricsExportInterval       time.Duration
	RuntimeReadMemStatsInterval time.Duration
}

type Providers struct {
	LogProvider   *sdklog.LoggerProvider
	MeterProvider *sdkmetric.MeterProvider
	Resource      *resource.Resource
}

func Init(ctx context.Context, cfg Config) (*Providers, error) {
	applyDefaults(&cfg)

	res, err := newResource(ctx, cfg)
	if err != nil {
		return nil, err
	}

	meterProvider, err := newMeterProvider(ctx, cfg, res)
	if err != nil {
		return nil, err
	}
	otel.SetMeterProvider(meterProvider)

	if err := runtimemetrics.Start(
		runtimemetrics.WithMinimumReadMemStatsInterval(cfg.RuntimeReadMemStatsInterval),
	); err != nil {
		return nil, err
	}

	logProvider, err := newLoggerProvider(ctx, cfg, res)
	if err != nil {
		return nil, err
	}
	global.SetLoggerProvider(logProvider)

	// Если входящий запрос содержит traceparent/tracestate и baggage, то otelgrpc их корректно распарсит и продолжит контекст
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Providers{
		LogProvider:   logProvider,
		MeterProvider: meterProvider,
		Resource:      res,
	}, nil
}

func (p *Providers) GracefulStop(ctx context.Context) error {
	if p == nil {
		return nil
	}

	var errs []error
	if p.LogProvider != nil {
		if err := p.LogProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if p.MeterProvider != nil {
		if err := p.MeterProvider.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func applyDefaults(cfg *Config) {
	if cfg.OTLPEndpoint == "" {
		cfg.OTLPEndpoint = "otel-collector:4317"
	}
	if cfg.MetricsExportInterval == 0 {
		cfg.MetricsExportInterval = 5 * time.Second
	}
	if cfg.RuntimeReadMemStatsInterval == 0 {
		cfg.RuntimeReadMemStatsInterval = 10 * time.Second
	}
}

func newResource(ctx context.Context, cfg Config) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{}
	if cfg.ServiceName != "" {
		attrs = append(attrs, semconv.ServiceName(cfg.ServiceName))
	}
	if cfg.ServiceVersion != "" {
		attrs = append(attrs, semconv.ServiceVersion(cfg.ServiceVersion))
	}
	if cfg.Environment != "" {
		attrs = append(attrs, attribute.String("deployment.environment", cfg.Environment))
	}

	return resource.New(
		ctx,
		resource.WithFromEnv(), // можно читать все аттрибуты из ENV
		resource.WithAttributes(attrs...),
	)
}

func newMeterProvider(ctx context.Context, cfg Config, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	metricExporterOptions := []otlpmetricgrpc.Option{
		// otlpmetricgrpc.WithTemporalitySelector(deltaTemporality) - темпоральностью агрегации: cumulative или delta.
		// В нашем случае по дефолту будет стоять DefaultTemporalitySelector - cumulative для всех InstrumentKind
		otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.Insecure {
		metricExporterOptions = append(metricExporterOptions, otlpmetricgrpc.WithInsecure())
	}

	metricExporter, err := otlpmetricgrpc.New(ctx, metricExporterOptions...)
	if err != nil {
		return nil, err
	}

	metricReader := sdkmetric.NewPeriodicReader(
		metricExporter,
		sdkmetric.WithInterval(cfg.MetricsExportInterval),
	)

	return sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(metricReader),
		sdkmetric.WithResource(res),
	), nil
}

func newLoggerProvider(ctx context.Context, cfg Config, res *resource.Resource) (*sdklog.LoggerProvider, error) {
	logExporterOptions := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.Insecure {
		logExporterOptions = append(logExporterOptions, otlploggrpc.WithInsecure())
	}

	logExporter, err := otlploggrpc.New(ctx, logExporterOptions...)
	if err != nil {
		return nil, err
	}

	return sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)), // NewBatchProcessor создается со всеми дефолтными Options
	), nil
}
