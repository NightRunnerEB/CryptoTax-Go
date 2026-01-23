package inmemory

import "github.com/NightRunner/CryptoTax-Go/services/price-svc/pkg/in-memory"

type Symbol = string
type CoinID = string

type CoinIdCache = inmemory.Store[Symbol, CoinID]
