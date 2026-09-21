// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/openchami/ochami/pkg/config"
)

// TestLoad_RequiredMissingSource verifies that a non-optional missing source is
// an error.
func TestLoad_RequiredMissingSource(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	_, err := config.Load([]config.Source{{Name: "explicit", Path: missing}})
	if err == nil {
		t.Fatal("Load(): expected error for missing required source, got nil")
	}
}

// TestLoad_SourceErrors verifies that malformed or invalid sources produce
// descriptive errors.
func TestLoad_SourceErrors(t *testing.T) {
	tests := []struct {
		name       string
		sourceName string
		data       string
		want       string
	}{
		{
			name:       "malformed YAML",
			sourceName: "system",
			data:       "clusters: [\n",
			want:       "unable to load config 'system'",
		},
		{
			name:       "null global",
			sourceName: "user",
			data:       "timeout:\n",
			want:       "invalid config 'user'",
		},
		{
			name:       "strict type mismatch",
			sourceName: "user",
			data:       "timeout: 30\n",
			want:       "unable to merge config 'user'",
		},
		{
			name:       "invalid cluster boolean",
			sourceName: "system",
			data:       "clusters:\n  - name: foo\n    cluster:\n      enable-auth: maybe\n",
			want:       "unable to merge cluster 'foo' from config 'system'",
		},
		{
			name:       "negative timeout",
			sourceName: "user",
			data:       "timeout: -1s\n",
			want:       "invalid merged config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := writeSource(t, tt.sourceName, tt.data)
			_, err := config.Load([]config.Source{src})
			if err == nil {
				t.Fatal("Load(): expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want substring %q", err, tt.want)
			}
		})
	}
}
