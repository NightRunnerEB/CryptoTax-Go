package resolver

import (
	"github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
	inmemory "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/infra/in-memory"
)

type CoinIdResolver struct {
	userSymbolRepo domain.UserSymbolRepo
	coinIdCache    *inmemory.CoinIdCache
}

func NewCoinIdResolver(userSymbolRepo domain.UserSymbolRepo, coinIdCache *inmemory.CoinIdCache) domain.CoinIdResolver {
	return &CoinIdResolver{
		userSymbolRepo: userSymbolRepo,
		coinIdCache:    coinIdCache,
	}
}

func (r *CoinIdResolver) Resolve(symbol string) (string, error) {
	if coinID, ok := r.coinIdCache.Get(symbol); ok {
		return coinID, nil
	}

	// userUUID, err := domain.ParseUserID(userID)
	// if err != nil {
	// 	return "", false
	// }

	// ts, err := r.userSymbolRepo.GetByUserSourceSymbol(nil, userUUID, source, symbol)
	// if err != nil || ts.CoinID == "" {
	// 	return "", false
	// }

	// r.coinIdCache.Set(cacheKey, ts.CoinID)
	return "", apperr.UnknownSymbol(symbol, "")
}
