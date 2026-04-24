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
		App   AppConfig   `yaml:"app"`
		Log   LogConfig   `yaml:"log"`
		OTel  OTelConfig  `yaml:"otel"`
		GRPC  GRPCConfig  `yaml:"grpc"`
		HTTP  HTTPConfig  `yaml:"http"`
		MinIO MinIOConfig `yaml:"minio"`
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

	HTTPConfig struct {
		Addr            string        `yaml:"addr"`
		ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	}

	MinIOConfig struct {
		Endpoint  string `yaml:"endpoint" env:"MINIO_ENDPOINT" env-required:"true"`
		AccessKey string `env:"MINIO_ACCESS_KEY" env-required:"true"`
		SecretKey string `env:"MINIO_SECRET_KEY" env-required:"true"`
		Bucket    string `yaml:"bucket"`
		UseSSL    bool   `yaml:"use_ssl"`
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
	if cfg.HTTP.Addr == "" {
		cfg.HTTP.Addr = "0.0.0.0:8099"
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

}
