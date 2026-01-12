// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"math"
	"reflect"
	"testing"
)

// approxEqual returns if two float64 numbers are exactly equal at low
// magnitudes and equal within a margin of error at high magnitudes.
func approxEqual(a, b float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	// Relative tolerance for big magnitudes, absolute for tiny.
	const eps = 1e-12
	diff := math.Abs(a - b)
	scale := math.Max(1, math.Max(math.Abs(a), math.Abs(b)))
	return diff/scale < eps || diff < eps
}

func TestStringToType(t *testing.T) {
	tests := []struct {
		in       string
		want     any
		wantType reflect.Type
	}{
		// bools
		{"true", true, reflect.TypeOf(true)},
		{"false", false, reflect.TypeOf(false)},
		{"tRUe", true, reflect.TypeOf(true)}, // case-insensitive
		{"1", true, reflect.TypeOf(true)},    // ParseBool handles "1"
		{"0", false, reflect.TypeOf(false)},  // ParseBool handles "0" too

		// integers (int64)
		{"01", int64(1), reflect.TypeOf(int64(0))}, // not a bool, parses as int
		{"-7", int64(-7), reflect.TypeOf(int64(0))},
		{"9223372036854775807", int64(math.MaxInt64), reflect.TypeOf(int64(0))},

		// floats (float64)
		{"3.14", float64(3.14), reflect.TypeOf(float64(0))},
		{"-2.5e3", float64(-2500), reflect.TypeOf(float64(0))},

		// too big for int64 -> falls back to float64
		{"9223372036854775808", float64(9.223372036854776e18), reflect.TypeOf(float64(0))},

		// special floats
		{"NaN", math.NaN(), reflect.TypeOf(float64(0))},

		// strings (fallback)
		{"hello", "hello", reflect.TypeOf("")},
		{"", "", reflect.TypeOf("")},
		{" True ", " True ", reflect.TypeOf("")}, // no trimming in parser
		{"yes", "yes", reflect.TypeOf("")},
	}

	for _, tt := range tests {
		got := StringToType(tt.in)

		// Check type first
		if reflect.TypeOf(got) != tt.wantType {
			t.Fatalf("StringToType(%q) type = %T, want %v", tt.in, got, tt.wantType)
		}

		// Check value based on type
		switch want := tt.want.(type) {
		case bool:
			if got != want {
				t.Fatalf("StringToType(%q) = %v, want %v", tt.in, got, want)
			}
		case int64:
			if got.(int64) != want {
				t.Fatalf("StringToType(%q) = %v, want %v", tt.in, got, want)
			}
		case float64:
			gf := got.(float64)
			if math.IsNaN(want) {
				if !math.IsNaN(gf) {
					t.Fatalf("StringToType(%q) = %v, want NaN", tt.in, got)
				}
			} else if !approxEqual(gf, want) {
				t.Fatalf("StringToType(%q) = %v, want %v", tt.in, gf, want)
			}
		case string:
			if got.(string) != want {
				t.Fatalf("StringToType(%q) = %q, want %q", tt.in, got, want)
			}
		default:
			t.Fatalf("unsupported want type for %q: %T", tt.in, tt.want)
		}
	}
}
