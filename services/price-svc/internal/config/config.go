package config

import (
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/coingecko"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

type (
	Config struct {
		App      App                `yaml:"app"`
		Log      Log                `yaml:"log"`
		OTel     OTelConfig         `yaml:"otel"`
		PG       PG                 `yaml:"postgres"`
		GRPC     GRPC               `yaml:"grpc"`
		HTTP     HTTP               `yaml:"http"`
		Redis    Redis              `yaml:"redis"`
		CG       coingecko.CGConfig `yaml:"coingecko"`
		Resolver Resolver           `yaml:"resolver"`
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

	Resolver struct {
		Path string `yaml:"path"`
	}

	OTelConfig struct {
		Endpoint                    string        `yaml:"endpoint" env:"OTEL_EXPORTER_OTLP_ENDPOINT" env-required:"true"`
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
	if cfg.HTTP.Addr == "" {
		cfg.HTTP.Addr = "0.0.0.0:8092"
	}
	if cfg.HTTP.ShutdownTimeout == 0 {
		cfg.HTTP.ShutdownTimeout = 5 * time.Second
	}
}
