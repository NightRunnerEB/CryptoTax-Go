package lock

import (
	"context"
	"fmt"
	"time"

	pkgredis "github.com/NightRunner/CryptoTax-Go/pkg/redis"
	"github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"
	"github.com/google/uuid"
)

type RedisLockManager struct {
	client pkgredis.Cache
}

func NewRedisLockManager(client pkgredis.Cache) *RedisLockManager {
	return &RedisLockManager{client: client}
}

func (m *RedisLockManager) AcquireImportLock(ctx context.Context, tenantID, importID uuid.UUID, ttl time.Duration) (bool, error) {
	if m == nil || m.client == nil {
		return false, fmt.Errorf("redis client is nil")
	}
	key := importLockKey(tenantID, importID)
	return m.client.SetNX(ctx, key, "1", ttl)
}

func (m *RedisLockManager) ReleaseImportLock(ctx context.Context, tenantID, importID uuid.UUID) error {
	if m == nil || m.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	key := importLockKey(tenantID, importID)
	return m.client.Del(ctx, key)
}

func importLockKey(tenantID, importID uuid.UUID) string {
	return fmt.Sprintf("agg:import:lock:%s:%s", tenantID.String(), importID.String())
}

var _ domain.LockManager = (*RedisLockManager)(nil)
