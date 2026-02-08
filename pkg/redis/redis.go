package redis

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	Close() error

	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	// DelAll(ctx context.Context, pattern string) error
}

type Redis struct {
	client *redis.Client
	jitter time.Duration
}

func New(ctx context.Context, url string, jitter time.Duration, opts ...Option) (Cache, error) {
	redisConfig, err := redis.ParseURL(url)

	if err != nil {
		return nil, fmt.Errorf("redis parse url failed: %w", err)
	}

	for _, opt := range opts {
		opt(redisConfig)
	}

	client := redis.NewClient(redisConfig)

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &Redis{client: client, jitter: jitter}, nil
}

func (r *Redis) Close() error {
	return r.client.Close()
}

func (r *Redis) Get(ctx context.Context, key string) (string, bool, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}

	return val, true, nil
}

func (r *Redis) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	ttl = ttlWithJitter(ttl, r.jitter)
	if err := r.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return err
	}

	return nil
}

func (r *Redis) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	if err := r.client.Del(ctx, keys...).Err(); err != nil {
		return err
	}

	return nil
}

// ttlJitterRandom returns base TTL + random up to maxJitter.
// Example usage: base=5min, maxJitter=30s
func ttlWithJitter(base, maxJitter time.Duration) time.Duration {
	if maxJitter <= 0 {
		return base
	}
	delta := time.Duration(rand.Int63n(int64(maxJitter)))
	return base + delta
}

// Note: Use carefully in prod
// func (r *Redis) DelAll(ctx context.Context, pattern string) error {
// 	if r.client == nil {
// 		return fmt.Errorf("redis client is not initialized (call Open)")
// 	}
// 	if pattern == "" {
// 		return fmt.Errorf("pattern is empty")
// 	}

// 	const batchSize = 500

// 	var (
// 		cursor  uint64
// 		deleted int
// 	)

// 	for {
// 		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, batchSize).Result()
// 		if err != nil {
// 			return fmt.Errorf("redis scan failed: %w", err)
// 		}
// 		cursor = nextCursor

// 		if len(keys) > 0 {
// 			if err := r.client.Del(ctx, keys...).Err(); err != nil && err != redis.Nil {
// 				return fmt.Errorf("redis del failed: %w", err)
// 			}
// 			deleted += len(keys)
// 		}

// 		if cursor == 0 {
// 			break
// 		}
// 	}

// 	r.log.Info("redis.DelAll: pattern=%s deleted=%d", pattern, deleted)
// 	return nil
// }
