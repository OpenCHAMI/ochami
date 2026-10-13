// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"os"
	"strings"
	"testing"
)

// TestConfigClusterSet_ThenShow verifies that "config cluster set" adds a cluster
// entry and "config cluster show" reads it back.
func TestConfigClusterSet_ThenShow(t *testing.T) {
	cfg := writeTempConfig(t, "")

	setRes := runOchami(t, "--config", cfg, "config", "cluster", "set",
		"foobar", "cluster.uri", "https://foobar.openchami.cluster")
	if setRes.err != nil {
		t.Fatalf("config cluster set: unexpected error: %v (exit %d)", setRes.err, setRes.exitCode)
	}

	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("failed to read config back: %v", err)
	}
	if !strings.Contains(string(data), "foobar") {
		t.Errorf("config file = %q, want it to contain the cluster name", string(data))
	}

	showRes := runOchami(t, "--config", cfg, "config", "cluster", "show", "foobar")
	if showRes.err != nil {
		t.Fatalf("config cluster show: unexpected error: %v (exit %d)", showRes.err, showRes.exitCode)
	}
	if !strings.Contains(showRes.stdout, "foobar.openchami.cluster") {
		t.Errorf("config cluster show stdout = %q, want it to contain the URI", showRes.stdout)
	}
}

// TestConfigClusterUnset_Success verifies that "config cluster unset" removes a key from
// an existing cluster entry in the config file.
func TestConfigClusterUnset_Success(t *testing.T) {
	// Seed a config file with a cluster that has a uri and a smd uri.
	cfg := writeTempConfig(t, `clusters:
  - name: foobar
    cluster:
      uri: https://foobar.openchami.cluster
      smd:
        uri: /hsm/v2
`)

	res := runOchami(t, "--config", cfg, "config", "cluster", "unset", "foobar", "cluster.smd.uri")
	if res.err != nil {
		t.Fatalf("config cluster unset: unexpected error: %v (exit %d)", res.err, res.exitCode)
	}

	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("failed to read config back: %v", err)
	}
	// The removed key's value should be gone; the cluster itself should remain.
	if strings.Contains(string(data), "/hsm/v2") {
		t.Errorf("config file = %q, want the unset key to be gone", string(data))
	}
	if !strings.Contains(string(data), "foobar") {
		t.Errorf("config file = %q, want the cluster to remain", string(data))
	}
}

// TestConfigClusterDelete_Success verifies that "config cluster delete" removes
// the named cluster's entry from the config file and keeps the other clusters.
func TestConfigClusterDelete_Success(t *testing.T) {
	cfg := writeTempConfig(t, `clusters:
  - name: foobar
    cluster:
      uri: https://foobar.openchami.cluster
  - name: bazqux
    cluster:
      uri: https://bazqux.openchami.cluster
`)

	res := runOchami(t, "--config", cfg, "config", "cluster", "delete", "foobar")
	if res.err != nil {
		t.Fatalf("config cluster delete: unexpected error: %v (exit %d)", res.err, res.exitCode)
	}

	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("failed to read config back: %v", err)
	}
	if strings.Contains(string(data), "foobar") {
		t.Errorf("config file = %q, want the deleted cluster to be gone", string(data))
	}
	if !strings.Contains(string(data), "bazqux") {
		t.Errorf("config file = %q, want the other cluster kept", string(data))
	}
}
