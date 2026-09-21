// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

const (
	processHelperEnv = "OCHAMI_PROCESS_HELPER"
	processArgsEnv   = "OCHAMI_PROCESS_ARGS"
)

// TestProcessHelper invokes the real main and cmd.Execute boundary in a child
// test process. Failures terminate this process through os.Exit, just as they
// do in the installed executable.
func TestProcessHelper(t *testing.T) {
	if os.Getenv(processHelperEnv) != "1" {
		return
	}

	var args []string
	if err := json.Unmarshal([]byte(os.Getenv(processArgsEnv)), &args); err != nil {
		t.Fatalf("decode process arguments: %v", err)
	}
	os.Args = append([]string{"ochami"}, args...)
	main()
}

func TestProcessExitCodes(t *testing.T) {
	tempDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"status":[]}`)); err != nil {
			t.Errorf("write test response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	badConfig := filepath.Join(tempDir, "invalid.yaml")
	if err := os.WriteFile(badConfig, []byte("clusters: [\n"), 0o600); err != nil {
		t.Fatalf("write malformed config: %v", err)
	}
	validConfig := filepath.Join(tempDir, "config.yaml")
	configData := []byte(`default-cluster: helper
clusters:
  - name: helper
    cluster:
      uri: http://127.0.0.1:1
      enable-auth: true
`)
	if err := os.WriteFile(validConfig, configData, 0o600); err != nil {
		t.Fatalf("write process config: %v", err)
	}
	noAuthConfig := filepath.Join(tempDir, "no-auth.yaml")
	noAuthConfigData := []byte("default-cluster: helper\nclusters:\n  - name: helper\n    cluster:\n      uri: " + server.URL + "\n      enable-auth: false\n")
	if err := os.WriteFile(noAuthConfig, noAuthConfigData, 0o600); err != nil {
		t.Fatalf("write no-auth process config: %v", err)
	}

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantOutput string
	}{
		{
			name:       "success",
			args:       []string{"--ignore-config", "version"},
			wantCode:   cli.CodeSuccess,
			wantOutput: "Version:",
		},
		{
			name:     "usage failure",
			args:     []string{"--ignore-config", "--not-a-real-flag"},
			wantCode: cli.CodeUsage,
		},
		{
			name:     "configuration failure",
			args:     []string{"--config", badConfig, "version"},
			wantCode: cli.CodeConfig,
		},
		{
			name:       "authentication failure",
			args:       []string{"--config", validConfig, "smd", "service", "status"},
			wantCode:   cli.CodeAuth,
			wantOutput: "HELPER_ACCESS_TOKEN unset",
		},
		{
			name:       "network failure",
			args:       []string{"--ignore-config", "--cluster-uri", "http://127.0.0.1:0", "--no-token", "smd", "service", "status"},
			wantCode:   cli.CodeNetwork,
			wantOutput: "failed to get SMD status",
		},
		{
			name:       "generic failure",
			args:       []string{"--config", noAuthConfig, "pcs", "status", "show", "x0c0s0b0n0"},
			wantCode:   cli.CodeGeneric,
			wantOutput: "no status found for the specified component",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argsJSON, err := json.Marshal(tt.args)
			if err != nil {
				t.Fatalf("encode process arguments: %v", err)
			}

			cmd := exec.Command(os.Args[0], "-test.run=^TestProcessHelper$")
			cmd.Env = replaceProcessEnv(os.Environ(),
				processHelperEnv+"=1",
				processArgsEnv+"="+string(argsJSON),
				"HOME="+tempDir,
				"XDG_CONFIG_HOME="+tempDir,
			)
			output, runErr := cmd.CombinedOutput()
			gotCode := cli.CodeSuccess
			if runErr != nil {
				var exitErr *exec.ExitError
				if !errors.As(runErr, &exitErr) {
					t.Fatalf("run helper process: %v", runErr)
				}
				gotCode = exitErr.ExitCode()
			}

			if gotCode != tt.wantCode {
				t.Errorf("exit code = %d, want %d; output:\n%s", gotCode, tt.wantCode, output)
			}
			if tt.wantOutput != "" && !strings.Contains(string(output), tt.wantOutput) {
				t.Errorf("output does not contain %s; got:\n%s", strconv.Quote(tt.wantOutput), output)
			}
		})
	}
}

func replaceProcessEnv(environ []string, replacements ...string) []string {
	replaced := make(map[string]struct{}, len(replacements))
	for _, replacement := range replacements {
		key, _, _ := strings.Cut(replacement, "=")
		replaced[key] = struct{}{}
	}

	result := make([]string, 0, len(environ)+len(replacements))
	for _, entry := range environ {
		key, _, _ := strings.Cut(entry, "=")
		if _, ok := replaced[key]; !ok {
			result = append(result, entry)
		}
	}
	return append(result, replacements...)
}
