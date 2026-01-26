package coingecko

import "time"

type CGConfig struct {
	APIKey string `env:"COINGECKO_API_KEY" env-required:"true"`

	BaseURL           string            `yaml:"base_url"`
	Currency          string            `yaml:"currency"`
	RateLimitPerMin   int               `yaml:"rate_limit_per_min"`
	GranularityPolicy GranularityPolicy `yaml:"granularity_policy"`
}

type GranularityPolicy map[string]time.Duration
