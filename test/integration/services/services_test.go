// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build integration

package services

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/openchami/ochami/test/integration/harness"
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
		name        string
		args        []string
		apiVersion  string
		wantPath    string
		wantHeaders map[string]string
		body        string
	}{
		{name: "service status", args: []string{"boot", "service", "status"}, wantPath: "/boot-service/health", body: `{"status":"ok"}`},
		{name: "node list", args: []string{"boot", "node", "list"}, wantPath: "/boot-service/nodes", body: `[]`},
		{name: "config list", args: []string{"boot", "config", "list"}, wantPath: "/boot-service/bootconfigurations", body: `[]`},
		{name: "bmc list", args: []string{"boot", "bmc", "list"}, wantPath: "/boot-service/bmcs", body: `[]`},
		{
			name:        "service status with config API version",
			args:        []string{"boot", "service", "status"},
			apiVersion:  "v1beta2",
			wantPath:    "/boot-service/health",
			wantHeaders: map[string]string{"Accept": "application/json;version=v1beta2"},
			body:        `{"status":"ok"}`,
		},
		{
			name:        "service status with flag API version",
			args:        []string{"boot", "--api-version", "v1beta3", "service", "status"},
			wantPath:    "/boot-service/health",
			wantHeaders: map[string]string{"Accept": "application/json;version=v1beta3"},
			body:        `{"status":"ok"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := harness.NewFakeHTTPServer(t, harness.NewServiceHandler(map[string]harness.ServiceResponse{
				"GET " + tc.wantPath: jsonResponse(tc.body),
			}))
			defer server.Close()

			configPath := harness.TempConfigFile(t, harness.ClusterConfig(fmt.Sprintf(`boot-service:
  uri: %s/boot-service%s
`, server.URL, apiVersionYAML(tc.apiVersion))))

			args := append([]string{"--config", configPath}, tc.args...)
			result := harness.RunCLI(t, args...)
			harness.AssertExitCode(t, result, 0)
			harness.AssertRequestCount(t, server, 1)
			harness.AssertLastRequest(t, server, http.MethodGet, tc.wantPath)
			for key, want := range tc.wantHeaders {
				harness.AssertLastRequestHeader(t, server, key, want)
			}
		})
	}
}

func TestMetadataServiceRequests(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		apiVersion  string
		wantPath    string
		wantHeaders map[string]string
		body        string
	}{
		{name: "service status", args: []string{"metadata", "service", "status"}, wantPath: "/metadata-service/health", body: `{"status":"ok"}`},
		{name: "group list", args: []string{"metadata", "group", "list"}, wantPath: "/metadata-service/groups", body: `[]`},
		{name: "instance list", args: []string{"metadata", "instance", "list"}, wantPath: "/metadata-service/instanceinfos", body: `[]`},
		{name: "defaults list", args: []string{"metadata", "defaults", "list"}, wantPath: "/metadata-service/clusterdefaultss", body: `[]`},
		{name: "peer list", args: []string{"metadata", "peer", "list"}, wantPath: "/metadata-service/wireguardpeers", body: `[]`},
		{
			name:        "service status with config API version",
			args:        []string{"metadata", "service", "status"},
			apiVersion:  "v1beta2",
			wantPath:    "/metadata-service/health",
			wantHeaders: map[string]string{"Accept": "application/json;version=v1beta2"},
			body:        `{"status":"ok"}`,
		},
		{
			name:        "service status with flag API version",
			args:        []string{"metadata", "--api-version", "v1beta3", "service", "status"},
			wantPath:    "/metadata-service/health",
			wantHeaders: map[string]string{"Accept": "application/json;version=v1beta3"},
			body:        `{"status":"ok"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := harness.NewFakeHTTPServer(t, harness.NewServiceHandler(map[string]harness.ServiceResponse{
				"GET " + tc.wantPath: jsonResponse(tc.body),
			}))
			defer server.Close()

			configPath := harness.TempConfigFile(t, harness.ClusterConfig(fmt.Sprintf(`metadata-service:
  uri: %s/metadata-service%s
`, server.URL, apiVersionYAML(tc.apiVersion))))

			args := append([]string{"--config", configPath}, tc.args...)
			result := harness.RunCLI(t, args...)
			harness.AssertExitCode(t, result, 0)
			harness.AssertRequestCount(t, server, 1)
			harness.AssertLastRequest(t, server, http.MethodGet, tc.wantPath)
			for key, want := range tc.wantHeaders {
				harness.AssertLastRequestHeader(t, server, key, want)
			}
		})
	}
}

