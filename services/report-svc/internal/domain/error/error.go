package apperr

import "fmt"

type ErrorCode string

const (
	ErrNotFound           ErrorCode = "NOT_FOUND"
	ErrInvalidArgument    ErrorCode = "INVALID_ARGUMENT"
	ErrConflict           ErrorCode = "CONFLICT"
	ErrInternal           ErrorCode = "INTERNAL_ERROR"
	ErrStorageUnavailable ErrorCode = "STORAGE_UNAVAILABLE"
	ErrStorageBadResponse ErrorCode = "STORAGE_BAD_RESPONSE"
	ErrRenderingFailed    ErrorCode = "RENDERING_FAILED"
)

type Detail interface {
	isDetail()
}

type FieldViolation struct {
	Field       string
	Description string
}

type Validation struct {
	Violations []FieldViolation
}

func (Validation) isDetail() {}

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
