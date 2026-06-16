// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build integration

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenCHAMI/ochami/test/integration/harness"
)

func TestConfigSet(t *testing.T) {
	tests := []struct {
		name         string
		initialCfg   string
		args         []string
		wantExitCode int
		wantInFile   string
	}{
		{
			name:         "set simple key",
			initialCfg:   "",
			args:         []string{"config", "set", "log.level", "debug"},
			wantExitCode: 0,
			wantInFile:   "level: debug",
		},
		{
			name: "set nested key",
			initialCfg: `log:
  level: warning
  format: rfc3339
`,
			args:         []string{"config", "set", "log.format", "json"},
			wantExitCode: 0,
			wantInFile:   "format: json",
		},
		{
			name:         "set default-cluster",
			initialCfg:   "",
			args:         []string{"config", "set", "default-cluster", "test-cluster"},
			wantExitCode: 0,
			wantInFile:   "default-cluster: test-cluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp config file
			configPath := harness.TempConfigFile(t, tt.initialCfg)

			// Run config set command
			args := append([]string{"--config", configPath}, tt.args...)
			result := harness.RunCLI(t, args...)

			// Check exit code
			harness.AssertExitCode(t, result, tt.wantExitCode)

			// Verify config file content
			content := harness.ReadConfigFile(t, configPath)
			harness.AssertContains(t, content, tt.wantInFile)
		})
	}
}

func TestConfigShow(t *testing.T) {
	tests := []struct {
		name         string
		configCfg    string
		args         []string
		wantExitCode int
		wantInStdout string
	}{
		{
			name: "show log level",
			configCfg: `log:
  level: debug
  format: json
`,
			args:         []string{"config", "show", "log.level"},
			wantExitCode: 0,
			wantInStdout: "debug",
		},
		{
			name: "show entire config",
			configCfg: `log:
  level: info
  format: rfc3339
default-cluster: my-cluster
`,
			args:         []string{"config", "show"},
			wantExitCode: 0,
			wantInStdout: "default-cluster: my-cluster",
		},
		{
			name: "show nonexistent key",
			configCfg: `log:
  level: warning
  format: rfc3339
`,
			args:         []string{"config", "show", "nonexistent.key"},
			wantExitCode: 0,
			wantInStdout: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := harness.TempConfigFile(t, tt.configCfg)

			args := append([]string{"--config", configPath}, tt.args...)
			result := harness.RunCLI(t, args...)

			harness.AssertExitCode(t, result, tt.wantExitCode)
			if tt.wantInStdout != "" {
				harness.AssertContains(t, result.Stdout, tt.wantInStdout)
			}
		})
	}
}

func TestConfigUnset(t *testing.T) {
	tests := []struct {
		name          string
		initialCfg    string
		args          []string
		wantExitCode  int
		wantNotInFile string
	}{
		{
			name: "unset log level",
			initialCfg: `log:
  level: debug
  format: json
`,
			args:          []string{"config", "unset", "log.level"},
			wantExitCode:  0,
			wantNotInFile: "level:",
		},
		{
			name: "unset default-cluster",
			initialCfg: `default-cluster: test-cluster
log:
  level: info
  format: rfc3339
`,
			args:          []string{"config", "unset", "default-cluster"},
			wantExitCode:  0,
			wantNotInFile: "default-cluster:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := harness.TempConfigFile(t, tt.initialCfg)

			args := append([]string{"--config", configPath}, tt.args...)
			result := harness.RunCLI(t, args...)

			harness.AssertExitCode(t, result, tt.wantExitCode)

			content := harness.ReadConfigFile(t, configPath)
			harness.AssertNotContains(t, content, tt.wantNotInFile)
		})
	}
}

func TestConfigClusterSet(t *testing.T) {
	tests := []struct {
		name         string
		initialCfg   string
		args         []string
		wantExitCode int
		wantInFile   string
	}{
		{
			name:         "create new cluster",
			initialCfg:   "",
			args:         []string{"config", "cluster", "set", "test-cluster", "cluster.uri", "https://test.example.com"},
			wantExitCode: 0,
			wantInFile:   "name: test-cluster",
		},
		{
			name: "update existing cluster URI",
			initialCfg: `clusters:
  - name: test-cluster
    cluster:
      uri: https://old.example.com
`,
			args:         []string{"config", "cluster", "set", "test-cluster", "cluster.uri", "https://new.example.com"},
			wantExitCode: 0,
			wantInFile:   "https://new.example.com",
		},
		{
			name: "set service-specific URI",
			initialCfg: `clusters:
  - name: test-cluster
    cluster:
      uri: https://cluster.example.com
`,
			args:         []string{"config", "cluster", "set", "test-cluster", "cluster.smd.uri", "https://smd.example.com"},
			wantExitCode: 0,
			wantInFile:   "https://smd.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := harness.TempConfigFile(t, tt.initialCfg)

			args := append([]string{"--config", configPath}, tt.args...)
			result := harness.RunCLI(t, args...)

			harness.AssertExitCode(t, result, tt.wantExitCode)

			content := harness.ReadConfigFile(t, configPath)
			harness.AssertContains(t, content, tt.wantInFile)
		})
	}
}

