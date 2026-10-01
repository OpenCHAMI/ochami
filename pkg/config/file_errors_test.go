// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config_test

// file_errors_test.go covers File's cluster-editing boundary errors: an
// invalid (delimiter-containing) cluster name and a malformed "clusters" block
// that fails to unmarshal, for both a set and an unset operation, plus
// unsetting a key that isn't present.

import (
	"strings"
	"testing"

	"github.com/openchami/ochami/pkg/config"
)

// TestFile_ClusterEditBoundaryErrors verifies that SetClusterKey and
// UnsetClusterKey reject a cluster name containing the key delimiter and a
// malformed clusters block, and that UnsetClusterKey rejects a key that isn't
// set, each with an error naming the problem.
func TestFile_ClusterEditBoundaryErrors(t *testing.T) {
	tests := []struct {
		name string
		data string
		call func(f *config.File) error
		want string
	}{
		{
			name: "SetClusterKey rejects delimiter in name",
			data: "clusters:\n- name: demo\n  cluster:\n    uri: https://example.com\n",
			call: func(f *config.File) error { return f.SetClusterKey("bad.name", "cluster.uri", "https://new") },
			want: "delimiter",
		},
		{
			name: "SetClusterKey on malformed clusters block",
			data: "clusters: scalar\n",
			call: func(f *config.File) error { return f.SetClusterKey("demo", "cluster.uri", "https://new") },
			want: "unmarshal clusters",
		},
		{
			name: "UnsetClusterKey rejects delimiter in name",
			data: "clusters:\n- name: demo\n  cluster:\n    uri: https://example.com\n",
			call: func(f *config.File) error { return f.UnsetClusterKey("bad.name", "cluster.uri") },
			want: "delimiter",
		},
		{
			name: "UnsetClusterKey on malformed clusters block",
			data: "clusters: scalar\n",
			call: func(f *config.File) error { return f.UnsetClusterKey("demo", "cluster.uri") },
			want: "unmarshal clusters",
		},
		{
			name: "UnsetClusterKey on missing key",
			data: "clusters:\n- name: demo\n  cluster:\n    uri: https://example.com\n",
			call: func(f *config.File) error { return f.UnsetClusterKey("demo", "cluster.smd.uri") },
			want: "does not exist",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := config.OpenFile(writeFileFixture(t, tt.data))
			if err != nil {
				t.Fatalf("OpenFile(): %v", err)
			}
			err = tt.call(f)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("call error = %v, want text %q", err, tt.want)
			}
		})
	}
}
