package lock

import (
	"context"
	"fmt"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisLockManager struct {
	client *redis.Client
}

func NewRedisLockManager(client *redis.Client) *RedisLockManager {
	return &RedisLockManager{client: client}
}

func (m *RedisLockManager) AcquireImportLock(ctx context.Context, tenantID, importID uuid.UUID, ttl time.Duration) (bool, error) {
	if m == nil || m.client == nil {
		return false, fmt.Errorf("redis client is nil")
	}
	key := importLockKey(tenantID, importID)
	return m.client.SetNX(ctx, key, "1", ttl).Result()
}

func (m *RedisLockManager) ReleaseImportLock(ctx context.Context, tenantID, importID uuid.UUID) error {
	if m == nil || m.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	key := importLockKey(tenantID, importID)
	return m.client.Del(ctx, key).Err()
}

func importLockKey(tenantID, importID uuid.UUID) string {
	return fmt.Sprintf("agg:import:lock:%s:%s", tenantID.String(), importID.String())
}

var _ domain.LockManager = (*RedisLockManager)(nil)
