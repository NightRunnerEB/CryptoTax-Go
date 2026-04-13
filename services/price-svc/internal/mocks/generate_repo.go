package mocks

//go:generate mockgen -destination=repo_mocks.gen.go -package=mocks github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain HistoricalPriceRepo,UserSymbolRepo
