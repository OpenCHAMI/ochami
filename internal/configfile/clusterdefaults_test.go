// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package configfile

// clusterdefaults_test.go covers the per-cluster defaults
// ReadConfigWithDefaults applies to well-formed cluster entries, including one
// with no cluster block. Malformed entries are covered in
// clusterdefaults_errors_test.go.

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

// TestReadConfigWithDefaults_NilClusterBlock verifies a cluster entry whose
// "cluster" block is nil is accepted.
func TestReadConfigWithDefaults_NilClusterBlock(t *testing.T) {
	path := writeTemp(t, `clusters:
- name: demo
`)
	if _, err := ReadConfigWithDefaults(path); err != nil {
		t.Errorf("ReadConfigWithDefaults with nil cluster block = %v, want nil", err)
	}
}

// TestReadConfigWithDefaults_ValidClusters verifies a well-formed multi-cluster
// config loads and applies defaults.
func TestReadConfigWithDefaults_ValidClusters(t *testing.T) {
	path := writeTemp(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: https://demo.example.com
- name: prod
  cluster:
    uri: https://prod.example.com
`)
	eff, err := ReadConfigWithDefaults(path)
	if err != nil {
		t.Fatalf("ReadConfigWithDefaults = %v, want nil", err)
	}
	if got := eff.Get("default-cluster"); got != "demo" {
		t.Errorf("default-cluster = %v, want demo", got)
	}
	clusters, ok := eff.Get("clusters").([]map[string]any)
	if !ok || len(clusters) != 2 {
		t.Fatalf("clusters = %v, want 2 entries", eff.Get("clusters"))
	}
	// Defaults (e.g. enable-auth) should have been applied per-cluster.
	demo, ok := clusters[0]["cluster"].(map[string]any)
	if !ok || demo["enable-auth"] != true {
		t.Errorf("demo cluster enable-auth = %v, want true (default applied)", demo["enable-auth"])
	}
}
