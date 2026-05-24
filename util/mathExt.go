package util

import "math"

// ToFixed rounds num to the given number of decimal places, returning
// the result as a float64. Mirrors the helper of the same name in the
// prod-vfeeg-backend (per ADR-0006 parity gap).
//
// Note: float64 round-trip is lossy for many decimal values; callers
// that need exact decimal arithmetic should use a shopspring/decimal-
// style library, not this helper.
func ToFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return math.Round(num*output) / output
}
