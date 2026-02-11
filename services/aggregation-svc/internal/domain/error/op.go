package apperr

import (
	"runtime"
	"strings"
)

// Op returns the function name that called the error factory.
// Expected stack:
//
//	<caller> -> <factory> -> apperr.Op() -> runtime.Caller
func Op() string {
	return Name(3)
}

// Name returns the function name at the specified stack depth.
func Name(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}

	return trimFunc(fn.Name())
}

// trimFunc shortens full path to a compact form.
func trimFunc(full string) string {
	if i := strings.LastIndexByte(full, '/'); i >= 0 && i+1 < len(full) {
		return full[i+1:]
	}
	return full
}
