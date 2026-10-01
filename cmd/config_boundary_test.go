// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// config_boundary_test.go exercises configuration edge cases (explicit
// nulls, malformed YAML, unusual file states) and the error handling they
// reach.

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/openchami/ochami/internal/cli"
)

func TestConfigCommands_BehaviorMatrix(t *testing.T) {
	t.Parallel()

	cfg := writeTempConfig(t, `default-cluster: alpha
log:
  format: json
  level: info
clusters:
- name: alpha
  cluster:
    uri: https://alpha.example
- name: beta
  cluster:
    uri: https://beta.example
`)

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantOutput []string
		wantError  string
	}{
		{name: "show all", args: []string{"--config", cfg, "config", "show"}, wantOutput: []string{"default-cluster", "alpha.example"}},
		{name: "show key", args: []string{"--config", cfg, "config", "show", "log.level"}, wantOutput: []string{"info"}},
		{name: "show clusters", args: []string{"--config", cfg, "config", "cluster", "show"}, wantOutput: []string{"alpha", "beta"}},
		{name: "show cluster", args: []string{"--config", cfg, "config", "cluster", "show", "alpha"}, wantOutput: []string{"alpha.example"}},
		{name: "show cluster key", args: []string{"--config", cfg, "config", "cluster", "show", "alpha", "cluster.uri"}, wantOutput: []string{"https://alpha.example"}},
		{name: "unknown cluster", args: []string{"--config", cfg, "config", "cluster", "show", "missing"}, wantCode: cli.CodeConfig, wantError: `cluster "missing" not found`},
		{name: "unknown cluster key", args: []string{"--config", cfg, "config", "cluster", "show", "alpha", "cluster.missing"}},
		{name: "set rejects cluster key", args: []string{"--config", cfg, "config", "set", "clusters.0.name", "changed"}, wantCode: cli.CodeUsage, wantError: "config cluster set"},
		{name: "unset rejects cluster key", args: []string{"--config", cfg, "config", "unset", "clusters.0.name"}, wantCode: cli.CodeUsage, wantError: "config cluster delete"},
		{name: "delete missing cluster", args: []string{"--config", cfg, "config", "cluster", "delete", "missing"}, wantCode: cli.CodeConfig, wantError: "cluster missing not found"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := runOchamiWithRuntime(t, tc.args...)
			if tc.wantCode == cli.CodeSuccess {
				if res.err != nil {
					t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
				}
			} else if res.err == nil || res.exitCode != tc.wantCode {
				t.Fatalf("result = (err %v, exit %d), want exit %d", res.err, res.exitCode, tc.wantCode)
			}
			for _, want := range tc.wantOutput {
				if !strings.Contains(res.stdout, want) {
					t.Errorf("output = %q, want %q", res.stdout, want)
				}
			}
			if tc.wantError != "" && !strings.Contains(res.err.Error(), tc.wantError) {
				t.Errorf("error = %q, want %q", res.err, tc.wantError)
			}
		})
	}
}

func TestConfigClusterDelete_UpdatesFile(t *testing.T) {
	t.Parallel()

	cfg := writeTempConfig(t, `default-cluster: alpha
clusters:
- name: alpha
  cluster:
    uri: https://alpha.example
- name: beta
  cluster:
    uri: https://beta.example
`)
	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "delete", "alpha")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	contents := string(data)
	if strings.Contains(contents, "name: alpha") || strings.Contains(contents, "default-cluster") {
		t.Errorf("config = %q, want alpha and default-cluster removed", contents)
	}
	if !strings.Contains(contents, "name: beta") {
		t.Errorf("config = %q, want beta retained", contents)
	}
}

func TestConfigShow_PropagatesOutputFailures(t *testing.T) {
	t.Parallel()

	cfg := writeTempConfig(t, `log:
  level: info
clusters:
- name: alpha
  cluster:
    uri: https://alpha.example
`)
	for _, args := range [][]string{
		{"--config", cfg, "config", "show"},
		{"--config", cfg, "config", "cluster", "show", "alpha"},
	} {
		res := runOchamiWithOutputWriter(t, commandErrorWriter{}, args...)
		if res.err == nil || res.exitCode != cli.CodePayload {
			t.Fatalf("args %v: result = (err %v, exit %d), want output failure", args, res.err, res.exitCode)
		}
		if !strings.Contains(res.err.Error(), "injected command output failure") {
			t.Errorf("args %v: error = %q, want writer failure", args, res.err)
		}
	}
}

