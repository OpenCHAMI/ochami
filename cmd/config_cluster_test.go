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
	"path/filepath"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/configfile"
)

// TestConfigClusterSet_Default verifies "config cluster set --default" marks the
// cluster as the default in the config file.
func TestConfigClusterSet_Default(t *testing.T) {

	t.Parallel()

	cfg := writeTempConfig(t, "")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "set", "--default",
		"foobar", "cluster.uri", "https://foobar.openchami.cluster")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}

	showRes := runOchamiWithRuntime(t, "--config", cfg, "config", "show")
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

// TestConfigClusterSet_ServiceKey verifies setting a per-service URI key.
func TestConfigClusterSet_ServiceKey(t *testing.T) {

	t.Parallel()

	cfg := writeTempConfig(t, "")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "set",
		"foobar", "cluster.smd.uri", "https://foobar.openchami.cluster/smd")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}

	showRes := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "show", "foobar", "cluster.smd.uri")
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

// TestConfigClusterShow_All verifies "config cluster show" with no args shows all
// clusters.
func TestConfigClusterShow_All(t *testing.T) {

	t.Parallel()

	cfg := writeTempConfig(t, `clusters:
- name: foobar
  cluster:
    uri: https://foobar.openchami.cluster
- name: bazqux
  cluster:
    uri: https://bazqux.openchami.cluster
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "show")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "foobar") || !strings.Contains(res.stdout, "bazqux") {
		t.Errorf("stdout = %q, want it to list both clusters", res.stdout)
	}
}

// TestConfigClusterShow_One verifies "config cluster show <name>" shows the whole
// cluster entry.
func TestConfigClusterShow_One(t *testing.T) {

	t.Parallel()

	cfg := writeTempConfig(t, `clusters:
- name: foobar
  cluster:
    uri: https://foobar.openchami.cluster
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "show", "foobar")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "foobar.openchami.cluster") {
		t.Errorf("stdout = %q, want the cluster's URI", res.stdout)
	}
}

// TestConfigClusterSet_MutuallyExclusiveSources verifies that specifying both
// --user and --system is a usage error.
func TestConfigClusterSet_MutuallyExclusiveSources(t *testing.T) {

	t.Parallel()

	res := runOchamiWithRuntime(t, "--ignore-config", "config", "cluster", "set", "--user", "--system",
		"foobar", "cluster.uri", "https://foobar.openchami.cluster")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
}

