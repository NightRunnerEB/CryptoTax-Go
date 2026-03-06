package apperr

import "fmt"

type ErrorCode string

const (
	ErrNotFound               ErrorCode = "NOT_FOUND"
	ErrInvalidArgument        ErrorCode = "INVALID_ARGUMENT"
	ErrConflict               ErrorCode = "CONFLICT"
	ErrInternal               ErrorCode = "INTERNAL_ERROR"
	ErrNotImplemented         ErrorCode = "NOT_IMPLEMENTED"
	ErrAggregationUnavailable ErrorCode = "AGGREGATION_UNAVAILABLE"
	ErrAggregationBadResponse ErrorCode = "AGGREGATION_BAD_RESPONSE"
	ErrAggregationFetchFailed ErrorCode = "AGGREGATION_FETCH_FAILED"
	ErrStorageUnavailable     ErrorCode = "STORAGE_UNAVAILABLE"
	ErrStorageBadResponse     ErrorCode = "STORAGE_BAD_RESPONSE"
	ErrJobClaimConflict       ErrorCode = "JOB_CLAIM_CONFLICT"
	ErrNeedsPriceResolution   ErrorCode = "NEEDS_PRICE_RESOLUTION"
	ErrNegativeInventory      ErrorCode = "NEGATIVE_INVENTORY"
	ErrInvalidTxShape         ErrorCode = "INVALID_TX_SHAPE"
	ErrUnsupportedKind        ErrorCode = "UNSUPPORTED_KIND"
	ErrMinIOUploadFailed      ErrorCode = "MINIO_UPLOAD_FAILED"
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
