package resolver

import (
	"fmt"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	inmemory "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/infra/in-memory"
)

type CoinIdResolver struct {
	tenantSymbolRepo domain.TenantSymbolRepo
	coinIdCache      *inmemory.CoinIdCache
}

func NewCoinIdResolver(tenantSymbolRepo domain.TenantSymbolRepo, coinIdCache *inmemory.CoinIdCache) domain.CoinIdResolver {
	return &CoinIdResolver{
		tenantSymbolRepo: tenantSymbolRepo,
		coinIdCache:      coinIdCache,
	}
}

func (r *CoinIdResolver) Resolve(symbol string) (string, error) {
	if coinID, ok := r.coinIdCache.Get(symbol); ok {
		return coinID, nil
	}

	// tenantUUID, err := domain.ParseTenantID(tenantID)
	// if err != nil {
	// 	return "", false
	// }

	// ts, err := r.tenantSymbolRepo.GetByTenantSourceSymbol(nil, tenantUUID, source, symbol)
	// if err != nil || ts.CoinID == "" {
	// 	return "", false
	// }

	// r.coinIdCache.Set(cacheKey, ts.CoinID)
	return "", fmt.Errorf("symbol not found: %s", symbol)
}
