package util

import (
	"math"
	"testing"
)

func TestToFixed(t *testing.T) {
	cases := []struct {
		name      string
		in        float64
		precision int
		want      float64
	}{
		{"round up at 2 decimals", 1.236, 2, 1.24},
		{"round down at 2 decimals", 1.234, 2, 1.23},
		{"truncate trailing 9s at 2 decimals", 0.999, 2, 1.00},
		{"zero precision rounds to integer", 1.5, 0, 2.0},
		{"larger precision than fractional part is no-op", 1.5, 5, 1.5},
		{"zero input stays zero", 0.0, 3, 0.0},
		// IEEE-754 edge case: 1.005 is stored as 1.00499... so
		// math.Round(100.4999...) is 100 → 1.00. Documented limitation;
		// callers needing exact decimal arithmetic must use a decimal
		// library (see comment on ToFixed).
		{"IEEE-754 edge case: 1.005 rounds DOWN due to binary representation", 1.005, 2, 1.00},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ToFixed(tc.in, tc.precision)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("ToFixed(%v, %d) = %v, want %v", tc.in, tc.precision, got, tc.want)
			}
		})
	}
}