func apiVersionYAML(apiVersion string) string {
	if apiVersion == "" {
		return ""
	}
	return fmt.Sprintf("\n  api-version: %s", apiVersion)
}

func jsonResponse(body string) harness.ServiceResponse {
	return harness.ServiceResponse{
		StatusCode: http.StatusOK,
		Body:       body,
		Headers:    map[string]string{"Content-Type": "application/json"},
	}
}

// TestSMDComponentWriteRequests verifies that write commands (POST/DELETE)
// against SMD produce requests in the form the service expects: correct HTTP
// method, path, and request body. These complement the read-only tests above.
func TestSMDComponentWriteRequests(t *testing.T) {
	const token = "test-token-abc123"

	t.Run("component add via flags sends POST with expected body", func(t *testing.T) {
		// SMD returns 201 Created with the created components on success.
		server := harness.NewFakeHTTPServer(t, harness.NewServiceHandler(map[string]harness.ServiceResponse{
			"POST /hsm/v2/State/Components": {
				StatusCode: http.StatusCreated,
				Body:       `{"Components":[{"ID":"x3000c1s7b56n0","NID":56}]}`,
				Headers:    map[string]string{"Content-Type": "application/json"},
			},
		}))
		defer server.Close()

		configPath := harness.TempConfigFile(t, harness.ClusterConfig(fmt.Sprintf(`smd:
  uri: %s/hsm/v2
`, server.URL)))

		result := harness.RunCLI(t,
			"--config", configPath,
			"--token", token,
			"smd", "component", "add",
			"--state", "Ready", "--enabled", "--role", "Compute", "--arch", "X86",
			"x3000c1s7b56n0", "56",
		)
		harness.AssertExitCode(t, result, 0)
		harness.AssertRequestCount(t, server, 1)
		harness.AssertLastRequest(t, server, http.MethodPost, "/hsm/v2/State/Components")
		// Verify the request body is well-formed and carries the component the
		// user specified via flags. Note: the flag-based path does not set NID
		// (only the -d payload path does), and the Component struct always
		// serializes an empty Type field.
		harness.AssertLastRequestJSONBody(t, server, `{
			"Components": [
				{"ID":"x3000c1s7b56n0","State":"Ready","Role":"Compute","Enabled":true,"Arch":"X86","Type":""}
			]
		}`)
		// A token was provided, so it must be forwarded as a Bearer header.
		harness.AssertLastRequestAuthToken(t, server, token)
	})

	t.Run("component add via -d payload sends POST with that body", func(t *testing.T) {
		server := harness.NewFakeHTTPServer(t, harness.NewServiceHandler(map[string]harness.ServiceResponse{
			"POST /hsm/v2/State/Components": {
				StatusCode: http.StatusCreated,
				Body:       `{"Components":[]}`,
				Headers:    map[string]string{"Content-Type": "application/json"},
			},
		}))
		defer server.Close()

		configPath := harness.TempConfigFile(t, harness.ClusterConfig(fmt.Sprintf(`smd:
  uri: %s/hsm/v2
`, server.URL)))

		payload := `{"Components":[{"ID":"x3000c1s7b56n1","NID":57,"State":"Ready","Role":"Compute","Enabled":true,"Arch":"X86"}]}`
		result := harness.RunCLI(t,
			"--config", configPath,
			"--token", token,
			"smd", "component", "add",
			"-d", payload,
		)
		harness.AssertExitCode(t, result, 0)
		harness.AssertRequestCount(t, server, 1)
		harness.AssertLastRequest(t, server, http.MethodPost, "/hsm/v2/State/Components")
		// The CLI unmarshals the payload into its Component struct and
		// re-marshals it, so the user-provided fields (including NID) must be
		// preserved on the wire (an empty Type field is added by the struct).
		harness.AssertLastRequestJSONBody(t, server, `{
			"Components": [
				{"ID":"x3000c1s7b56n1","NID":57,"State":"Ready","Role":"Compute","Enabled":true,"Arch":"X86","Type":""}
			]
		}`)
	})

	t.Run("component delete sends DELETE to per-xname path", func(t *testing.T) {
		server := harness.NewFakeHTTPServer(t, harness.NewServiceHandler(map[string]harness.ServiceResponse{
			"DELETE /hsm/v2/State/Components/x3000c1s7b56n0": {
				StatusCode: http.StatusOK,
				Body:       `{"code":0,"message":"deleted 1 entry"}`,
				Headers:    map[string]string{"Content-Type": "application/json"},
			},
		}))
		defer server.Close()

		configPath := harness.TempConfigFile(t, harness.ClusterConfig(fmt.Sprintf(`smd:
  uri: %s/hsm/v2
`, server.URL)))

		// --no-confirm bypasses the interactive deletion prompt.
		result := harness.RunCLI(t,
			"--config", configPath,
			"--token", token,
			"smd", "component", "delete", "--no-confirm",
			"x3000c1s7b56n0",
		)
		harness.AssertExitCode(t, result, 0)
		harness.AssertRequestCount(t, server, 1)
		harness.AssertLastRequest(t, server, http.MethodDelete, "/hsm/v2/State/Components/x3000c1s7b56n0")
		harness.AssertLastRequestAuthToken(t, server, token)
	})
}

