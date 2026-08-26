// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build integration

package legacy

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/openchami/ochami/test/integration/harness"
)

func TestBSSRequests(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
		body     string
	}{
		{name: "service status", args: []string{"bss", "service", "status"}, wantPath: "/boot/v1/service/status", body: `{"status":"ok"}`},
		{name: "boot params get", args: []string{"bss", "boot", "params", "get"}, wantPath: "/boot/v1/bootparameters", body: `[]`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := harness.NewFakeHTTPServer(t, harness.NewServiceHandler(map[string]harness.ServiceResponse{
				"GET " + tc.wantPath: jsonResponse(tc.body),
			}))
			defer server.Close()

			configPath := harness.TempConfigFile(t, harness.ClusterConfig(fmt.Sprintf(`bss:
  uri: %s/boot/v1
`, server.URL)))

			args := append([]string{"--config", configPath}, tc.args...)
			result := harness.RunCLI(t, args...)
			harness.AssertExitCode(t, result, 0)
			harness.AssertRequestCount(t, server, 1)
			harness.AssertLastRequest(t, server, http.MethodGet, tc.wantPath)
		})
	}
}

func TestCloudInitRequests(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
	}{
		{name: "service status", args: []string{"cloud-init", "service", "status"}, wantPath: "/cloud-init/version"},
		{name: "service version", args: []string{"cloud-init", "service", "version"}, wantPath: "/cloud-init/version"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := harness.NewFakeHTTPServer(t, harness.NewServiceHandler(map[string]harness.ServiceResponse{
				"GET " + tc.wantPath: jsonResponse(`{"version":"test"}`),
			}))
			defer server.Close()

			configPath := harness.TempConfigFile(t, harness.ClusterConfig(fmt.Sprintf(`cloud-init:
  uri: %s/cloud-init
`, server.URL)))

			args := append([]string{"--config", configPath}, tc.args...)
			result := harness.RunCLI(t, args...)
			harness.AssertExitCode(t, result, 0)
			harness.AssertRequestCount(t, server, 1)
			harness.AssertLastRequest(t, server, http.MethodGet, tc.wantPath)
		})
	}
}

func jsonResponse(body string) harness.ServiceResponse {
	return harness.ServiceResponse{
		StatusCode: http.StatusOK,
		Body:       body,
		Headers:    map[string]string{"Content-Type": "application/json"},
	}
}

// TestBSSBootParamsWriteRequests verifies that BSS write commands construct
// requests in the form the service expects (method, path, body, auth header).
func TestBSSBootParamsWriteRequests(t *testing.T) {
	const token = "test-token-abc123"

	t.Run("boot params add sends POST with expected body", func(t *testing.T) {
		server := harness.NewFakeHTTPServer(t, harness.NewServiceHandler(map[string]harness.ServiceResponse{
			"POST /boot/v1/bootparameters": {
				StatusCode: http.StatusCreated,
				Body:       ``,
				Headers:    map[string]string{"Content-Type": "application/json"},
			},
		}))
		defer server.Close()

		configPath := harness.TempConfigFile(t, harness.ClusterConfig(fmt.Sprintf(`bss:
  uri: %s/boot/v1
`, server.URL)))

		result := harness.RunCLI(t,
			"--config", configPath,
			"--token", token,
			"bss", "boot", "params", "add",
			"--mac", "00:de:ad:be:ef:00",
			"--kernel", "https://example.com/kernel",
			"--initrd", "https://example.com/initrd",
			"--params", "quiet nosplash",
		)
		harness.AssertExitCode(t, result, 0)
		harness.AssertRequestCount(t, server, 1)
		harness.AssertLastRequest(t, server, http.MethodPost, "/boot/v1/bootparameters")
		// Verify the request body carries the boot parameters the user set. The
		// BootParams struct uses omitempty for most fields, so only the set
		// fields (plus the always-present cloud-init object) appear.
		harness.AssertLastRequestBodyContains(t, server, `"macs":["00:de:ad:be:ef:00"]`)
		harness.AssertLastRequestBodyContains(t, server, `"kernel":"https://example.com/kernel"`)
		harness.AssertLastRequestBodyContains(t, server, `"initrd":"https://example.com/initrd"`)
		harness.AssertLastRequestBodyContains(t, server, `"params":"quiet nosplash"`)
		harness.AssertLastRequestAuthToken(t, server, token)
	})

	t.Run("boot params add fails on 400 response", func(t *testing.T) {
		server := harness.NewFakeHTTPServer(t, harness.NewServiceHandler(map[string]harness.ServiceResponse{
			"POST /boot/v1/bootparameters": {
				StatusCode: http.StatusBadRequest,
				Body:       `{"error":"bad request"}`,
				Headers:    map[string]string{"Content-Type": "application/json"},
			},
		}))
		defer server.Close()

		configPath := harness.TempConfigFile(t, harness.ClusterConfig(fmt.Sprintf(`bss:
  uri: %s/boot/v1
`, server.URL)))

		result := harness.RunCLI(t,
			"--config", configPath,
			"--token", token,
			"bss", "boot", "params", "add",
			"--mac", "00:de:ad:be:ef:00",
			"--kernel", "https://example.com/kernel",
		)
		harness.AssertRequestCount(t, server, 1)
		harness.AssertLastRequest(t, server, http.MethodPost, "/boot/v1/bootparameters")
		if result.ExitCode == 0 {
			t.Fatalf("expected non-zero exit code on 400 response, got 0\nstdout: %s\nstderr: %s", result.Stdout, result.Stderr)
		}
	})
}
