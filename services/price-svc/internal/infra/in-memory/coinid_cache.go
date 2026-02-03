package inmemory

import (
	"os"
	"strconv"
	"strings"

	apperr "github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain/error"
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
		return nil, apperr.Internal("read coin id file failed", err, map[string]string{
			"path": path,
		})
	}

	var f coinIdFile
	if err := yaml.Unmarshal(b, &f); err != nil {
		return nil, apperr.Internal("parse coin id file failed", err, map[string]string{
			"path": path,
		})
	}

	m := make(map[Symbol]CoinID, len(f.Coins))
	for i, c := range f.Coins {
		sym := strings.TrimSpace(c.Symbol)
		id := strings.TrimSpace(c.CoinID)

		if sym == "" || id == "" {
			return nil, apperr.InvalidArgument("invalid coin id entry", nil, apperr.FieldViolation{
				Field:       "coins[" + strconv.Itoa(i) + "]",
				Description: "symbol and coin_id are required",
			})
		}
		if _, exists := m[sym]; exists {
			return nil, apperr.InvalidArgument("duplicate coin symbol", nil, apperr.FieldViolation{
				Field:       "symbol",
				Description: "duplicate value: " + sym,
			})
		}
		m[sym] = id
	}

	store := inmemory.NewStore[Symbol, CoinID]()
	store.ReplaceAll(m)

	return store, nil
}
