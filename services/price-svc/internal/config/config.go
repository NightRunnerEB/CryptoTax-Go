package config

import (
	"fmt"
	"os"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/coingecko"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type (
	Config struct {
		App      App                `yaml:"app"`
		Log      Log                `yaml:"log"`
		PG       PG                 `yaml:"postgres"`
		GRPC     GRPC               `yaml:"grpc"`
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
)

func NewConfig() (*Config, error) {
	if os.Getenv("APP_ENV") != "prod" {
		_ = godotenv.Load()
	}

	var cfg Config

	if err := cleanenv.ReadConfig("config.yaml", &cfg); err != nil {
		return nil, fmt.Errorf("read config.yaml: %w", err)
	}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("read env: %w", err)
	}

	return &cfg, nil
}
