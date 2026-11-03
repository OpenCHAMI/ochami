// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_errors_test.go covers HTTP- and config-failure paths for the commands
// directly under "smd", and the empty or invalid IDs every "smd" delete
// command rejects before sending a request; see smd_test.go and the
// smd_<resource>_test.go files for the success paths these mirror.

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/config"
)

// TestSMDDeprecatedStatus_HTTPError verifies that an unsuccessful HTTP response from the
// SMD service status endpoint resolves to CodeHTTP.
func TestSMDDeprecatedStatus_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "service", "status", "--ignore-config", "--uri", srv.URL)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDServiceStatus_DuplicateCluster verifies that "smd service status"
// warns about and then fails with a config error, without contacting SMD,
// when the cluster selected through default-cluster or --cluster is defined
// more than once in the config file, even when --uri is also passed.
func TestSMDServiceStatus_DuplicateCluster(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
	}))
	defer srv.Close()
	cfg := writeTempConfig(t, duplicateClusterConfig)

	for _, args := range [][]string{
		{"--config", cfg, "smd", "service", "status"},
		{"--config", cfg, "smd", "service", "status", "--cluster", "demo"},
		{"--config", cfg, "smd", "service", "status", "--uri", srv.URL},
	} {
		res := runOchamiWithRuntime(t, args...)
		if !errors.As(res.err, &config.ErrDuplicateCluster{}) {
			t.Errorf("%v: error = %v, want ErrDuplicateCluster", args, res.err)
		}
		if res.exitCode != cli.CodeConfig {
			t.Errorf("%v: exit code = %d, want %d (%s)", args, res.exitCode, cli.CodeConfig, cli.CodeName(cli.CodeConfig))
		}
		if strings.Count(res.stdout, "WRN") != 1 || !strings.Contains(res.stdout, `cluster "demo" is defined 2 times`) {
			t.Errorf("%v: output = %q, want one warning about cluster demo", args, res.stdout)
		}
	}
	if n := requests.Load(); n != 0 {
		t.Errorf("SMD received %d requests, want 0", n)
	}
}

// TestSMDDelete_RejectsEmptyData verifies an explicit empty payload cannot
// turn a requested deletion into a silent no-op.
func TestSMDDelete_RejectsEmptyData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		payload string
		wantMsg string
	}{
		{name: "interface", command: "iface", payload: `[]`, wantMsg: "payload contained no ethernet interfaces to delete"},
		{name: "group", command: "group", payload: `[]`, wantMsg: "payload contained no groups to delete"},
		{name: "redfish endpoint", command: "rfe", payload: `{"RedfishEndpoints":[]}`, wantMsg: "payload contained no redfish endpoints to delete"},
		{name: "component endpoint", command: "compep", payload: `[]`, wantMsg: "payload contained no component endpoints to delete"},
		{name: "component", command: "component", payload: `{"Components":[]}`, wantMsg: "payload contained no components to delete"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var deletes int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					deletes++
				}
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "smd", "--ignore-config", tc.command, "delete",
				"--uri", srv.URL, "--token", "t", "--no-confirm", "-d", tc.payload)
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodeUsage {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
			}
			if !strings.Contains(res.err.Error(), tc.wantMsg) {
				t.Errorf("error = %q, want it to contain %q", res.err.Error(), tc.wantMsg)
			}
			if deletes != 0 {
				t.Errorf("DELETE count = %d, want 0 (no request should be made for an empty payload)", deletes)
			}
		})
	}
}

// TestSMDDelete_RejectsInvalidPayloadIDs verifies that a payload item whose ID
// is blank or is not a single path segment, which would otherwise address the
// whole collection, fails the delete with CodeUsage before any request is
// made, including for the payload's valid items.
func TestSMDDelete_RejectsInvalidPayloadIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
		payload string
		wantMsg string
	}{
		{name: "component without ID", command: "component", payload: `{"Components":[{}]}`, wantMsg: "component ID cannot be blank (item 1 of 1)"},
		{name: "component with dot ID", command: "component", payload: `{"Components":[{"ID":"."}]}`, wantMsg: `component ID "." is not a single path segment`},
		{name: "component after a valid one", command: "component", payload: `{"Components":[{"ID":"x0c0s0b0n0"},{"ID":"x0/.."}]}`, wantMsg: "(item 2 of 2)"},
		{name: "interface without ID", command: "iface", payload: `[{}]`, wantMsg: "ethernet interface ID cannot be blank"},
		{name: "interface with only a MAC address", command: "iface", payload: `[{"MACAddress":"de:ad:be:ef:00:01"}]`, wantMsg: "delete does not derive an ID from MACAddress"},
		{name: "group without label", command: "group", payload: `[{}]`, wantMsg: "group label cannot be blank"},
		{name: "redfish endpoint without ID", command: "rfe", payload: `{"RedfishEndpoints":[{}]}`, wantMsg: "redfish endpoint ID cannot be blank"},
		{name: "component endpoint without ID", command: "compep", payload: `[{}]`, wantMsg: "component endpoint ID cannot be blank"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var deletes int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					deletes++
				}
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "smd", "--ignore-config", tc.command, "delete",
				"--uri", srv.URL, "--token", "t", "--no-confirm", "-d", tc.payload)
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodeUsage {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
			}
			if !strings.Contains(res.err.Error(), tc.wantMsg) {
				t.Errorf("error = %q, want it to contain %q", res.err.Error(), tc.wantMsg)
			}
			if deletes != 0 {
				t.Errorf("DELETE count = %d, want 0", deletes)
			}
		})
	}
}

// TestSMDDelete_RejectsInvalidArgumentIDs verifies that a positional ID that
// is blank or is not a single path segment fails the delete with CodeUsage
// before any request is made, including for the valid arguments.
func TestSMDDelete_RejectsInvalidArgumentIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantMsg string
	}{
		{name: "blank component", args: []string{"component", "delete", ""}, wantMsg: "component ID cannot be blank"},
		{name: "component after a valid one", args: []string{"component", "delete", "x0c0s0b0n0", ".."}, wantMsg: `component ID ".." is not a single path segment (item 2 of 2)`},
		{name: "interface with slash", args: []string{"iface", "delete", "x0/.."}, wantMsg: "ethernet interface ID"},
		{name: "blank group", args: []string{"group", "delete", ""}, wantMsg: "group label cannot be blank"},
		{name: "blank redfish endpoint", args: []string{"rfe", "delete", " "}, wantMsg: "redfish endpoint ID cannot be blank"},
		{name: "dot component endpoint", args: []string{"compep", "delete", "."}, wantMsg: "component endpoint ID"},
		{name: "blank member group", args: []string{"group", "member", "delete", "", "x0c0s0b0n0"}, wantMsg: "group label cannot be blank"},
		{name: "blank group member", args: []string{"group", "member", "delete", "compute", ""}, wantMsg: "group member cannot be blank"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var deletes int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					deletes++
				}
			}))
			defer srv.Close()

			args := append([]string{"smd", "--ignore-config"}, tc.args...)
			args = append(args, "--uri", srv.URL, "--token", "t", "--no-confirm")
			res := runOchamiWithRuntime(t, args...)
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodeUsage {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
			}
			if !strings.Contains(res.err.Error(), tc.wantMsg) {
				t.Errorf("error = %q, want it to contain %q", res.err.Error(), tc.wantMsg)
			}
			if deletes != 0 {
				t.Errorf("DELETE count = %d, want 0", deletes)
			}
		})
	}
}
