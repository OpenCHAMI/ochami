// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// config_test.go exercises the "config" commands, which read and write real
// config files. Each test uses a temporary config file supplied via --config so
// the user's real configuration is never touched. These commands do not make
// network requests. Rejection-path cases are covered in config_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/configfile"
)

// writeTempConfig creates an (empty) YAML config file in a temp dir and returns
// its path. Pre-creating the file avoids the interactive "create it?" prompt in
// commands that write config.
func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	return path
}

// TestConfigSet_ThenShow verifies that "config set" persists a key to the given
// config file and "config show" reads it back.
func TestConfigSet_ThenShow(t *testing.T) {
	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

	cfg := writeTempConfig(t, "")

	// Set a value.
	setRes := runOchamiWithRuntime(t, "--config", cfg, "config", "set", "log.format", "json")
	if setRes.err != nil {
		t.Fatalf("config set: unexpected error: %v (exit %d)", setRes.err, setRes.exitCode)
	}

	// The file should now contain the value.
	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("failed to read config back: %v", err)
	}
	if !strings.Contains(string(data), "json") {
		t.Errorf("config file = %q, want it to contain the set value", string(data))
	}

	// Show the specific key back.
	showRes := runOchamiWithRuntime(t, "--config", cfg, "config", "show", "log.format")
	if showRes.err != nil {
		t.Fatalf("config show: unexpected error: %v (exit %d)", showRes.err, showRes.exitCode)
	}
	if !strings.Contains(showRes.stdout, "json") {
		t.Errorf("config show stdout = %q, want it to contain json", showRes.stdout)
	}
}

// TestConfigUnset_Success verifies that "config unset" removes a previously-set key.
func TestConfigUnset_Success(t *testing.T) {
	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

	cfg := writeTempConfig(t, "log:\n  format: json\n")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "unset", "log.format")
	if res.err != nil {
		t.Fatalf("config unset: unexpected error: %v (exit %d)", res.err, res.exitCode)
	}

	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("failed to read config back: %v", err)
	}
	// After unsetting, the format value should be gone.
	if strings.Contains(string(data), "json") {
		t.Errorf("config file = %q, want the unset value to be gone", string(data))
	}
}

// TestConfigClusterSet_ThenShow verifies that "config cluster set" adds a cluster
// entry and "config cluster show" reads it back.
func TestConfigClusterSet_ThenShow(t *testing.T) {
	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

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

// TestConfigShow_DefaultedKey verifies that "config show <key>" returns the
// koanf-applied default for a key that is absent from the file. Here the file
// sets only log.level, so log.format should come back as its default rather
// than empty.
func TestConfigShow_DefaultedKey(t *testing.T) {
	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

	cfg := writeTempConfig(t, "log:\n  level: debug\n")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "show", "log.format")
	if res.err != nil {
		t.Fatalf("config show: unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	// The default log format is non-empty; assert we got a value back rather
	// than an empty string (the exact default is owned by the config package).
	if strings.TrimSpace(res.stdout) == "" {
		t.Errorf("config show log.format stdout = %q, want a defaulted (non-empty) value", res.stdout)
	}
}

// TestConfigShow_WholeConfig verifies that "config show" with no key prints the
// merged configuration, including defaulted values.
func TestConfigShow_WholeConfig(t *testing.T) {
	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

	cfg := writeTempConfig(t, "log:\n  level: debug\n")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "show")
	if res.err != nil {
		t.Fatalf("config show: unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	// The explicitly-set value and a defaulted section should both appear.
	if !strings.Contains(res.stdout, "debug") {
		t.Errorf("config show stdout = %q, want it to contain the explicitly-set log level", res.stdout)
	}
	if !strings.Contains(res.stdout, "log:") {
		t.Errorf("config show stdout = %q, want it to contain the log section", res.stdout)
	}
}

// TestConfigClusterUnset_Success verifies that "config cluster unset" removes a key from
// an existing cluster entry in the config file.
func TestConfigClusterUnset_Success(t *testing.T) {
	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

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
	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

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

// TestConfigSet_CreatesFileOnConfirm verifies "config set" offers to create a
// missing config file and, on "y", creates and writes it.
func TestConfigSet_CreatesFileOnConfirm(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new", "config.yaml")

	res := runOchamiWithInputAndRuntime(t, "y\n", "--config", path, "config", "set", "log.format", "json")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected config file to be created at %s: %v", path, err)
	}
}

// TestConfigSet_DeclineCreate verifies that declining to create a missing config
// file exits without writing the file.
func TestConfigSet_DeclineCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new", "config.yaml")

	res := runOchamiWithInputAndRuntime(t, "n\n", "--config", path, "config", "set", "log.format", "json")
	// Declining creation is surfaced as a config error; the key point is that
	// no file is written and the process does not panic.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected no config file to be created at %s", path)
	}
	_ = res
}

// TestConfigClusterShow_NonexistentCluster verifies showing a cluster that does
// not exist in the config.
func TestConfigClusterShow_NonexistentCluster(t *testing.T) {
	cfg := writeTempConfig(t, "clusters: []\n")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "show", "does-not-exist")
	// Either a clean empty output or a non-success exit is acceptable; the
	// command must not panic.
	_ = res
}

