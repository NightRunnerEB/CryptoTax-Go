package config

import (
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"

	apperr "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain/error"
)

type (
	Config struct {
		App         App         `yaml:"app"`
		Log         Log         `yaml:"log"`
		OTel        OTelConfig  `yaml:"otel"`
		GRPC        GRPC        `yaml:"grpc"`
		HTTP        HTTP        `yaml:"http"`
		PG          PG          `yaml:"postgres"`
		Redis       Redis       `yaml:"redis"`
		Ledger      Ledger      `yaml:"ledger"`
		Price       Price       `yaml:"price"`
		RabbitMQ    RabbitMQ    `yaml:"rabbitmq"`
		Aggregation Aggregation `yaml:"aggregation"`
	}

	App struct {
		Name    string `yaml:"name"`
		Env     string `yaml:"env"`
		Version string `env:"APP_VERSION" env-required:"true"`
	}

	Log struct {
		Level string `yaml:"level"`
	}

	GRPC struct {
		Addr string `yaml:"addr"`
	}

	HTTP struct {
		Addr            string        `yaml:"addr"`
		ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	}

	PG struct {
		URL            string        `env:"DATABASE_URL" env-required:"true"`
		PoolMax        int           `yaml:"pool_max"`
		ConnectTimeout time.Duration `yaml:"conn_timeout"`
		AttemptTimeout time.Duration `yaml:"attempt_timeout"`
		ConnAttempts   int           `yaml:"conn_attempts"`
	}

	Redis struct {
		RedisURL string        `env:"REDIS_URL" env-required:"true"`
		PoolMax  int           `yaml:"pool_max"`
		Jitter   time.Duration `yaml:"jitter"`
	}

	Ledger struct {
		BaseURL string        `yaml:"base_url" env:"LEDGER_SVC_BASE_URL" env-required:"true"`
		Timeout time.Duration `yaml:"timeout"`
	}

	Price struct {
		Addr      string        `yaml:"addr" env:"PRICE_SVC_ADDR" env-required:"true"`
		Timeout   time.Duration `yaml:"timeout"`
		BatchSize int           `yaml:"batch_size"`
	}

	RabbitMQ struct {
		URL               string        `yaml:"url" env:"RABBITMQ_URL" env-required:"true"`
		Exchange          string        `yaml:"exchange"`
		Queue             string        `yaml:"queue"`
		QueueDurable      bool          `yaml:"queue_durable"`
		RoutingKey        string        `yaml:"routing_key"`
		ConsumerName      string        `yaml:"consumer_name"`
		Prefetch          int           `yaml:"prefetch"`
		Concurrency       int           `yaml:"concurrency"`
		ReconnectInterval time.Duration `yaml:"reconnect_interval"`
	}

	Aggregation struct {
		DefaultFiatCurrency string        `yaml:"default_fiat_currency"`
		DefaultTimezone     string        `yaml:"default_timezone"`
		ImportLockTTL       time.Duration `yaml:"import_lock_ttl"`
	}

	OTelConfig struct {
		Endpoint                    string        `yaml:"endpoint" env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
		Insecure                    bool          `yaml:"insecure" env:"OTEL_EXPORTER_OTLP_INSECURE"`
		MetricsExportInterval       time.Duration `yaml:"metrics_export_interval" env:"OTEL_METRICS_EXPORT_INTERVAL"`
		RuntimeReadMemStatsInterval time.Duration `yaml:"runtime_read_mem_stats_interval" env:"OTEL_RUNTIME_READ_MEM_STATS_INTERVAL"`
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
	if cfg.HTTP.ShutdownTimeout == 0 {
		cfg.HTTP.ShutdownTimeout = 5 * time.Second
	}
	if cfg.Price.BatchSize == 0 {
		cfg.Price.BatchSize = 10000
	}
	if cfg.Aggregation.DefaultFiatCurrency == "" {
		cfg.Aggregation.DefaultFiatCurrency = "rub"
	}
	if cfg.Aggregation.DefaultTimezone == "" {
		cfg.Aggregation.DefaultTimezone = "Europe/Moscow"
	}
	if cfg.Aggregation.ImportLockTTL == 0 {
		cfg.Aggregation.ImportLockTTL = 10 * time.Minute
	}
	if cfg.Ledger.Timeout == 0 {
		cfg.Ledger.Timeout = 10 * time.Second
	}
	if cfg.Price.Timeout == 0 {
		cfg.Price.Timeout = 10 * time.Second
	}
	if cfg.RabbitMQ.Exchange == "" {
		cfg.RabbitMQ.Exchange = "ledger.events"
	}
	if cfg.RabbitMQ.Queue == "" {
		cfg.RabbitMQ.Queue = "aggregation.import.completed"
	}
	if cfg.RabbitMQ.RoutingKey == "" {
		cfg.RabbitMQ.RoutingKey = "ImportCompleted"
	}
	if cfg.RabbitMQ.ConsumerName == "" {
		cfg.RabbitMQ.ConsumerName = "aggregation-import-completed-consumer"
	}
	if cfg.RabbitMQ.Prefetch <= 0 {
		cfg.RabbitMQ.Prefetch = 10
	}
	if cfg.RabbitMQ.Concurrency <= 0 {
		cfg.RabbitMQ.Concurrency = 1
	}
	if cfg.RabbitMQ.ReconnectInterval == 0 {
		cfg.RabbitMQ.ReconnectInterval = 2 * time.Second
	}
}
