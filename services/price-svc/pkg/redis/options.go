package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Option func(*redis.Options)

// IMPORTANT:
//
//	WithURL MUST be applied BEFORE any other Option.
//	Applying WithURL after other options will overwrite previously configured values.
func WithURL(url string) Option {
	return func(opt *redis.Options) {
		parsed, err := redis.ParseURL(url)
		if err != nil {
			panic(fmt.Errorf("redis.ParseURL failed: %w", err))
		}

		*opt = *parsed
	}
}

func WithAddr(addr string) Option {
	return func(opt *redis.Options) {
		opt.Addr = addr
	}
}

func WithPassword(password string) Option {
	return func(opt *redis.Options) {
		opt.Password = password
	}
}

func WithDB(db int) Option {
	return func(opt *redis.Options) {
		opt.DB = db
	}
}

func WithPoolSize(n int) Option {
	return func(opt *redis.Options) {
		opt.PoolSize = n
	}
}

func WithMinIdleConns(n int) Option {
	return func(opt *redis.Options) {
		opt.MinIdleConns = n
	}
}

func WithDialTimeout(d time.Duration) Option {
	return func(opt *redis.Options) {
		opt.DialTimeout = d
	}
}

func WithReadTimeout(d time.Duration) Option {
	return func(opt *redis.Options) {
		opt.ReadTimeout = d
	}
}

func WithWriteTimeout(d time.Duration) Option {
	return func(opt *redis.Options) {
		opt.WriteTimeout = d
	}
}

func WithPoolTimeout(d time.Duration) Option {
	return func(opt *redis.Options) {
		opt.PoolTimeout = d
	}
}

func WithMaxRetries(n int) Option {
	return func(opt *redis.Options) {
		opt.MaxRetries = n
	}
}

func WithMinRetryBackoff(d time.Duration) Option {
	return func(opt *redis.Options) {
		opt.MinRetryBackoff = d
	}
}

func WithMaxRetryBackoff(d time.Duration) Option {
	return func(opt *redis.Options) {
		opt.MaxRetryBackoff = d
	}
}

func WithOnConnect(fn func(ctx context.Context, cn *redis.Conn) error) Option {
	return func(opt *redis.Options) {
		opt.OnConnect = fn
	}
}
