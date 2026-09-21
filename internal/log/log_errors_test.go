// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package log

import (
	"io"
	"testing"
)

// TestNew_Validation verifies that New rejects an invalid log level, format, or
// color setting.
func TestNew_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		level  string
		format string
		color  string
	}{
		{name: "level", level: "invalid", format: "basic", color: "off"},
		{name: "format", level: "info", format: "invalid", color: "off"},
		{name: "color", level: "info", format: "basic", color: "invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := New(io.Discard, tt.level, tt.format, tt.color); err == nil {
				t.Fatal("New() error = nil, want validation error")
			}
		})
	}
}
