package apperr

import "fmt"

type ErrorCode string

// Generic
const (
	ErrNotFound        ErrorCode = "NOT_FOUND"
	ErrInvalidArgument ErrorCode = "INVALID_ARGUMENT"
	ErrConflict        ErrorCode = "CONFLICT"
	ErrInternal        ErrorCode = "INTERNAL_ERROR"
)

// Price (crypto -> USD)
const (
	ErrPriceUnavailable    ErrorCode = "PRICE_UNAVAILABLE"
	ErrProviderUnavailable ErrorCode = "PROVIDER_UNAVAILABLE"
	ErrProviderBadResponse ErrorCode = "PROVIDER_BAD_RESPONSE"
)

// FX (USD -> Fiat)
const (
	ErrFXUnavailable   ErrorCode = "FX_UNAVAILABLE"
	ErrUnsupportedFiat ErrorCode = "UNSUPPORTED_FIAT"
)

// Asset resolution
const (
	ErrUnknownSymbol     ErrorCode = "UNKNOWN_SYMBOL"
	ErrAmbiguousSymbol   ErrorCode = "AMBIGUOUS_SYMBOL"
	ErrUnsupportedSource ErrorCode = "UNSUPPORTED_SOURCE"
)

type Detail interface {
	isDetail()
}

type FieldViolation struct {
	Field       string
	Description string
}

// Validation detail
type Validation struct {
	Violations []FieldViolation
}

func (Validation) isDetail() {}

// Resource detail
type Resource struct {
	Type string
	Name string
}

func (Resource) isDetail() {}

type Error struct {
	Op      string
	Code    ErrorCode
	Msg     string
	Meta    map[string]string
	Details []Detail
	Cause   error
}

func (e *Error) Error() string {
	if e.Op == "" {
		return e.Msg
	}
	return fmt.Sprintf("%s: %s", e.Op, e.Msg)
}

func (e *Error) Unwrap() error { return e.Cause }
