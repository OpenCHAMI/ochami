// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build integration

package services

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/OpenCHAMI/ochami/test/integration/harness"
)

func TestSMDRequests(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
		body     string
	}{
		{
			name:     "service status",
			args:     []string{"smd", "service", "status"},
			wantPath: "/hsm/v2/service/ready",
			body:     `{"status":"ok"}`,
		},
		{
			name:     "component get",
			args:     []string{"smd", "component", "get"},
			wantPath: "/hsm/v2/State/Components",
			body:     `{"Components":[]}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := harness.NewFakeHTTPServer(t, harness.NewServiceHandler(map[string]harness.ServiceResponse{
				"GET " + tc.wantPath: jsonResponse(tc.body),
			}))
			defer server.Close()

			configPath := harness.TempConfigFile(t, harness.ClusterConfig(fmt.Sprintf(`smd:
  uri: %s/hsm/v2
`, server.URL)))

			args := append([]string{"--config", configPath}, tc.args...)
			result := harness.RunCLI(t, args...)
			harness.AssertExitCode(t, result, 0)
			harness.AssertRequestCount(t, server, 1)
			harness.AssertLastRequest(t, server, http.MethodGet, tc.wantPath)
		})
	}
}

func TestBootServiceRequests(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
		body     string
	}{
		{name: "service status", args: []string{"boot", "service", "status"}, wantPath: "/boot-service/health", body: `{"status":"ok"}`},
		{name: "node list", args: []string{"boot", "node", "list"}, wantPath: "/boot-service/nodes", body: `[]`},
		{name: "config list", args: []string{"boot", "config", "list"}, wantPath: "/boot-service/bootconfigurations", body: `[]`},
		{name: "bmc list", args: []string{"boot", "bmc", "list"}, wantPath: "/boot-service/bmcs", body: `[]`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := harness.NewFakeHTTPServer(t, harness.NewServiceHandler(map[string]harness.ServiceResponse{
				"GET " + tc.wantPath: jsonResponse(tc.body),
			}))
			defer server.Close()

			configPath := harness.TempConfigFile(t, harness.ClusterConfig(fmt.Sprintf(`boot-service:
  uri: %s/boot-service
`, server.URL)))

			args := append([]string{"--config", configPath}, tc.args...)
			result := harness.RunCLI(t, args...)
			harness.AssertExitCode(t, result, 0)
			harness.AssertRequestCount(t, server, 1)
			harness.AssertLastRequest(t, server, http.MethodGet, tc.wantPath)
		})
	}
}

func TestMetadataServiceRequests(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
		body     string
	}{
		{name: "service status", args: []string{"metadata", "service", "status"}, wantPath: "/metadata-service/health", body: `{"status":"ok"}`},
		{name: "group list", args: []string{"metadata", "group", "list"}, wantPath: "/metadata-service/groups", body: `[]`},
		{name: "instance list", args: []string{"metadata", "instance", "list"}, wantPath: "/metadata-service/instanceinfos", body: `[]`},
		{name: "defaults list", args: []string{"metadata", "defaults", "list"}, wantPath: "/metadata-service/clusterdefaultss", body: `[]`},
		{name: "peer list", args: []string{"metadata", "peer", "list"}, wantPath: "/metadata-service/wireguardpeers", body: `[]`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := harness.NewFakeHTTPServer(t, harness.NewServiceHandler(map[string]harness.ServiceResponse{
				"GET " + tc.wantPath: jsonResponse(tc.body),
			}))
			defer server.Close()

			configPath := harness.TempConfigFile(t, harness.ClusterConfig(fmt.Sprintf(`metadata-service:
  uri: %s/metadata-service
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
