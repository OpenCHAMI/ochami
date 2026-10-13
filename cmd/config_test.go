// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
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
	"github.com/openchami/ochami/pkg/config"
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
	t.Parallel()

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
	t.Parallel()

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

// TestConfigShow_DefaultedKey verifies that "config show <key>" returns the
// koanf-applied default for a key that is absent from the file. Here the file
// sets only log.level, so log.format should come back as its default rather
// than empty.
func TestConfigShow_DefaultedKey(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	cfg := writeTempConfig(t, "log:\n  level: debug\n")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "show")
	if res.err != nil {
		t.Fatalf("config show: unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	// The file sets only log.level, so log.format must come from the defaults.
	if !strings.Contains(res.stdout, "debug") {
		t.Errorf("config show stdout = %q, want it to contain the explicitly-set log level", res.stdout)
	}
	if !strings.Contains(res.stdout, "format:") {
		t.Errorf("config show stdout = %q, want it to contain the defaulted log format", res.stdout)
	}
}

// TestConfigSet_CreatesFileOnConfirm verifies "config set" offers to create a
// missing config file and, on "y", creates and writes it.
func TestConfigSet_CreatesFileOnConfirm(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "new", "config.yaml")

	res := runOchamiWithInputAndRuntime(t, "y\n", "--config", path, "config", "set", "log.format", "json")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected config file to be created at %s: %v", path, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "format: json") {
		t.Errorf("config = %q, want log.format set to json", data)
	}
}

// TestConfigSet_DeclineCreate verifies that declining to create a missing config
// file exits without writing the file.
func TestConfigSet_DeclineCreate(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "new", "config.yaml")

	res := runOchamiWithInputAndRuntime(t, "n\n", "--config", path, "config", "set", "log.format", "json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected no config file to be created at %s", path)
	}
	if res.exitCode != cli.CodeDeclined {
		t.Errorf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodeDeclined, cli.CodeName(cli.CodeDeclined))
	}
}

// TestConfigUnset_ViaConfigFlag verifies "config unset <key>" removes a key from
// an explicit --config file.
func TestConfigUnset_ViaConfigFlag(t *testing.T) {
	t.Parallel()

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
	f, err := config.OpenFile(cfg)
	if err != nil {
		t.Fatalf("read semantic config: %v", err)
	}
	if f.Get("log.level") != nil {
		t.Error("log.level still exists after unset")
	}
}

// TestDefaultCluster_URIResolution verifies a command resolves its base URI from
// the default cluster's cluster.uri in a config file (no --uri flag).
func TestDefaultCluster_URIResolution(t *testing.T) {
	t.Parallel()

	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.Write([]byte(`[]`))
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

// TestPerServiceURI_Override verifies a per-service URI override in the cluster
// config is honored.
func TestPerServiceURI_Override(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`[]`))
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

// TestEnableAuth_ReadsTokenFromEnv verifies that, for a cluster with
// enable-auth set, a command reads the token from <CLUSTER>_ACCESS_TOKEN and
// sends it as a bearer token.
func TestEnableAuth_ReadsTokenFromEnv(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`[]`))
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

// TestEnableAuth_DisabledSkipsToken verifies that with enable-auth false, no
// token is required or sent.
func TestEnableAuth_DisabledSkipsToken(t *testing.T) {
	t.Parallel()

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`[]`))
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
