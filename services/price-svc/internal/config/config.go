package config

import (
	"fmt"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type (
	Config struct {
		App  App
		Log  Log
		PG   PG
		GRPC GRPC
	}

	App struct {
		Name    string `env:"APP_NAME" envDefault:"price-svc"`
		Version string `env:"APP_VERSION,required"`
		Env     string `env:"APP_ENV" envDefault:"dev"`
	}

	PG struct {
		URL            string        `env:"DATABASE_URL,required"`
		PoolMax        int           `env:"DB_MAX_CONNS" envDefault:"10"`
		ConnectTimeout time.Duration `env:"DB_CONN_TIMEOUT" envDefault:"3s"`
		AttemptTimeout time.Duration `env:"DB_ATTEMPT_TIMEOUT" envDefault:"1s"`
	}

	RedisConfig struct {
		RedisURL string        `env:"REDIS_URL,required"`
		PoolMax  int           `env:"REDIS_POOL_MAX" envDefault:"4"`
		Skew     time.Duration `env:"REDIS_SKEW_SECS" envDefault:"5s"`
	}

	GRPC struct {
		Addr string `env:"GRPC_ADDR" envDefault:":8091"`
	}

	Log struct {
		Level string `env:"LOG_LEVEL" envDefault:"info"`
	}

	// Metrics struct {
	// 	Enabled bool `env:"METRICS_ENABLED" envDefault:"true"`
	// }

	// Swagger struct {
	// 	Enabled bool `env:"SWAGGER_ENABLED" envDefault:"false"`
	// }
)

func NewConfig() (*Config, error) {
	if os.Getenv("APP_ENV") != "prod" {
		if err := godotenv.Load(); err != nil {
			return nil, fmt.Errorf("failed to load .env file: %w", err)
		}
	}

	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}
