package mocks

//go:generate mockgen -destination=usecase_deps_mocks.gen.go -package=mocks github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain TaxProfileRepo,TaxJobRepo,ObjectStorage,AggregatedTxProvider,ReportClient
