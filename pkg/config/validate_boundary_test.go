// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"strings"
	"testing"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
)

// TestValidateConfig_NullAndShapeBoundaries verifies that validateConfig
// rejects a null or empty global value and a null cluster boolean, and leaves a
// non-map cluster block to the loader's shape checks.
func TestValidateConfig_NullAndShapeBoundaries(t *testing.T) {
	tests := []struct {
		name string
		raw  map[string]any
		want string
	}{
		{name: "global null", raw: map[string]any{"timeout": nil}, want: "non-null"},
		{name: "empty global", raw: map[string]any{"log": map[string]any{"format": ""}}, want: "non-empty"},
		{
			name: "null cluster boolean",
			raw: map[string]any{
				"clusters": []map[string]any{{"name": "demo", "cluster": map[string]any{"enable-auth": nil}}},
			},
			want: "boolean",
		},
		{
			name: "non-map cluster is ignored by semantic validation",
			raw: map[string]any{
				"clusters": []map[string]any{{"name": "demo", "cluster": "bad"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ko := koanf.NewWithConf(koanfConf)
			if err := ko.Load(confmap.Provider(tt.raw, "."), nil); err != nil {
				t.Fatal(err)
			}
			err := validateConfig(ko)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("validateConfig() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validateConfig() error = %v, want text %q", err, tt.want)
			}
		})
	}
}
