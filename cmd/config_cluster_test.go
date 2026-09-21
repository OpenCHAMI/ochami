// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// config_cluster_test.go exercises the "config cluster" commands: set
// (including --default and non-URI per-service keys), unset, show (all
// clusters or one), and delete. Rejection-path cases, including the mutually
// exclusive --user/--system/--config source flags, are covered in
// config_cluster_errors_test.go.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/config"
)

// TestConfigClusterSet_Default verifies "config cluster set --default" marks the
// cluster as the default in the config file.
func TestConfigClusterSet_Default(t *testing.T) {
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
	f, err := config.OpenFile(cfg)
	if err != nil {
		t.Fatalf("read semantic config: %v", err)
	}
	if got, _ := f.Get("default-cluster").(string); got != "foobar" {
		t.Errorf("default-cluster = %q, want foobar", got)
	}
}

// TestConfigClusterSet_ServiceKey verifies setting a per-service URI key.
func TestConfigClusterSet_ServiceKey(t *testing.T) {
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
	f, err := config.OpenFile(cfg)
	if err != nil {
		t.Fatalf("read semantic config: %v", err)
	}
	clusters, err := f.Clusters()
	if err != nil {
		t.Fatalf("Clusters(): %v", err)
	}
	if got := clusters[0].Cluster.SMD.URI; got != "https://foobar.openchami.cluster/smd" {
		t.Errorf("cluster.smd.uri = %v, want configured service URI", got)
	}
}

// TestConfigClusterShow_All verifies "config cluster show" with no args shows all
// clusters.
func TestConfigClusterShow_All(t *testing.T) {
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

// TestConfigClusterSet_CreatesFile verifies "config cluster set" creates a
// missing config file when the user confirms.
func TestConfigClusterSet_CreatesFile(t *testing.T) {
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

// TestConfigClusterShow_KeyOfCluster verifies "config cluster show <name> <key>"
// returns the value for a nested key.
func TestConfigClusterShow_KeyOfCluster(t *testing.T) {
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

// TestConfigClusterSet_DeclineCreate verifies that declining to create a
// missing config file exits with CodeDeclined and leaves no file behind.
func TestConfigClusterSet_DeclineCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new", "config.yaml")

	res := runOchamiWithInput(t, "n\n", "--config", path, "config", "cluster", "set",
		"foobar", "cluster.uri", "https://foobar.openchami.cluster")
	if res.err == nil || res.exitCode != cli.CodeDeclined {
		t.Fatalf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodeDeclined, cli.CodeName(cli.CodeDeclined))
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("config path stat error = %v, want not-exist", err)
	}
}