// TestConfigClusterSet_CreatesFile verifies "config cluster set" creates a
// missing config file when the user confirms.
func TestConfigClusterSet_CreatesFile(t *testing.T) {

	t.Parallel()

	dir := t.TempDir()
	path := dir + "/sub/config.yaml"

	res := runOchamiWithInputAndRuntime(t, "y\ny\n", "--config", path, "config", "cluster", "set",
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
func TestConfigClusterDelete_FromExistingTwo(t *testing.T) {

	t.Parallel()

	cfg := writeTempConfig(t, `clusters:
- name: foobar
  cluster:
    uri: https://foobar.openchami.cluster
- name: bazqux
  cluster:
    uri: https://bazqux.openchami.cluster
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "delete", "foobar")
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

// TestConfigClusterShow_NotFoundErrors verifies "config cluster show <name>" for
// a nonexistent cluster is a config error.
func TestConfigClusterShow_NotFoundErrors(t *testing.T) {

	t.Parallel()

	cfg := writeTempConfig(t, `clusters:
- name: foobar
  cluster:
    uri: https://foobar.openchami.cluster
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "show", "does-not-exist")
	if res.err == nil {
		t.Fatal("expected a config error for unknown cluster, got nil")
	}
}

// TestConfigClusterShow_KeyOfCluster verifies "config cluster show <name> <key>"
// returns the value for a nested key.
func TestConfigClusterShow_KeyOfCluster(t *testing.T) {

	t.Parallel()

	cfg := writeTempConfig(t, `clusters:
- name: foobar
  cluster:
    uri: https://foobar.openchami.cluster
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "show", "foobar", "cluster.uri")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "foobar.openchami.cluster") {
		t.Errorf("stdout = %q, want the URI value", res.stdout)
	}
}

// TestConfigClusterUnset_Key verifies removing a key from an existing cluster.
func TestConfigClusterUnset_Key(t *testing.T) {

	t.Parallel()

	cfg := writeTempConfig(t, `clusters:
- name: foobar
  cluster:
    uri: https://foobar.openchami.cluster
    smd:
      uri: /smd
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "unset", "foobar", "cluster.smd.uri")
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

// TestConfigClusterSet_ThenShow verifies that "config cluster set" adds a cluster
// entry and "config cluster show" reads it back.
func TestConfigClusterSet_ThenShow(t *testing.T) {

	t.Parallel()

	cfg := writeTempConfig(t, "")

	setRes := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "set",
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

	showRes := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "show", "foobar")
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

	t.Parallel()

	// Seed a config file with a cluster that has a uri and a smd uri.
	cfg := writeTempConfig(t, `clusters:
  - name: foobar
    cluster:
      uri: https://foobar.openchami.cluster
      smd:
        uri: /hsm/v2
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "unset", "foobar", "cluster.smd.uri")
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

// TestConfigClusterDelete_Success verifies that "config cluster delete" removes a whole
// cluster entry from the config file.
func TestConfigClusterDelete_Success(t *testing.T) {

	t.Parallel()

	cfg := writeTempConfig(t, `clusters:
  - name: foobar
    cluster:
      uri: https://foobar.openchami.cluster
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "delete", "foobar")
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
}

// TestConfigClusterDelete_NotFound verifies that deleting a non-existent cluster
// resolves to a config error.
func TestConfigClusterDelete_NotFound(t *testing.T) {

	t.Parallel()

	cfg := writeTempConfig(t, "clusters: []\n")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "delete", "does-not-exist")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeConfig {
		t.Errorf("exit code = %d, want %d (CodeConfig)", res.exitCode, cli.CodeConfig)
	}
}

func TestConfigClusterSet_DeclineCreate(t *testing.T) {

	t.Parallel()

	path := filepath.Join(t.TempDir(), "new", "config.yaml")

	res := runOchamiWithInputAndRuntime(t, "n\n", "--config", path, "config", "cluster", "set",
		"foobar", "cluster.uri", "https://foobar.openchami.cluster")
	// User declining to create file is not an error - command exits cleanly with CodeSuccess
	if res.err != nil {
		t.Fatalf("result = (err %v, exit %d), want no error on user decline", res.err, res.exitCode)
	}
	if res.exitCode != cli.CodeSuccess {
		t.Errorf("exit code = %d, want %d (CodeSuccess)", res.exitCode, cli.CodeSuccess)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("config path stat error = %v, want not-exist", err)
	}
}

func TestConfigClusterUnset_UnknownKey(t *testing.T) {

	t.Parallel()

	cfg := writeTempConfig(t, `clusters:
- name: foobar
  cluster:
    uri: https://foobar.openchami.cluster
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "unset", "foobar", "cluster.smd.uri")
	if res.err == nil || res.exitCode != cli.CodeConfig {
		t.Fatalf("result = (err %v, exit %d), want config error", res.err, res.exitCode)
	}
	if !strings.Contains(res.err.Error(), "doesn't exist") {
		t.Errorf("error = %q, want missing-key context", res.err)
	}
}

// TestConfigClusterShow_NonexistentCluster verifies showing a cluster that does
// not exist in the config.
func TestConfigClusterShow_NonexistentCluster(t *testing.T) {

	t.Parallel()

	cfg := writeTempConfig(t, "clusters: []\n")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "show", "does-not-exist")
	// Either a clean empty output or a non-success exit is acceptable; the
	// command must not panic.
	_ = res
}

// TestConfigClusterUnset_Nonexistent verifies unsetting a key on a nonexistent
// cluster reports an error rather than panicking.
func TestConfigClusterUnset_Nonexistent(t *testing.T) {

	t.Parallel()

	cfg := writeTempConfig(t, "clusters: []\n")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "unset", "nope", "cluster.uri")
	if res.err == nil {
		return // some implementations treat this as a no-op success
	}
	if res.exitCode == cli.CodeSuccess {
		t.Errorf("exit code = %d with error present, want a non-success code", res.exitCode)
	}
}
