// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build integration

package legacy

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/OpenCHAMI/ochami/test/integration/harness"
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
