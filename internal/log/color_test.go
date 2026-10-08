// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package log

import (
	"fmt"
	"testing"
)

// TestColorize verifies that colorize wraps a value in the ANSI escape codes
// for a color unless coloring is disabled explicitly or by NO_COLOR.
func TestColorize(t *testing.T) {
	type args struct {
		s        interface{}
		c        int
		disabled bool
	}
	tests := []struct {
		name    string
		args    args
		noColor string
		want    string
	}{
		{
			name: "color",
			args: args{
				s:        ">",
				c:        colorBlack,
				disabled: false,
			},
			want: fmt.Sprintf("\x1b[%dm%v\x1b[0m", colorBlack, ">"),
		},
		{
			name: "bold",
			args: args{
				s:        ">",
				c:        colorBold,
				disabled: false,
			},
			want: fmt.Sprintf("\x1b[%dm%v\x1b[0m", colorBold, ">"),
		},
		{
			name: "NO_COLOR",
			args: args{
				s:        ">",
				c:        colorBold,
				disabled: false,
			},
			noColor: "1",
			want:    ">",
		},
		{
			name: "disabled",
			args: args{
				s:        ">",
				c:        colorBold,
				disabled: true,
			},
			want: ">",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("NO_COLOR", tt.noColor)
			if got := colorize(tt.args.s, tt.args.c, tt.args.disabled); got != tt.want {
				t.Errorf("colorize() = %v, want %v", got, tt.want)
			}
		})
	}
}
