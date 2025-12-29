package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/pkg/logger"
	"github.com/redis/go-redis/v9"
)

const (
	_defaultAddr = "127.0.0.1:6379"
	_defaultDB   = 0

	_defaultPoolSize     = 10
	_defaultMinIdleConns = 2

	_defaultDialTimeout  = 3 * time.Second
	_defaultReadTimeout  = 2 * time.Second
	_defaultWriteTimeout = 2 * time.Second
	_defaultPoolTimeout  = 2 * time.Second

	_defaultMaxRetries      = 3
	_defaultMinRetryBackoff = 100 * time.Millisecond
	_defaultMaxRetryBackoff = 1 * time.Second
)

type Cache interface {
	Open(ctx context.Context) error
	Close() error

	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	DelAll(ctx context.Context, pattern string) error
}

type Redis struct {
	client *redis.Client
	log    logger.Logger
	opt    redis.Options
}

func defaultOptions() redis.Options {
	return redis.Options{
		Addr: _defaultAddr,
		DB:   _defaultDB,

		PoolSize:     _defaultPoolSize,
		MinIdleConns: _defaultMinIdleConns,

		DialTimeout:  _defaultDialTimeout,
		ReadTimeout:  _defaultReadTimeout,
		WriteTimeout: _defaultWriteTimeout,
		PoolTimeout:  _defaultPoolTimeout,

		MaxRetries:      _defaultMaxRetries,
		MinRetryBackoff: _defaultMinRetryBackoff,
		MaxRetryBackoff: _defaultMaxRetryBackoff,
	}
}

func New(log logger.Logger, opts ...Option) Cache {
	redisConfig := defaultOptions()

	for _, opt := range opts {
		opt(&redisConfig)
	}

	return &Redis{
		opt: redisConfig,
		log: log,
	}
}

func (r *Redis) Open(ctx context.Context) error {
	if r.client != nil {
		// If injected client exists
		if err := r.client.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("redis ping failed: %w", err)
		}
		r.log.Info("redis connected (injected client)")
		return nil
	}

	if r.opt.Addr == "" {
		return fmt.Errorf("redis addr is empty (use WithAddr or WithClient)")
	}

	r.client = redis.NewClient(&r.opt)

	if err := r.client.Ping(ctx).Err(); err != nil {
		_ = r.client.Close()
		r.client = nil
		return fmt.Errorf("redis ping failed: %w", err)
	}

	r.log.Info("redis connected: addr=%s db=%d pool=%d", r.opt.Addr, r.opt.DB, r.opt.PoolSize)
	return nil
}

func (r *Redis) Close() error {
	if r.client == nil {
		return nil
	}

	err := r.client.Close()
	r.client = nil
	return err
}

func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	if r.client == nil {
		return "", fmt.Errorf("redis client is not initialized (call Open)")
	}

	res, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return res, nil
}

func (r *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if r.client == nil {
		return fmt.Errorf("redis.Set: client is not initialized (call Open)")
	}

	if err := r.client.Set(ctx, key, value, expiration).Err(); err != nil {
		r.log.Error("redis.Set: %s", err)
		return err
	}

	return nil
}

func (r *Redis) Del(ctx context.Context, keys ...string) error {
	if r.client == nil {
		return fmt.Errorf("redis.Del: client is not initialized (call Open)")
	}

	if err := r.client.Del(ctx, keys...).Err(); err != nil {
		r.log.Error("redis.Del: %s", err)
		return err
	}

	return nil
}

// Note: Use carefully in prod
func (r *Redis) DelAll(ctx context.Context, pattern string) error {
	if r.client == nil {
		return fmt.Errorf("redis client is not initialized (call Open)")
	}
	if pattern == "" {
		return fmt.Errorf("pattern is empty")
	}

	const batchSize = 500

	var (
		cursor  uint64
		deleted int
	)

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, batchSize).Result()
		if err != nil {
			return fmt.Errorf("redis scan failed: %w", err)
		}
		cursor = nextCursor

		if len(keys) > 0 {
			if err := r.client.Del(ctx, keys...).Err(); err != nil && err != redis.Nil {
				return fmt.Errorf("redis del failed: %w", err)
			}
			deleted += len(keys)
		}

		if cursor == 0 {
			break
		}
	}

	r.log.Info("redis.DelAll: pattern=%s deleted=%d", pattern, deleted)
	return nil
}
