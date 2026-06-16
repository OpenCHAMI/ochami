// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build integration

package uri

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/OpenCHAMI/ochami/test/integration/harness"
)

// TestURIRoutingPrecedence tests URI routing behavior with cluster defaults,
// cluster flags, cluster URI flags, and service-specific URI flags.
func TestURIRoutingPrecedence(t *testing.T) {
	// Create a fake server that echoes the request path
	server := harness.NewFakeHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"path": "%s"}`, r.URL.Path)
	})
	defer server.Close()

	tests := []struct {
		name           string
		config         string
		args           []string
		wantPathPrefix string
		wantErr        bool
	}{
		{
			name: "default cluster from config",
			config: fmt.Sprintf(`default-cluster: my-cluster
clusters:
  - name: my-cluster
    cluster:
      uri: %s
`, server.URL),
			args:           []string{"smd", "service", "status"},
			wantPathPrefix: "/hsm/v2",
		},
		{
			name: "--cluster flag overrides when no default",
			config: fmt.Sprintf(`clusters:
  - name: cluster-a
    cluster:
      uri: %s
  - name: cluster-b
    cluster:
      uri: %s/other
`, server.URL, server.URL),
			args:           []string{"--cluster", "cluster-b", "smd", "service", "status"},
			wantPathPrefix: "/other/hsm/v2",
		},
		{
			name: "--cluster-uri flag overrides cluster",
			config: fmt.Sprintf(`default-cluster: my-cluster
clusters:
  - name: my-cluster
    cluster:
      uri: %s/config-uri
`, server.URL),
			args:           []string{"--cluster-uri", server.URL + "/flag-uri", "smd", "service", "status"},
			wantPathPrefix: "/flag-uri/hsm/v2",
		},
		{
			name: "service --uri flag overrides all",
			config: fmt.Sprintf(`default-cluster: my-cluster
clusters:
  - name: my-cluster
    cluster:
      uri: %s/cluster
      smd:
        uri: %s/smd-config
`, server.URL, server.URL),
			args:           []string{"smd", "--uri", server.URL + "/smd-flag", "service", "status"},
			wantPathPrefix: "/smd-flag",
		},
		{
			name: "boot-service default path",
			config: fmt.Sprintf(`default-cluster: my-cluster
clusters:
  - name: my-cluster
    cluster:
      uri: %s
`, server.URL),
			args:           []string{"boot", "service", "status"},
			wantPathPrefix: "/boot-service",
		},
		{
			name: "bss default path",
			config: fmt.Sprintf(`default-cluster: my-cluster
clusters:
  - name: my-cluster
    cluster:
      uri: %s
`, server.URL),
			args:           []string{"bss", "service", "status"},
			wantPathPrefix: "/boot/v1",
		},
		{
			name: "metadata-service default path",
			config: fmt.Sprintf(`default-cluster: my-cluster
clusters:
  - name: my-cluster
    cluster:
      uri: %s
`, server.URL),
			args:           []string{"metadata", "service", "status"},
			wantPathPrefix: "/metadata-service",
		},
		{
			name: "cloud-init default path",
			config: fmt.Sprintf(`default-cluster: my-cluster
clusters:
  - name: my-cluster
    cluster:
      uri: %s
`, server.URL),
			args:           []string{"cloud-init", "service", "status"},
			wantPathPrefix: "/cloud-init",
		},
		{
			name: "absolute service URI bypasses cluster",
			config: fmt.Sprintf(`default-cluster: my-cluster
clusters:
  - name: my-cluster
    cluster:
      uri: %s/cluster
      smd:
        uri: %s/standalone-smd
