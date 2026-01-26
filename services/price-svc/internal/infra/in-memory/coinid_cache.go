package inmemory

import (
	"fmt"
	"os"
	"strings"

	"github.com/NightRunner/CryptoTax-Go/services/price-svc/pkg/in-memory"
	"gopkg.in/yaml.v3"
)

type Symbol = string
type CoinID = string

type CoinIdCache = inmemory.Store[Symbol, CoinID]

type coinIdFile struct {
	Coins []struct {
		Symbol string `yaml:"symbol"`
		CoinID string `yaml:"coin_id"`
	} `yaml:"coins"`
}

func NewCoinIdCache(path string) (*CoinIdCache, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var f coinIdFile
	if err := yaml.Unmarshal(b, &f); err != nil {
		return nil, err
	}

	m := make(map[Symbol]CoinID, len(f.Coins))
	for i, c := range f.Coins {
		sym := strings.TrimSpace(c.Symbol)
		id := strings.TrimSpace(c.CoinID)

		if sym == "" || id == "" {
			return nil, fmt.Errorf("coinid: invalid entry at idx=%d (symbol=%q, coin_id=%q)", i, c.Symbol, c.CoinID)
		}
		if _, exists := m[sym]; exists {
			return nil, fmt.Errorf("coinid: duplicate symbol %q", sym)
		}
		m[sym] = id
	}

	store := inmemory.NewStore[Symbol, CoinID]()
	store.ReplaceAll(m)

	return store, nil
}
