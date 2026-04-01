package resolver

import (
	"testing"

	pkginmemory "github.com/NightRunner/CryptoTax-Go/pkg/in-memory"
	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
)

func TestResolve_CacheHit(t *testing.T) {
	cache := pkginmemory.NewStore[string, string]()
	cache.UpsertMany(map[string]string{"BTC": "bitcoin"})

	r := &CoinIdResolver{
		coinIdCache: cache,
	}

	got, err := r.Resolve("BTC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "bitcoin" {
		t.Fatalf("got %q, want bitcoin", got)
	}
}

func TestResolve_CacheMiss(t *testing.T) {
	cache := pkginmemory.NewStore[string, string]()
	r := &CoinIdResolver{
		coinIdCache: cache,
	}

	_, err := r.Resolve("BTC")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.ErrUnknownSymbol {
		t.Fatalf("expected UNKNOWN_SYMBOL, got %v", err)
	}
}
