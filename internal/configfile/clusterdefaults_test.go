// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package configfile

// clusterdefaults_test.go covers the error arms of applyClusterDefaults (reached
// via ReadConfigWithDefaults) for malformed cluster entries: a missing name, a
// nil cluster block, and a non-map cluster block.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/openchami/ochami/pkg/config"
)

func writeTemp(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

// TestReadConfigMissingClusterName verifies a cluster entry without a name is
// rejected.
func TestReadConfigMissingClusterName(t *testing.T) {
	path := writeTemp(t, `clusters:
- cluster:
    uri: https://example.com
`)
	if _, err := ReadConfigWithDefaults(path); err == nil {
		t.Error("ReadConfigWithDefaults with unnamed cluster = nil, want error")
	}
}

// TestReadConfigNilClusterBlock verifies a cluster entry whose "cluster" block
// is nil is accepted (defaults are applied).
func TestReadConfigNilClusterBlock(t *testing.T) {
	path := writeTemp(t, `clusters:
- name: demo
`)
	if _, err := ReadConfigWithDefaults(path); err != nil {
		t.Errorf("ReadConfigWithDefaults with nil cluster block = %v, want nil", err)
	}
}

// TestReadConfigNonMapClusterBlock verifies a cluster entry whose "cluster"
// block is not a map is rejected.
func TestReadConfigNonMapClusterBlock(t *testing.T) {
	path := writeTemp(t, `clusters:
- name: demo
  cluster: "not-a-map"
`)
	if _, err := ReadConfigWithDefaults(path); err == nil {
		t.Error("ReadConfigWithDefaults with non-map cluster block = nil, want error")
	}
}

// TestReadConfigValidClusters verifies a well-formed multi-cluster config loads
// and applies defaults.
func TestReadConfigValidClusters(t *testing.T) {
	path := writeTemp(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: https://demo.example.com
- name: prod
  cluster:
    uri: https://prod.example.com
`)
	ko, err := ReadConfigWithDefaults(path)
	if err != nil {
		t.Fatalf("ReadConfigWithDefaults = %v, want nil", err)
	}
	if ko == nil {
		t.Fatal("ReadConfigWithDefaults returned nil koanf")
	}
}

// TestGetConfigClusterString covers fetching a whole cluster, a specific key,
// and a missing key.
func TestGetConfigClusterStringVariants(t *testing.T) {
	cl := config.ConfigCluster{
		Name:    "demo",
		Cluster: config.ConfigClusterConfig{URI: "https://demo.example.com"},
	}

	// Whole cluster (empty key).
	whole, err := GetConfigClusterString(cl, "")
	if err != nil {
		t.Fatalf("GetConfigClusterString(whole) = %v, want nil", err)
	}
	if whole == "" {
		t.Error("GetConfigClusterString(whole) returned empty")
	}

	// Specific key.
	uri, err := GetConfigClusterString(cl, "cluster.uri")
	if err != nil {
		t.Fatalf("GetConfigClusterString(cluster.uri) = %v, want nil", err)
	}
	if uri == "" {
		t.Error("GetConfigClusterString(cluster.uri) returned empty")
	}

	// Missing key -> empty string, no error.
	missing, err := GetConfigClusterString(cl, "cluster.does.not.exist")
	if err != nil {
		t.Errorf("GetConfigClusterString(missing) = %v, want nil", err)
	}
	if missing != "" {
		t.Errorf("GetConfigClusterString(missing) = %q, want empty", missing)
	}
}

// TestEffectiveKoanfLayered covers EffectiveKoanf merging multiple config files.
func TestEffectiveKoanfLayered(t *testing.T) {
	sys := writeTemp(t, "log:\n  level: error\n")
	usr := writeTemp(t, "log:\n  level: debug\n")

	ko, err := EffectiveKoanf(sys, usr)
	if err != nil {
		t.Fatalf("EffectiveKoanf = %v, want nil", err)
	}
	// The later (user) file should win.
	if got := ko.String("log.level"); got != "debug" {
		t.Errorf("log.level = %q, want debug (user overrides system)", got)
	}
}
