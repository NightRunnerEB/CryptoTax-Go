package usecase

import "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/internal/domain"

var supportedFiatCurrencies = []domain.SupportedFiatCurrency{
	{Code: "USD", DisplayName: "US Dollar"},
	{Code: "RUB", DisplayName: "Russian Ruble"},
	{Code: "KZT", DisplayName: "Kazakhstani Tenge"},
}

var supportedFiatSet = map[string]struct{}{
	"USD": {},
	"RUB": {},
	"KZT": {},
}

func listSupportedFiatCurrencies() []domain.SupportedFiatCurrency {
	out := make([]domain.SupportedFiatCurrency, len(supportedFiatCurrencies))
	copy(out, supportedFiatCurrencies)
	return out
}

func isSupportedFiatCurrency(code string) bool {
	_, ok := supportedFiatSet[code]
	return ok
}