func TestConfigClusterShow(t *testing.T) {
	tests := []struct {
		name         string
		configCfg    string
		args         []string
		wantExitCode int
		wantInStdout string
	}{
		{
			name: "show cluster",
			configCfg: `clusters:
  - name: test-cluster
    cluster:
      uri: https://test.example.com
`,
			args:         []string{"config", "cluster", "show", "test-cluster"},
			wantExitCode: 0,
			wantInStdout: "https://test.example.com",
		},
		{
			name: "show cluster URI",
			configCfg: `clusters:
  - name: test-cluster
    cluster:
      uri: https://test.example.com
      smd:
        uri: https://smd.example.com
`,
			args:         []string{"config", "cluster", "show", "test-cluster", "cluster.uri"},
			wantExitCode: 0,
			wantInStdout: "https://test.example.com",
		},
		{
			name: "show nonexistent cluster",
			configCfg: `clusters:
  - name: exists
    cluster:
      uri: https://exists.example.com
`,
			args:         []string{"config", "cluster", "show", "nonexistent"},
			wantExitCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := harness.TempConfigFile(t, tt.configCfg)

			args := append([]string{"--config", configPath}, tt.args...)
			result := harness.RunCLI(t, args...)

			harness.AssertExitCode(t, result, tt.wantExitCode)
			if tt.wantInStdout != "" {
				harness.AssertContains(t, result.Stdout, tt.wantInStdout)
			}
		})
	}
}

func TestConfigClusterDelete(t *testing.T) {
	tests := []struct {
		name          string
		initialCfg    string
		args          []string
		wantExitCode  int
		wantNotInFile string
	}{
		{
			name: "delete cluster",
			initialCfg: `clusters:
  - name: test-cluster
    cluster:
      uri: https://test.example.com
  - name: other-cluster
    cluster:
      uri: https://other.example.com
`,
			args:          []string{"config", "cluster", "delete", "test-cluster"},
			wantExitCode:  0,
			wantNotInFile: "test-cluster",
		},
		{
			name: "delete default cluster removes default-cluster key",
			initialCfg: `default-cluster: test-cluster
clusters:
  - name: test-cluster
    cluster:
      uri: https://test.example.com
`,
			args:          []string{"config", "cluster", "delete", "test-cluster"},
			wantExitCode:  0,
			wantNotInFile: "default-cluster:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := harness.TempConfigFile(t, tt.initialCfg)

			args := append([]string{"--config", configPath}, tt.args...)
			result := harness.RunCLI(t, args...)

			harness.AssertExitCode(t, result, tt.wantExitCode)

			content := harness.ReadConfigFile(t, configPath)
			harness.AssertNotContains(t, content, tt.wantNotInFile)
		})
	}
}

func TestConfigFileCreation(t *testing.T) {
	t.Run("config set creates file if it doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "new-config.yaml")

		// Ensure file doesn't exist
		if _, err := os.Stat(configPath); err == nil {
			t.Fatal("config file should not exist initially")
		}

		// Run config set, answering yes to the creation prompt.
		result := harness.RunCLIWithInput(t, "y\n", "--config", configPath, "config", "set", "log.level", "info")

		// Command should succeed
		harness.AssertExitCode(t, result, 0)

		// File should now exist
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Fatal("config file should have been created")
		}

		// Verify content
		content := harness.ReadConfigFile(t, configPath)
		harness.AssertContains(t, content, "level: info")
	})
}

func TestConfigRejectsClusterKeys(t *testing.T) {
	t.Run("config set rejects cluster keys", func(t *testing.T) {
		configPath := harness.TempConfigFile(t, "")

		result := harness.RunCLI(t, "--config", configPath, "config", "set", "clusters.test.uri", "https://test.com")

		// Should fail
		if result.ExitCode == 0 {
			t.Errorf("config set should reject cluster keys, but succeeded")
		}

		// Error message should guide user to cluster commands
		output := result.Stderr + result.Stdout
		if !strings.Contains(strings.ToLower(output), "cluster") {
			t.Errorf("error message should mention cluster commands:\n%s", output)
		}
	})
}