`, server.URL, server.URL),
			args:           []string{"smd", "service", "status"},
			wantPathPrefix: "/standalone-smd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp config
			configPath := harness.TempConfigFile(t, tt.config)

			// Run CLI command
			args := append([]string{"--config", configPath}, tt.args...)
			result := harness.RunCLI(t, args...)

			if tt.wantErr {
				if result.ExitCode == 0 {
					t.Errorf("expected error, but command succeeded")
				}
				return
			}

			harness.AssertExitCode(t, result, 0)

			// Check that request was made to correct path
			if len(server.Requests) == 0 {
				t.Fatal("no requests recorded")
			}

			lastReq := server.Requests[len(server.Requests)-1]
			if len(lastReq.URL.Path) < len(tt.wantPathPrefix) || lastReq.URL.Path[:len(tt.wantPathPrefix)] != tt.wantPathPrefix {
				t.Errorf("request path = %q, want prefix %q", lastReq.URL.Path, tt.wantPathPrefix)
			}
		})
	}
}

func TestURIServiceSpecificOverrides(t *testing.T) {
	// Create a fake server
	server := harness.NewFakeHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"path": "%s"}`, r.URL.Path)
	})
	defer server.Close()

	tests := []struct {
		name        string
		config      string
		args        []string
		wantInPath  string
		description string
	}{
		{
			name: "SMD service override",
			config: fmt.Sprintf(`default-cluster: test
clusters:
  - name: test
    cluster:
      uri: %s
      smd:
        uri: %s/custom-smd
`, server.URL, server.URL),
			args:        []string{"smd", "service", "status"},
			wantInPath:  "/custom-smd",
			description: "SMD-specific URI should be used instead of cluster URI + default path",
		},
		{
			name: "boot-service override",
			config: fmt.Sprintf(`default-cluster: test
clusters:
  - name: test
    cluster:
      uri: %s
      boot-service:
        uri: %s/custom-boot
`, server.URL, server.URL),
			args:        []string{"boot", "service", "status"},
			wantInPath:  "/custom-boot",
			description: "boot-service-specific URI should be used",
		},
		{
			name: "metadata-service override",
			config: fmt.Sprintf(`default-cluster: test
clusters:
  - name: test
    cluster:
      uri: %s
      metadata-service:
        uri: %s/custom-metadata
`, server.URL, server.URL),
			args:        []string{"metadata", "service", "status"},
			wantInPath:  "/custom-metadata",
			description: "metadata-service-specific URI should be used",
		},
		{
			name: "relative service path with cluster URI",
			config: fmt.Sprintf(`default-cluster: test
clusters:
  - name: test
    cluster:
      uri: %s
      bss:
        uri: /custom-bss-path
`, server.URL),
			args:        []string{"bss", "service", "status"},
			wantInPath:  "/custom-bss-path",
			description: "relative service path should be joined with cluster URI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := harness.TempConfigFile(t, tt.config)

			args := append([]string{"--config", configPath}, tt.args...)
			result := harness.RunCLI(t, args...)

			if result.ExitCode != 0 {
				t.Logf("stdout: %s", result.Stdout)
				t.Logf("stderr: %s", result.Stderr)
				t.Fatalf("command failed: %v", result.Err)
			}

			if len(server.Requests) == 0 {
				t.Fatal("no requests recorded")
			}

			lastReq := server.Requests[len(server.Requests)-1]
			harness.AssertContains(t, lastReq.URL.Path, tt.wantInPath)
		})
	}
}

func TestURIMissingConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		args    []string
		wantErr string
	}{
		{
			name:    "no config no flags",
			config:  "",
			args:    []string{"smd", "service", "status"},
			wantErr: "could not get",
		},
		{
			name: "default cluster not found",
			config: `default-cluster: nonexistent
clusters:
  - name: exists
    cluster:
      uri: https://example.com
`,
			args:    []string{"smd", "service", "status"},
			wantErr: "not found",
		},
		{
			name: "cluster flag with nonexistent cluster",
			config: `clusters:
  - name: exists
    cluster:
      uri: https://example.com
`,
			args:    []string{"--cluster", "nonexistent", "smd", "service", "status"},
			wantErr: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := harness.TempConfigFile(t, tt.config)

			args := append([]string{"--config", configPath}, tt.args...)
			result := harness.RunCLI(t, args...)

			if result.ExitCode == 0 {
				t.Errorf("expected error, but command succeeded")
			}

			output := result.Stderr + result.Stdout
			harness.AssertContains(t, output, tt.wantErr)
		})
	}
}
