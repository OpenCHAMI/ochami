// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// config_cluster_test.go extends the "config cluster" coverage to branches the
// existing tests do not reach: the --default flag on set, setting non-URI
// per-service keys, showing all clusters vs a single cluster, and the mutually
// exclusive --user/--system/--config source flags.

import (
	"os"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/configfile"
)

// TestConfigClusterSetDefault verifies "config cluster set --default" marks the
// cluster as the default in the config file.
func TestConfigClusterSetDefault(t *testing.T) {
	cfg := writeTempConfig(t, "")

	res := runOchami(t, "--config", cfg, "config", "cluster", "set", "--default",
		"foobar", "cluster.uri", "https://foobar.openchami.cluster")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}

	showRes := runOchami(t, "--config", cfg, "config", "show")
	if showRes.err != nil {
		t.Fatalf("config show: unexpected error: %v (exit %d)", showRes.err, showRes.exitCode)
	}
	if !strings.Contains(showRes.stdout, "foobar") {
		t.Errorf("config show stdout = %q, want it to reference the default cluster", showRes.stdout)
	}
	ko, err := configfile.ReadConfig(cfg)
	if err != nil {
		t.Fatalf("read semantic config: %v", err)
	}
	if got := ko.String("default-cluster"); got != "foobar" {
		t.Errorf("default-cluster = %q, want foobar", got)
	}
}

// TestConfigClusterSetServiceKey verifies setting a per-service URI key.
func TestConfigClusterSetServiceKey(t *testing.T) {
	cfg := writeTempConfig(t, "")

	res := runOchami(t, "--config", cfg, "config", "cluster", "set",
		"foobar", "cluster.smd.uri", "https://foobar.openchami.cluster/smd")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}

	showRes := runOchami(t, "--config", cfg, "config", "cluster", "show", "foobar", "cluster.smd.uri")
	if showRes.err != nil {
		t.Fatalf("config cluster show: unexpected error: %v (exit %d)", showRes.err, showRes.exitCode)
	}
	if !strings.Contains(showRes.stdout, "foobar.openchami.cluster/smd") {
		t.Errorf("stdout = %q, want the service URI", showRes.stdout)
	}
	ko, err := configfile.ReadConfig(cfg)
	if err != nil {
		t.Fatalf("read semantic config: %v", err)
	}
	var clusters []map[string]any
	if err := ko.Unmarshal("clusters", &clusters); err != nil {
		t.Fatalf("unmarshal clusters: %v", err)
	}
	cluster := clusters[0]["cluster"].(map[string]any)
	smd := cluster["smd"].(map[string]any)
	if got := smd["uri"]; got != "https://foobar.openchami.cluster/smd" {
		t.Errorf("cluster.smd.uri = %v, want configured service URI", got)
	}
}

// TestConfigClusterShowAll verifies "config cluster show" with no args shows all
// clusters.
func TestConfigClusterShowAll(t *testing.T) {
	cfg := writeTempConfig(t, `clusters:
- name: foobar
  cluster:
    uri: https://foobar.openchami.cluster
- name: bazqux
  cluster:
    uri: https://bazqux.openchami.cluster
`)

	res := runOchami(t, "--config", cfg, "config", "cluster", "show")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "foobar") || !strings.Contains(res.stdout, "bazqux") {
		t.Errorf("stdout = %q, want it to list both clusters", res.stdout)
	}
}

// TestConfigClusterShowOne verifies "config cluster show <name>" shows the whole
// cluster entry.
func TestConfigClusterShowOne(t *testing.T) {
	cfg := writeTempConfig(t, `clusters:
- name: foobar
  cluster:
    uri: https://foobar.openchami.cluster
`)

	res := runOchami(t, "--config", cfg, "config", "cluster", "show", "foobar")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "foobar.openchami.cluster") {
		t.Errorf("stdout = %q, want the cluster's URI", res.stdout)
	}
}

// TestConfigClusterSetMutuallyExclusiveSources verifies that specifying both
// --user and --system is a usage error.
func TestConfigClusterSetMutuallyExclusiveSources(t *testing.T) {
	res := runOchami(t, "config", "cluster", "set", "--user", "--system",
		"foobar", "cluster.uri", "https://foobar.openchami.cluster")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
}

// TestConfigClusterSetCreatesFile verifies "config cluster set" creates a
// missing config file when the user confirms.
func TestConfigClusterSetCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/sub/config.yaml"

	res := runOchamiWithInput(t, "y\ny\n", "--config", path, "config", "cluster", "set",
		"foobar", "cluster.uri", "https://foobar.openchami.cluster")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected config file created at %s: %v", path, err)
	}
}

// TestConfigClusterDeleteFromExisting verifies deleting a cluster and that the
// file is updated. (Covers the delete RunE success path with a real file.)
func TestConfigClusterDeleteFromExistingTwo(t *testing.T) {
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
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(data), "foobar") {
		t.Errorf("config = %q, want foobar removed", string(data))
	}
	if !strings.Contains(string(data), "bazqux") {
		t.Errorf("config = %q, want bazqux retained", string(data))
	}
}

// TestConfigClusterShowNotFoundErrors verifies "config cluster show <name>" for
// a nonexistent cluster is a config error.
func TestConfigClusterShowNotFoundErrors(t *testing.T) {
	cfg := writeTempConfig(t, `clusters:
- name: foobar
  cluster:
    uri: https://foobar.openchami.cluster
`)

	res := runOchami(t, "--config", cfg, "config", "cluster", "show", "does-not-exist")
	if res.err == nil {
		t.Fatal("expected a config error for unknown cluster, got nil")
	}
}

// TestConfigClusterShowKeyOfCluster verifies "config cluster show <name> <key>"
// returns the value for a nested key.
func TestConfigClusterShowKeyOfCluster(t *testing.T) {
	cfg := writeTempConfig(t, `clusters:
- name: foobar
  cluster:
    uri: https://foobar.openchami.cluster
`)

	res := runOchami(t, "--config", cfg, "config", "cluster", "show", "foobar", "cluster.uri")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "foobar.openchami.cluster") {
		t.Errorf("stdout = %q, want the URI value", res.stdout)
	}
}

// TestConfigClusterUnsetKey verifies removing a key from an existing cluster.
func TestConfigClusterUnsetKey(t *testing.T) {
	cfg := writeTempConfig(t, `clusters:
- name: foobar
  cluster:
    uri: https://foobar.openchami.cluster
    smd:
      uri: /smd
`)

	res := runOchami(t, "--config", cfg, "config", "cluster", "unset", "foobar", "cluster.smd.uri")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(data), "/smd") {
		t.Errorf("config = %q, want smd uri removed", string(data))
	}
}
