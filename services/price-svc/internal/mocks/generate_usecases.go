package mocks

//go:generate mockgen -destination=usecase_mocks.gen.go -package=mocks github.com/NightRunner/CryptoTax-Go/services/price-svc/internal/domain HistoricalPriceUseCase,FXProvider,CoinIdResolver,UserSymbolUseCase
