// Package engines contains jurisdiction-specific tax calculation engines.
//
// The intent is:
// - one engine per jurisdiction (RU, KZ, ...)
// - each engine can keep its own classification rules and tax policy behavior
// - worker/usecase picks engine by tax job policy snapshot jurisdiction
package engines
