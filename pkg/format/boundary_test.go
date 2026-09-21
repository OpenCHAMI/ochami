// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package format

import "testing"

func TestUnmarshalDataSliceReportsElementErrors(t *testing.T) {
	tests := []struct {
		name   string
		data   string
		format DataFormat
	}{
		{name: "JSON array", data: `[{`, format: DataFormatJson},
		{name: "YAML mapping", data: "key: value\n", format: DataFormatYaml},
		{name: "YAML sequence", data: "- key: value\n", format: DataFormatYaml},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var values []int
			if err := UnmarshalDataSlice([]byte(tt.data), &values, tt.format); err == nil {
				t.Fatal("UnmarshalDataSlice() returned nil for incompatible data")
			}
		})
	}
}
