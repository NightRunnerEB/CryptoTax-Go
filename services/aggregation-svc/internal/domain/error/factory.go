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

func LedgerUnavailable(msg string, cause error, meta map[string]string) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrLedgerUnavailable,
		Msg:   msg,
		Meta:  meta,
		Cause: cause,
	}
}

func LedgerBadResponse(msg string, cause error, meta map[string]string) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrLedgerBadResponse,
		Msg:   msg,
		Meta:  meta,
		Cause: cause,
	}
}

func PriceUnavailable(msg string, cause error, meta map[string]string) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrPriceUnavailable,
		Msg:   msg,
		Meta:  meta,
		Cause: cause,
	}
}

func PriceBadResponse(msg string, cause error, meta map[string]string) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrPriceBadResponse,
		Msg:   msg,
		Meta:  meta,
		Cause: cause,
	}
}

func DataNotReady(msg string, cause error, meta map[string]string, details ...Detail) *Error {
	return &Error{
		Op:      Op(),
		Code:    ErrDataNotReady,
		Msg:     msg,
		Meta:    meta,
		Cause:   cause,
		Details: details,
	}
}

func ImportAlreadyDone(msg string, cause error, meta map[string]string) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrImportAlreadyDone,
		Msg:   msg,
		Meta:  meta,
		Cause: cause,
	}
}

func ImportLocked(msg string, cause error, meta map[string]string) *Error {
	return &Error{
		Op:    Op(),
		Code:  ErrImportLocked,
		Msg:   msg,
		Meta:  meta,
		Cause: cause,
	}
}
