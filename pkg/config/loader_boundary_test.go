// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoad_ClusterShapeBoundaries verifies that Load rejects a cluster with a
// missing or empty name or a non-map cluster block, and a global key whose type
// conflicts with the defaults.
func TestLoad_ClusterShapeBoundaries(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{name: "missing name", data: "clusters:\n  - cluster: {}\n", want: "missing a name"},
		{name: "empty name", data: "clusters:\n  - name: ''\n    cluster: {}\n", want: "missing a name"},
		{name: "non-map cluster", data: "clusters:\n  - name: demo\n    cluster: bad\n", want: "not a map"},
		{name: "type conflict", data: "log: scalar\n", want: "unable to merge"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(tt.data), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Load([]Source{{Name: "test", Path: path}})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Load() error = %v, want text %q", err, tt.want)
			}
		})
	}
}

// TestLoad_AcceptsNullClusterConfig verifies that Load accepts a cluster whose
// "cluster" block is an explicit null and applies the cluster defaults to it.
func TestLoad_AcceptsNullClusterConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("clusters:\n  - name: demo\n    cluster:\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load([]Source{{Name: "test", Path: path}})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Clusters) != 1 || cfg.Clusters[0].Name != "demo" || !cfg.Clusters[0].Cluster.EnableAuth {
		t.Fatalf("clusters = %#v, want defaulted demo cluster", cfg.Clusters)
	}
}
