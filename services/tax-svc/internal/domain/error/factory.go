package apperr

func NotFound(msg string, res Resource, cause error) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrNotFound,
		Msg:   msg,
		Cause: cause,
		Details: []Detail{
			res,
		},
		Meta: map[string]string{
			"resource_type": res.Type,
			"resource_name": res.Name,
		},
	}
}

func InvalidArgument(msg string, cause error, violations ...FieldViolation) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrInvalidArgument,
		Msg:   msg,
		Cause: cause,
		Details: []Detail{
			Validation{Violations: violations},
		},
	}
}

func Conflict(msg string, cause error, meta map[string]string, details ...Detail) *Error {
	return &Error{
		Op:      Op(),
		Code:    ErrConflict,
		Msg:     msg,
		Cause:   cause,
		Meta:    meta,
		Details: details,
	}
}

func Internal(msg string, cause error, meta map[string]string, details ...Detail) *Error {
	return &Error{
		Op:      Op(),
		Code:    ErrInternal,
		Msg:     msg,
		Cause:   cause,
		Meta:    meta,
		Details: details,
	}
}

func NotImplemented(msg string, cause error, meta map[string]string) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrNotImplemented,
		Msg:   msg,
		Meta:  meta,
		Cause: cause,
	}
}

func AggregationUnavailable(msg string, cause error, meta map[string]string) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrAggregationUnavailable,
		Msg:   msg,
		Meta:  meta,
		Cause: cause,
	}
}

func AggregationBadResponse(msg string, cause error, meta map[string]string) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrAggregationBadResponse,
		Msg:   msg,
		Meta:  meta,
		Cause: cause,
	}
}

func AggregationFetchFailed(msg string, cause error, meta map[string]string) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrAggregationFetchFailed,
		Msg:   msg,
		Meta:  meta,
		Cause: cause,
	}
}

func StorageUnavailable(msg string, cause error, meta map[string]string) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrStorageUnavailable,
		Msg:   msg,
		Meta:  meta,
		Cause: cause,
	}
}

func StorageBadResponse(msg string, cause error, meta map[string]string) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrStorageBadResponse,
		Msg:   msg,
		Meta:  meta,
		Cause: cause,
	}
}

func JobClaimConflict(msg string, cause error, meta map[string]string) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrJobClaimConflict,
		Msg:   msg,
		Meta:  meta,
		Cause: cause,
	}
}

func NeedsPriceResolution(msg string, cause error, meta map[string]string) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrNeedsPriceResolution,
		Msg:   msg,
		Meta:  meta,
		Cause: cause,
	}
}

func NegativeInventory(msg string, cause error, meta map[string]string) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrNegativeInventory,
		Msg:   msg,
		Meta:  meta,
		Cause: cause,
	}
}

func InvalidTxShape(msg string, cause error, meta map[string]string) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrInvalidTxShape,
		Msg:   msg,
		Meta:  meta,
		Cause: cause,
	}
}

func UnsupportedKind(msg string, cause error, meta map[string]string) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrUnsupportedKind,
		Msg:   msg,
		Meta:  meta,
		Cause: cause,
	}
}

func MinIOUploadFailed(msg string, cause error, meta map[string]string) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrMinIOUploadFailed,
		Msg:   msg,
		Meta:  meta,
		Cause: cause,
	}
}
