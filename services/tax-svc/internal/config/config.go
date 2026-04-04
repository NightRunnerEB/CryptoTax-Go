package config

import (
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"

	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
)

type (
	Config struct {
		App         AppConfig         `yaml:"app"`
		Log         LogConfig         `yaml:"log"`
		OTel        OTelConfig        `yaml:"otel"`
		GRPC        GRPCConfig        `yaml:"grpc"`
		HTTP        HTTPConfig        `yaml:"http"`
		PG          PGConfig          `yaml:"postgres"`
		Aggregation AggregationConfig `yaml:"aggregation"`
		Report      ReportConfig      `yaml:"report"`
		MinIO       MinIOConfig       `yaml:"minio"`
		Worker      WorkerConfig      `yaml:"worker"`
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
		Endpoint                    string        `yaml:"endpoint" env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
		Insecure                    bool          `yaml:"insecure" env:"OTEL_EXPORTER_OTLP_INSECURE"`
		MetricsExportInterval       time.Duration `yaml:"metrics_export_interval" env:"OTEL_METRICS_EXPORT_INTERVAL"`
		RuntimeReadMemStatsInterval time.Duration `yaml:"runtime_read_mem_stats_interval" env:"OTEL_RUNTIME_READ_MEM_STATS_INTERVAL"`
	}

	GRPCConfig struct {
		Addr string `yaml:"addr"`
	}

	HTTPConfig struct {
		Addr            string        `yaml:"addr"`
		ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	}

	PGConfig struct {
		URL            string        `env:"DATABASE_URL" env-required:"true"`
		MaxConns       int           `yaml:"max_conns"`
		ConnTimeout    time.Duration `yaml:"conn_timeout"`
		AttemptTimeout time.Duration `yaml:"attempt_timeout"`
		ConnAttempts   int           `yaml:"conn_attempts"`
	}

	AggregationConfig struct {
		Addr    string        `yaml:"addr" env:"AGGREGATION_SVC_ADDR" env-required:"true"`
		Timeout time.Duration `yaml:"timeout"`
	}

	ReportConfig struct {
		Addr    string        `yaml:"addr" env:"REPORT_SVC_ADDR"`
		Timeout time.Duration `yaml:"timeout"`
	}

	MinIOConfig struct {
		Endpoint       string        `yaml:"endpoint" env:"MINIO_ENDPOINT" env-required:"true"`
		AccessKey      string        `env:"MINIO_ACCESS_KEY" env-required:"true"`
		SecretKey      string        `env:"MINIO_SECRET_KEY" env-required:"true"`
		Bucket         string        `yaml:"bucket"`
		UseSSL         bool          `yaml:"use_ssl"`
		PresignTTL     time.Duration `yaml:"presign_ttl"`
		RequestTimeout time.Duration `yaml:"request_timeout"`
		RetryMax       int           `yaml:"retry_max"`
		RetryBaseDelay time.Duration `yaml:"retry_base_delay"`
		RetryMaxDelay  time.Duration `yaml:"retry_max_delay"`
	}

	WorkerConfig struct {
		PollInterval     time.Duration `yaml:"poll_interval"`
		IdleSleep        time.Duration `yaml:"idle_sleep"`
		RetryMaxAttempts int           `yaml:"retry_max_attempts"`
		RetryBaseDelay   time.Duration `yaml:"retry_base_delay"`
		RetryMaxDelay    time.Duration `yaml:"retry_max_delay"`
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
		cfg.GRPC.Addr = "0.0.0.0:8096"
	}
	if cfg.HTTP.Addr == "" {
		cfg.HTTP.Addr = "0.0.0.0:8097"
	}
	if cfg.HTTP.ShutdownTimeout == 0 {
		cfg.HTTP.ShutdownTimeout = 5 * time.Second
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

	if cfg.Aggregation.Timeout == 0 {
		cfg.Aggregation.Timeout = 10 * time.Second
	}
	if cfg.Report.Addr == "" {
		cfg.Report.Addr = "127.0.0.1:8098"
	}
	if cfg.Report.Timeout == 0 {
		cfg.Report.Timeout = 10 * time.Second
	}

	if cfg.MinIO.PresignTTL == 0 {
		cfg.MinIO.PresignTTL = 15 * time.Minute
	}
	if cfg.MinIO.RequestTimeout == 0 {
		cfg.MinIO.RequestTimeout = 10 * time.Second
	}
	if cfg.MinIO.RetryMax <= 0 {
		cfg.MinIO.RetryMax = 3
	}
	if cfg.MinIO.RetryBaseDelay <= 0 {
		cfg.MinIO.RetryBaseDelay = 200 * time.Millisecond
	}
	if cfg.MinIO.RetryMaxDelay <= 0 {
		cfg.MinIO.RetryMaxDelay = 2 * time.Second
	}
	if cfg.MinIO.RetryMaxDelay < cfg.MinIO.RetryBaseDelay {
		cfg.MinIO.RetryMaxDelay = cfg.MinIO.RetryBaseDelay
	}
	if cfg.Worker.PollInterval <= 0 {
		cfg.Worker.PollInterval = 2 * time.Second
	}
	if cfg.Worker.IdleSleep <= 0 {
		cfg.Worker.IdleSleep = 500 * time.Millisecond
	}
	if cfg.Worker.RetryMaxAttempts <= 0 {
		cfg.Worker.RetryMaxAttempts = 3
	}
	if cfg.Worker.RetryBaseDelay <= 0 {
		cfg.Worker.RetryBaseDelay = 10 * time.Second
	}
	if cfg.Worker.RetryMaxDelay <= 0 {
		cfg.Worker.RetryMaxDelay = 2 * time.Minute
	}
}
