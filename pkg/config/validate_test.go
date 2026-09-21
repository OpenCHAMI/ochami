// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"testing"
)

func TestCoerceBool(t *testing.T) {
	tests := []struct {
		name   string
		in     any
		want   bool
		wantOK bool
	}{
		{"bool true", true, true, true},
		{"bool false", false, false, true},
		{"string true", "true", true, true},
		{"string True", "True", true, true},
		{"string false", "False", false, true},
		{"string 1", "1", true, true},
		{"string 0", "0", false, true},
		{"invalid string", "yesplease", false, false},
		{"int not supported", 1, false, false},
		{"nil", nil, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := coerceBool(tt.in)
			if ok != tt.wantOK || (ok && got != tt.want) {
				t.Fatalf("coerceBool(%v) = (%v, %v), want (%v, %v)", tt.in, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
