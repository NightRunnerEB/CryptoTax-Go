package apperr

import (
	"path"
	"runtime"
	"strings"
)

func Op() string {
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		return "unknown:0"
	}

	fn := runtime.FuncForPC(pc)
	fname := "unknown"
	if fn != nil {
		parts := strings.Split(fn.Name(), "/")
		fname = parts[len(parts)-1]
	}

	base := path.Base(file)
	return fname + "@" + base + ":" + itoa(line)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	sign := ""
	if v < 0 {
		sign = "-"
		v = -v
	}
	var out [20]byte
	i := len(out)
	for v > 0 {
		i--
		out[i] = byte('0' + v%10)
		v /= 10
	}
	return sign + string(out[i:])
}
