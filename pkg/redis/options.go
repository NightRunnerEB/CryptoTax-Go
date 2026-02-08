package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Option func(*redis.Options)

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
