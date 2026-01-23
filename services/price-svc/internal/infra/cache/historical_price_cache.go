package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	redis "github.com/NightRunner/CryptoTax-Go/services/price-svc/pkg/redis"
)

type PriceKey struct {
	CoinID      string
	BucketStart time.Time
}

func (k PriceKey) RedisKey() string {
	return fmt.Sprintf("hp:%s:%d",
		k.CoinID,
		k.BucketStart.UTC().Unix(),
	)
}

type HistoricalPriceCache struct {
	r redis.Cache
}

func NewHistoricalPriceCache(r redis.Cache) *HistoricalPriceCache {
	return &HistoricalPriceCache{r}
}

func (c *HistoricalPriceCache) Get(ctx context.Context, key PriceKey) (domain.HistoricalPrice, bool, error) {
	raw, ok, err := c.r.Get(ctx, key.RedisKey())
	if err != nil || !ok {
		return domain.HistoricalPrice{}, ok, err
	}

	var p domain.HistoricalPrice
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		_ = c.r.Del(ctx, key.RedisKey())
		return domain.HistoricalPrice{}, false, err
	}

	return p, true, nil
}

func (c *HistoricalPriceCache) Set(ctx context.Context, key PriceKey, p domain.HistoricalPrice, ttl time.Duration) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return c.r.Set(ctx, key.RedisKey(), string(b), ttl)
}

func (c *HistoricalPriceCache) Delete(ctx context.Context, key PriceKey) error {
	return c.r.Del(ctx, key.RedisKey())
}