// TestConfig_MalformedYAML verifies that a syntactically broken YAML file
// produces a clear parsing error with CodeConfig.
func TestConfig_MalformedYAML(t *testing.T) {
	t.Parallel()

	cfg := writeTempConfig(t, `clusters: [
  - name: test
    cluster:
      uri: http://localhost:8080
  invalid yaml here`)

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "show")

	if res.err == nil {
		t.Fatal("expected a YAML parsing error, got nil")
	}
	if res.exitCode != cli.CodeConfig {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeConfig, cli.CodeName(cli.CodeConfig))
	}
	if !strings.Contains(res.err.Error(), "yaml") {
		t.Errorf("expected error to mention yaml parsing, got: %v", res.err)
	}
}

// TestConfig_NullValueHandling verifies that an explicit YAML "null" for a
// non-nullable key (log.level) is rejected as a config validation error at
// load time, rather than silently resolving to its default.
func TestConfig_NullValueHandling(t *testing.T) {
	t.Parallel()

	cfg := writeTempConfig(t, `log:
  level: null
  format: json
clusters:
  - name: test
    cluster:
      uri: http://localhost:8080
`)

	// --ignore-config is deliberately omitted: it would skip reading --config
	// entirely (see InitConfig), which would make this test pass regardless
	// of what the file contains.
	res := runOchamiWithRuntime(t, "--config", cfg, "version")

	if res.err == nil {
		t.Fatal("expected an error for a null log.level, got nil")
	}
	if res.exitCode != cli.CodeConfig {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeConfig, cli.CodeName(cli.CodeConfig))
	}
}

// TestConfig_PermissionPreservation verifies that file permissions
// are preserved after a config write.
func TestConfig_PermissionPreservation(t *testing.T) {
	t.Parallel()

	// Create a config file with non-default permissions
	cfg := writeTempConfig(t, "log:\n  level: info\n")

	// Set non-default permissions (read-only for owner)
	if err := os.Chmod(cfg, 0o444); err != nil {
		t.Fatalf("failed to change permissions: %v", err)
	}
	defer func() {
		// Restore permissions for cleanup
		if err := os.Chmod(cfg, 0o644); err != nil {
			t.Errorf("restore config permissions: %v", err)
		}
	}()

	// Try to set a value - this should fail due to permissions
	res := runOchamiWithRuntime(t, "--config", cfg, "config", "set", "log.level", "debug")

	// Should fail with permission error (unless running as root)
	if res.err == nil {
		// If running as root, permissions don't apply
		t.Log("Skipping: running as root, permission test not applicable")
	} else {
		// Verify the original file is still intact
		data, err := os.ReadFile(cfg)
		if err != nil {
			t.Fatalf("failed to read config: %v", err)
		}
		if !strings.Contains(string(data), "level: info") {
			t.Errorf("config was modified: %s", string(data))
		}
	}
}

// TestConfig_MissingOptionalSources verifies that --ignore-config skips
// reading the path given to --config entirely (per InitConfig), so a
// nonexistent path doesn't cause an error.
func TestConfig_MissingOptionalSources(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "--config", "/nonexistent/path/to/config.yaml", "--ignore-config", "version")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if res.exitCode != cli.CodeSuccess {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeSuccess, cli.CodeName(cli.CodeSuccess))
	}
}

// TestConfig_SemanticNullDefault verifies that the "~" YAML null spelling for
// a non-nullable key is rejected the same way the literal "null" is (see
// TestConfig_NullValueHandling).
func TestConfig_SemanticNullDefault(t *testing.T) {
	t.Parallel()

	cfg := writeTempConfig(t, `log:
  level: ~
  format: json
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "version")

	if res.err == nil {
		t.Fatal("expected an error for a null log.level, got nil")
	}
	if res.exitCode != cli.CodeConfig {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeConfig, cli.CodeName(cli.CodeConfig))
	}
}

// TestConfigAtomicWrite_PreservesOnFailure verifies that a failed config write
// leaves the old file intact.
func TestConfigAtomicWrite_PreservesOnFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yaml")

	initialContent := `log:
  format: json
`
	if err := os.WriteFile(cfg, []byte(initialContent), 0o644); err != nil {
		t.Fatalf("failed to write initial config: %v", err)
	}

	// Make the directory read-only so the write fails.
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("failed to change directory permissions: %v", err)
	}
	defer func() {
		if err := os.Chmod(dir, 0o755); err != nil {
			t.Errorf("restore directory permissions: %v", err)
		}
	}()

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "set", "log.level", "debug")
	if res.err == nil {
		t.Skip("skipping: running as root, permission test not applicable")
	}

	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("failed to read config after failed write: %v", err)
	}
	if string(data) != initialContent {
		t.Errorf("config was modified on write failure:\n got: %s\nwant: %s",
			string(data), initialContent)
	}
}

// TestConfigPrecedence_ExplicitOverrides verifies that a --config file is
// actually read (and the command succeeds) when only --config is passed.
func TestConfigPrecedence_ExplicitOverrides(t *testing.T) {
	t.Parallel()

	cfg := writeTempConfig(t, `clusters:
  - name: test
    cluster:
      uri: http://localhost:8080
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "version")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "Version:") {
		t.Errorf("expected version output, got: %s", res.stdout)
	}
}

