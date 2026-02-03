package apperr

import (
	"runtime"
	"strings"
)

// Op возвращает имя функции, которая вызвала фабрику ошибки.
// Хрупко: предполагаем стабильный call stack.
// Ожидаемый стек:
//
//	<caller> -> <factory> -> apperr.Op() -> runtime.Caller
func Op() string {
	return Name(3)
}

// Name возвращает имя функции на указанной глубине стека.
// skip=0 -> Name
// skip=1 -> Op
// skip=2 -> caller of Op (обычно фабрика)
// skip=3 -> caller of factory (обычно место, где создали ошибку)
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

// trimFunc режет полный путь до компактного вида.
// Было: "github.com/NightRunner/.../internal/coingecko.(*CGClient).Do"
// Стало: "coingecko.(*CGClient).Do"
func trimFunc(full string) string {
	if i := strings.LastIndexByte(full, '/'); i >= 0 && i+1 < len(full) {
		return full[i+1:]
	}
	return full
}