// TestSMDComponentErrorConditions verifies the CLI surfaces service error
// responses as a non-zero exit code rather than reporting success.
func TestSMDComponentErrorConditions(t *testing.T) {
	const token = "test-token-abc123"

	t.Run("component add fails on 500 response", func(t *testing.T) {
		server := harness.NewFakeHTTPServer(t, harness.NewServiceHandler(map[string]harness.ServiceResponse{
			"POST /hsm/v2/State/Components": {
				StatusCode: http.StatusInternalServerError,
				Body:       `{"error":"boom"}`,
				Headers:    map[string]string{"Content-Type": "application/json"},
			},
		}))
		defer server.Close()

		configPath := harness.TempConfigFile(t, harness.ClusterConfig(fmt.Sprintf(`smd:
  uri: %s/hsm/v2
`, server.URL)))

		result := harness.RunCLI(t,
			"--config", configPath,
			"--token", token,
			"smd", "component", "add",
			"x3000c1s7b56n0", "56",
		)
		// The request should still have been sent in the correct form, but the
		// CLI must report failure.
		harness.AssertRequestCount(t, server, 1)
		harness.AssertLastRequest(t, server, http.MethodPost, "/hsm/v2/State/Components")
		if result.ExitCode == 0 {
			t.Fatalf("expected non-zero exit code on 500 response, got 0\nstdout: %s\nstderr: %s", result.Stdout, result.Stderr)
		}
	})

	t.Run("component get fails on 404 response", func(t *testing.T) {
		// Empty handler map means every path returns the default 404.
		server := harness.NewFakeHTTPServer(t, harness.NewServiceHandler(map[string]harness.ServiceResponse{}))
		defer server.Close()

		configPath := harness.TempConfigFile(t, harness.ClusterConfig(fmt.Sprintf(`smd:
  uri: %s/hsm/v2
`, server.URL)))

		result := harness.RunCLI(t,
			"--config", configPath,
			"--token", token,
			"smd", "component", "get",
		)
		harness.AssertRequestCount(t, server, 1)
		if result.ExitCode == 0 {
			t.Fatalf("expected non-zero exit code on 404 response, got 0\nstdout: %s\nstderr: %s", result.Stdout, result.Stderr)
		}
	})
}