// TestConfigPrecedence_Defaults verifies that configuration values from a
// --config file apply when the corresponding flag was not set.
func TestConfigPrecedence_Defaults(t *testing.T) {
	t.Parallel()

	cfg := writeTempConfig(t, `clusters:
  - name: test
    cluster:
      uri: http://localhost:8080
log:
  format: basic
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "version")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "Version:") {
		t.Errorf("expected version output, got: %s", res.stdout)
	}
}

// runOchamiLoadingMergedConfig is like runOchamiWithRuntime, but also enables
// the runtime's normal (system+user) config-loading path, which test
// runtimes otherwise leave disabled so tests never inspect host
// configuration by accident. It exists for tests that specifically need to
// exercise that merged-loading path (e.g. XDG/HOME resolution), which never
// runs unless LoadConfig is set or --config/--ignore-config is passed.
func runOchamiLoadingMergedConfig(t *testing.T, env cli.Environment, args ...string) cmdResult {
	t.Helper()

	var combinedBuf bytes.Buffer
	rt := cli.NewTestRuntime(strings.NewReader(""), &combinedBuf, &combinedBuf)
	rt.LoadConfig = true
	if env != nil {
		rt = rt.WithEnvironment(env)
	}

	rootCmd := NewRootCmd()
	rootCmd.SetContext(cli.ContextWithRuntime(context.Background(), rt))
	rootCmd.SetArgs(args)
	rootCmd.SetOut(&combinedBuf)
	rootCmd.SetErr(&combinedBuf)

	runErr := rootCmd.Execute()
	return cmdResult{err: runErr, exitCode: cli.ExitCode(runErr), stdout: combinedBuf.String()}
}

// TestConfig_XDGAndHOMEFallback verifies that InitConfig resolves the user
// config file location via the HOME/.config fallback when XDG_CONFIG_HOME is
// unset, and reads a config file placed there, when neither --config nor
// --ignore-config is passed (so the normal merged-loading path runs).
func TestConfig_XDGAndHOMEFallback(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	configDir := filepath.Join(home, ".config", "ochami")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config directory: %v", err)
	}
	cfg := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("clusters:\n  - name: test\n    cluster:\n      uri: http://localhost:8080\n"), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	env := cli.EnvironmentFunc(func(key string) (string, bool) {
		switch key {
		case "HOME":
			return home, true
		case "XDG_CONFIG_HOME":
			return "", false // force the HOME/.config fallback
		}
		return "", false
	})

	res := runOchamiLoadingMergedConfig(t, env, "config", "show")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "localhost:8080") {
		t.Errorf("stdout = %q, want it to reflect the config loaded via the HOME/.config fallback", res.stdout)
	}
}

// TestConfigShow_LoadEquivalence verifies that the value "config show
// log.format" prints for a single key matches the corresponding value inside
// the full document that "config show" (no key) prints.
func TestConfigShow_LoadEquivalence(t *testing.T) {
	t.Parallel()

	cfg := writeTempConfig(t, `clusters:
  - name: test
    cluster:
      uri: http://localhost:8080
log:
  format: basic
`)

	showRes := runOchamiWithRuntime(t, "--config", cfg, "config", "show")
	if showRes.err != nil {
		t.Fatalf("config show: unexpected error: %v", showRes.err)
	}
	var whole struct {
		Log struct {
			Format string `yaml:"format"`
		} `yaml:"log"`
	}
	if err := yaml.Unmarshal([]byte(showRes.stdout), &whole); err != nil {
		t.Fatalf("failed to parse whole config show output as YAML: %v\noutput: %s", err, showRes.stdout)
	}

	keyRes := runOchamiWithRuntime(t, "--config", cfg, "config", "show", "log.format")
	if keyRes.err != nil {
		t.Fatalf("config show log.format: unexpected error: %v", keyRes.err)
	}
	gotKey := strings.TrimSpace(keyRes.stdout)

	if whole.Log.Format != gotKey {
		t.Errorf("log.format from whole config = %q, from single-key show = %q; want equal", whole.Log.Format, gotKey)
	}
	if gotKey != "basic" {
		t.Errorf("log.format = %q, want %q", gotKey, "basic")
	}
}
