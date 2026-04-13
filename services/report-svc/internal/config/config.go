package config

import (
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"

	apperr "github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain/error"
)

type (
	Config struct {
		App    AppConfig    `yaml:"app"`
		Log    LogConfig    `yaml:"log"`
		OTel   OTelConfig   `yaml:"otel"`
		GRPC   GRPCConfig   `yaml:"grpc"`
		PG     PGConfig     `yaml:"postgres"`
		Rabbit RabbitConfig `yaml:"rabbitmq"`
		MinIO  MinIOConfig  `yaml:"minio"`
		Worker WorkerConfig `yaml:"worker"`
	}

	AppConfig struct {
		Name    string `yaml:"name"`
		Env     string `yaml:"env"`
		Version string `env:"APP_VERSION" env-required:"true"`
	}

	LogConfig struct {
		Level string `yaml:"level"`
	}

	OTelConfig struct {
		Endpoint                    string        `yaml:"endpoint"`
		Insecure                    bool          `yaml:"insecure"`
		MetricsExportInterval       time.Duration `yaml:"metrics_export_interval"`
		RuntimeReadMemStatsInterval time.Duration `yaml:"runtime_read_mem_stats_interval"`
	}

	GRPCConfig struct {
		Addr string `yaml:"addr"`
	}

	PGConfig struct {
		URL            string        `env:"DATABASE_URL" env-required:"true"`
		MaxConns       int           `yaml:"max_conns"`
		ConnTimeout    time.Duration `yaml:"conn_timeout"`
		AttemptTimeout time.Duration `yaml:"attempt_timeout"`
		ConnAttempts   int           `yaml:"conn_attempts"`
	}

	RabbitConfig struct {
		URL                         string        `env:"RABBIT_URL" env-required:"true"`
		Exchange                    string        `yaml:"exchange"`
		QueueRenderRequested        string        `yaml:"queue_render_requested"`
		RoutingRenderRequest        string        `yaml:"routing_render_request"`
		RoutingRendered             string        `yaml:"routing_rendered"`
		RoutingRenderFailed         string        `yaml:"routing_render_failed"`
		ConsumerNameRenderRequested string        `yaml:"consumer_name_render_requested"`
		Prefetch                    int           `yaml:"prefetch"`
		Concurrency                 int           `yaml:"concurrency"`
		ReconnectInterval           time.Duration `yaml:"reconnect_interval"`
		OutboxBatchSize             int32         `yaml:"outbox_batch_size"`
		OutboxPollInterval          time.Duration `yaml:"outbox_poll_interval"`
		OutboxMaxAttempts           int32         `yaml:"outbox_max_attempts"`
		HandlerTimeout              time.Duration `yaml:"handler_timeout"`
		QueueDurable                bool          `yaml:"queue_durable"`
		SkipQueueDeclare            bool          `yaml:"skip_queue_declare"`
	}

	MinIOConfig struct {
		Endpoint  string `yaml:"endpoint"`
		AccessKey string `env:"MINIO_ACCESS_KEY" env-required:"true"`
		SecretKey string `env:"MINIO_SECRET_KEY" env-required:"true"`
		Bucket    string `yaml:"bucket"`
		UseSSL    bool   `yaml:"use_ssl"`
	}

	WorkerConfig struct {
		TemplateVersion  string `yaml:"template_version"`
		MaxPreviewEvents int    `yaml:"max_preview_events"`
	}
)

func NewConfig(path string) (*Config, error) {
	if os.Getenv("APP_ENV") != "prod" {
		_ = godotenv.Load()
	}

	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, apperr.Internal("read config file failed", err, map[string]string{
			"file": path,
		})
	}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, apperr.Internal("read environment failed", err, nil)
	}

	applyDefaults(&cfg)
	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.GRPC.Addr == "" {
		cfg.GRPC.Addr = "0.0.0.0:8098"
	}
	if cfg.OTel.Endpoint == "" {
		cfg.OTel.Endpoint = "otel-collector:4317"
	}
	if cfg.OTel.MetricsExportInterval == 0 {
		cfg.OTel.MetricsExportInterval = 10 * time.Second
	}
	if cfg.OTel.RuntimeReadMemStatsInterval == 0 {
		cfg.OTel.RuntimeReadMemStatsInterval = 10 * time.Second
	}

	if cfg.PG.MaxConns <= 0 {
		cfg.PG.MaxConns = 10
	}
	if cfg.PG.ConnTimeout == 0 {
		cfg.PG.ConnTimeout = 2 * time.Second
	}
	if cfg.PG.AttemptTimeout == 0 {
		cfg.PG.AttemptTimeout = 5 * time.Second
	}
	if cfg.PG.ConnAttempts <= 0 {
		cfg.PG.ConnAttempts = 3
	}

	if cfg.Rabbit.Exchange == "" {
		cfg.Rabbit.Exchange = "tax.pipeline"
	}
	if cfg.Rabbit.QueueRenderRequested == "" {
		cfg.Rabbit.QueueRenderRequested = "report.render.requested"
	}
	if cfg.Rabbit.RoutingRenderRequest == "" {
		cfg.Rabbit.RoutingRenderRequest = "ReportRenderRequested"
	}
	if cfg.Rabbit.RoutingRendered == "" {
		cfg.Rabbit.RoutingRendered = "ReportRendered"
	}
	if cfg.Rabbit.RoutingRenderFailed == "" {
		cfg.Rabbit.RoutingRenderFailed = "ReportRenderFailed"
	}
	if cfg.Rabbit.ConsumerNameRenderRequested == "" {
		cfg.Rabbit.ConsumerNameRenderRequested = "report-render-requested-consumer"
	}
	if cfg.Rabbit.Prefetch <= 0 {
		cfg.Rabbit.Prefetch = 10
	}
	if cfg.Rabbit.Concurrency <= 0 {
		cfg.Rabbit.Concurrency = 1
	}
	if cfg.Rabbit.ReconnectInterval == 0 {
		cfg.Rabbit.ReconnectInterval = 2 * time.Second
	}
	if cfg.Rabbit.OutboxBatchSize <= 0 {
		cfg.Rabbit.OutboxBatchSize = 100
	}
	if cfg.Rabbit.OutboxPollInterval == 0 {
		cfg.Rabbit.OutboxPollInterval = time.Second
	}
	if cfg.Rabbit.OutboxMaxAttempts <= 0 {
		cfg.Rabbit.OutboxMaxAttempts = 10
	}
	if cfg.Rabbit.HandlerTimeout == 0 {
		cfg.Rabbit.HandlerTimeout = time.Minute
	}

	if cfg.Worker.TemplateVersion == "" {
		cfg.Worker.TemplateVersion = "v1"
	}
	if cfg.Worker.MaxPreviewEvents <= 0 {
		cfg.Worker.MaxPreviewEvents = 20
	}
}