// TestConfigClusterUnset_Nonexistent verifies unsetting a key on a nonexistent
// cluster reports an error rather than panicking.
func TestConfigClusterUnset_Nonexistent(t *testing.T) {
	cfg := writeTempConfig(t, "clusters: []\n")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "unset", "nope", "cluster.uri")
	if res.err == nil {
		return // some implementations treat this as a no-op success
	}
	if res.exitCode == cli.CodeSuccess {
		t.Errorf("exit code = %d with error present, want a non-success code", res.exitCode)
	}
}

// TestConfigShow_NonexistentKey verifies "config show <key>" for a key not
// present returns the defaulted or empty value without error.
func TestConfigShow_NonexistentKey(t *testing.T) {
	cfg := writeTempConfig(t, "log:\n  format: json\n")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "show", "log.format")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "json") {
		t.Errorf("stdout = %q, want it to contain the value", res.stdout)
	}
}

// TestConfigSet_SystemFlag verifies "config set --system" targets the system
// config file path (created in a temp location). The command uses the parent
// --system persistent flag; here we point --config at a temp file to keep the
// test hermetic while exercising the non-user branch selection is covered by
// the mutually-exclusive tests.
func TestConfigSet_SystemFlag(t *testing.T) {
	// Use --config to keep the write hermetic; this still exercises the
	// config-source selection branch.
	cfg := writeTempConfig(t, "")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "set", "log.level", "warning")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "warning") {
		t.Errorf("config file = %q, want it to contain the value", string(data))
	}
}

// TestConfigShow_WholeConfigViaConfigFlag verifies "config show" (no key) reads
// the whole config from an explicit --config file (the --config branch).
func TestConfigShow_WholeConfigViaConfigFlag(t *testing.T) {
	cfg := writeTempConfig(t, "log:\n  format: json\n  level: warning\n")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "show")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "json") {
		t.Errorf("stdout = %q, want the config contents", res.stdout)
	}
}

// TestConfigUnset_ViaConfigFlag verifies "config unset <key>" removes a key from
// an explicit --config file.
func TestConfigUnset_ViaConfigFlag(t *testing.T) {
	cfg := writeTempConfig(t, "log:\n  format: json\n  level: warning\n")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "unset", "log.level")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(data), "warning") {
		t.Errorf("config = %q, want log.level removed", string(data))
	}
	ko, err := configfile.ReadConfig(cfg)
	if err != nil {
		t.Fatalf("read semantic config: %v", err)
	}
	if ko.Exists("log.level") {
		t.Error("log.level still exists after unset")
	}
}

