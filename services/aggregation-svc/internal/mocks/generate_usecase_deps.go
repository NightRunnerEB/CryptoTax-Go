package mocks

//go:generate mockgen -destination=usecase_deps_mocks.gen.go -package=mocks github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain AggregatedTransactionRepo,ImportStateRepo,UserSettingsRepo,LedgerClient,PriceClient,LockManager
