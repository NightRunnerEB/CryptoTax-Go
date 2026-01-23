package apperr

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrConflict        = errors.New("conflict")

	ErrPriceUnavailable    = errors.New("price unavailable")
	ErrProviderUnavailable = errors.New("provider unavailable")
	ErrProviderBadResponse = errors.New("provider bad response")

	ErrFXUnavailable   = errors.New("fx unavailable")
	ErrUnsupportedFiat = errors.New("unsupported fiat")

	ErrUnknownSymbol     = errors.New("unknown symbol")
	ErrAmbiguousSymbol   = errors.New("ambiguous symbol")
	ErrUnsupportedSource = errors.New("unsupported source")
)
