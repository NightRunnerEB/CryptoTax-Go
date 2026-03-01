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

// External dependencies
const (
	ErrLedgerUnavailable  ErrorCode = "LEDGER_UNAVAILABLE"
	ErrLedgerBadResponse  ErrorCode = "LEDGER_BAD_RESPONSE"
	ErrPriceUnavailable   ErrorCode = "PRICE_UNAVAILABLE"
	ErrPriceBadResponse   ErrorCode = "PRICE_BAD_RESPONSE"
	ErrDataNotReady       ErrorCode = "DATA_NOT_READY"
	ErrImportAlreadyDone  ErrorCode = "IMPORT_ALREADY_COMPLETED"
	ErrImportLocked       ErrorCode = "IMPORT_LOCKED"
	ErrImportInconsistent ErrorCode = "IMPORT_INCONSISTENT"
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
