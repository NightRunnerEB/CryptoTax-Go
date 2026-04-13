package mocks

//go:generate mockgen -destination=server_uc_mocks.gen.go -package=mocks github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain AggregationUseCase,UserSettingsUseCase