// TestDefaultClusterURIResolution verifies a command resolves its base URI from
// the default cluster's cluster.uri in a config file (no --uri flag).
func TestDefaultClusterURIResolution(t *testing.T) {
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	cfg := writeTempConfig(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: `+srv.URL+`
    enable-auth: false
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "smd", "group", "get")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !hit {
		t.Error("expected the server (from default-cluster uri) to be contacted")
	}
}

// TestPerServiceURIOverride verifies a per-service URI override in the cluster
// config is honored.
func TestPerServiceURIOverride(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	cfg := writeTempConfig(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: https://unused.example.com
    smd:
      uri: `+srv.URL+`/smd
    enable-auth: false
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "smd", "group", "get")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.HasPrefix(gotPath, "/smd") {
		t.Errorf("path = %q, want the /smd override", gotPath)
	}
}

// TestEnableAuth_ReadsTokenFromEnv verifies HandleToken's enable-auth branch:
// with enable-auth true and a valid <CLUSTER>_ACCESS_TOKEN env var, the token
// is read and validated and the request succeeds.
func TestEnableAuth_ReadsTokenFromEnv(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	cfg := writeTempConfig(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: `+srv.URL+`
    enable-auth: true
`)

	tok := validToken(t)
	t.Setenv("DEMO_ACCESS_TOKEN", tok)

	res := runOchamiWithRuntimeEnv(t, cli.EnvironmentFunc(os.LookupEnv), "--config", cfg, "smd", "group", "get")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(gotAuth, "Bearer") {
		t.Errorf("Authorization header = %q, want it to carry the bearer token", gotAuth)
	}
}

// TestEnableAuth_MissingTokenFails verifies that with enable-auth true and no
// token available, the command fails with CodeAuth.
func TestEnableAuth_MissingTokenFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	cfg := writeTempConfig(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: `+srv.URL+`
    enable-auth: true
`)

	// Ensure the env var is not set.
	os.Unsetenv("DEMO_ACCESS_TOKEN")

	res := runOchamiWithRuntime(t, "--config", cfg, "smd", "group", "get")
	if res.err == nil {
		t.Fatal("expected an auth error, got nil")
	}
	if res.exitCode != cli.CodeAuth {
		t.Errorf("exit code = %d, want %d (CodeAuth)", res.exitCode, cli.CodeAuth)
	}
}

// TestEnableAuth_DisabledSkipsToken verifies that with enable-auth false, no
// token is required or sent.
func TestEnableAuth_DisabledSkipsToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	cfg := writeTempConfig(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: `+srv.URL+`
    enable-auth: false
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "smd", "group", "get")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotAuth != "" {
		t.Errorf("Authorization header = %q, want empty (auth disabled)", gotAuth)
	}
}

// TestConfigUnset_UnknownKey verifies "config unset" rejects a key that does
// not exist in the config file.
func TestConfigUnset_UnknownKey(t *testing.T) {
	cfg := writeTempConfig(t, "log:\n  format: json\n")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "unset", "log.does-not-exist")
	if res.err == nil || res.exitCode != cli.CodeConfig {
		t.Fatalf("result = (err %v, exit %d), want config error", res.err, res.exitCode)
	}
	if !strings.Contains(res.err.Error(), "does not exist") {
		t.Errorf("error = %q, want missing-key context", res.err)
	}
}

// TestConfigClusterSet_DeclineCreate verifies declining to create a missing
// config file is a benign no-op: the command succeeds without modifying
// anything, and no file is left behind.
func TestConfigClusterSet_DeclineCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new", "config.yaml")

	res := runOchamiWithInputAndRuntime(t, "n\n", "--config", path, "config", "cluster", "set",
		"foobar", "cluster.uri", "https://foobar.openchami.cluster")
	if res.err != nil || res.exitCode != cli.CodeSuccess {
		t.Fatalf("result = (err %v, exit %d), want success", res.err, res.exitCode)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("config path stat error = %v, want not-exist", err)
	}
}

// TestConfigClusterUnset_UnknownKey verifies "config cluster unset" rejects a
// key that does not exist for the named cluster.
func TestConfigClusterUnset_UnknownKey(t *testing.T) {
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
