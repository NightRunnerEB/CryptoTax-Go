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

func PriceUnavailable(msg, coinID string, meta map[string]string, cause error) *Error {
	m := map[string]string{"coin_id": coinID}
	for k, v := range meta {
		m[k] = v
	}

	return &Error{
		Op:    Op(),
		Code:  ErrPriceUnavailable,
		Msg:   msg,
		Meta:  m,
		Cause: cause,
		Details: []Detail{
			Resource{Type: "coin", Name: coinID},
		},
	}
}

func ProviderUnavailable(msg, provider string, cause error, meta map[string]string) *Error {
	m := map[string]string{"provider": provider}
	for k, v := range meta {
		m[k] = v
	}

	return &Error{
		Op:    Op(),
		Code:  ErrProviderUnavailable,
		Msg:   msg,
		Meta:  m,
		Cause: cause,
	}
}

func ProviderBadResponse(msg, provider string, cause error, meta map[string]string) *Error {
	m := map[string]string{"provider": provider}
	for k, v := range meta {
		m[k] = v
	}

	return &Error{
		Op:    Op(),
		Code:  ErrProviderBadResponse,
		Msg:   msg,
		Meta:  m,
		Cause: cause,
	}
}

func FXUnavailable(msg, fiat string, meta map[string]string, cause error) *Error {
	m := map[string]string{"fiat": fiat}
	for k, v := range meta {
		m[k] = v
	}

	return &Error{
		Op:    Op(),
		Code:  ErrFXUnavailable,
		Msg:   msg,
		Meta:  m,
		Cause: cause,
		Details: []Detail{
			Resource{Type: "fiat", Name: fiat},
		},
	}
}

func UnsupportedFiat(msg, fiat string) *Error {
	return &Error{
		Op:   Op(),
		Code: ErrUnsupportedFiat,
		Msg:  msg,
		Meta: map[string]string{"fiat": fiat},
		Details: []Detail{
			Resource{Type: "fiat", Name: fiat},
		},
	}
}

func UnknownSymbol(symbol, source string) *Error {
	m := map[string]string{"symbol": symbol}
	if source != "" {
		m["source"] = source
	}

	return &Error{
		Op:   Op(),
		Code: ErrUnknownSymbol,
		Msg:  "symbol cannot be resolved",
		Meta: m,
		Details: []Detail{
			Resource{Type: "symbol", Name: symbol},
		},
	}
}

func AmbiguousSymbol(msg, symbol, source string, meta map[string]string) *Error {
	m := map[string]string{"symbol": symbol}
	if source != "" {
		m["source"] = source
	}
	for k, v := range meta {
		m[k] = v
	}

	return &Error{
		Op:   Op(),
		Code: ErrAmbiguousSymbol,
		Msg:  msg,
		Meta: m,
		Details: []Detail{
			Resource{Type: "symbol", Name: symbol},
		},
	}
}

func UnsupportedSource(msg, source string) *Error {
	return &Error{
		Op:   Op(),
		Code: ErrUnsupportedSource,
		Msg:  msg,
		Meta: map[string]string{"source": source},
		Details: []Detail{
			Resource{Type: "source", Name: source},
		},
	}
}
