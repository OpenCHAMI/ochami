// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// commands_errors_test.go covers error arms shared across the command tree, as
// tables over the commands they apply to, and asserts their exact exit codes:
// missing base URI (CodeConfig), missing/invalid token (CodeAuth/CodePayload),
// invalid CA cert, network/HTTP transport failures, malformed/extra-arg payload
// handling, arguments a client rejects before sending a request (CodeUsage),
// output-writer failures, rejecting a missing token before any request,
// commands built and run without an invocation runtime, and malformed success
// bodies. Per-family error arms live in each family's own _errors_test.go file.

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	bootcmd "github.com/openchami/ochami/cmd/boot"
	bsscmd "github.com/openchami/ochami/cmd/bss"
	cloudinitcmd "github.com/openchami/ochami/cmd/cloud_init"
	configcmd "github.com/openchami/ochami/cmd/config"
	metadatacmd "github.com/openchami/ochami/cmd/metadata"
	pcscmd "github.com/openchami/ochami/cmd/pcs"
	rcscmd "github.com/openchami/ochami/cmd/rcs"
	smdcmd "github.com/openchami/ochami/cmd/smd"
	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/config"
)

// TestGetClient_NoBaseURI verifies that each listed command fails with
// CodeConfig when no base URI is configured for its service.
func TestGetClient_NoBaseURI(t *testing.T) {
	t.Parallel()

	cases := [][]string{
		// cloud-init
		{"cloud-init", "group", "get", "raw"},
		{"cloud-init", "group", "get", "config"},
		{"cloud-init", "group", "get", "meta-data"},
		{"cloud-init", "group", "add", "-d", `[{"name":"c"}]`},
		{"cloud-init", "group", "set", "-d", `[{"name":"c"}]`},
		{"cloud-init", "group", "render", "compute", "x0c0s0b0n0"},
		{"cloud-init", "node", "get", "meta-data", "x0c0s0b0n0"},
		{"cloud-init", "node", "get", "user-data", "x0c0s0b0n0"},
		{"cloud-init", "node", "get", "vendor-data", "x0c0s0b0n0"},
		{"cloud-init", "node", "get", "group", "x0c0s0b0n0", "compute"},
		{"cloud-init", "node", "set", "-d", `[{"id":"x0"}]`},
		{"cloud-init", "defaults", "set", "-d", `{"cluster-name":"c"}`},
		// smd group
		{"smd", "group", "get"},
		{"smd", "group", "add", "compute"},
		{"smd", "group", "update", "--description", "d", "compute"},
		{"smd", "group", "delete", "--no-confirm", "compute"},
		{"smd", "group", "membership"},
		{"smd", "group", "member", "get", "compute"},
		{"smd", "group", "member", "add", "compute", "x0"},
		{"smd", "group", "member", "set", "compute", "x0"},
		{"smd", "group", "member", "delete", "--no-confirm", "compute", "x0"},
		// smd iface
		{"smd", "iface", "get"},
		{"smd", "iface", "add", "x0", "de:ad:be:ef:00:00", "NMN,172.16.0.1"},
		{"smd", "iface", "delete", "--no-confirm", "decafc0ffeee"},
		// smd rfe
		{"smd", "rfe", "get"},
		{"smd", "rfe", "add", "x0", "n", "172.16.0.1", "de:ad:be:ef:00:00"},
		{"smd", "rfe", "delete", "--no-confirm", "x0"},
		// smd compep / component
		{"smd", "compep", "get"},
		{"smd", "compep", "delete", "--no-confirm", "x0"},
		{"smd", "component", "get"},
		{"smd", "component", "add", "x0", "1"},
		{"smd", "component", "delete", "--no-confirm", "x0"},
		// bss
		{"bss", "boot", "params", "get"},
		{"bss", "boot", "params", "add", "--mac", "de:ad:be:ef:00:00", "--kernel", "http://k"},
		{"bss", "boot", "params", "set", "--mac", "de:ad:be:ef:00:00", "--kernel", "http://k"},
		{"bss", "boot", "params", "update", "--mac", "de:ad:be:ef:00:00", "--kernel", "http://k"},
		{"bss", "boot", "params", "delete", "--no-confirm", "--mac", "de:ad:be:ef:00:00", "--kernel", "http://k"},
		{"bss", "boot", "script", "get", "--mac", "de:ad:be:ef:00:00"},
		{"bss", "boot", "image", "set", "--mac", "de:ad:be:ef:00:00", "http://img"},
		{"bss", "hosts", "get"},
		{"bss", "history", "--xname", "x0"},
		// metadata (all four types, all verbs)
		{"metadata", "defaults", "list"},
		{"metadata", "defaults", "get", "uid"},
		{"metadata", "defaults", "add", "-d", `{"name":"n"}`},
		{"metadata", "defaults", "set", "uid", "-d", `{"name":"n"}`},
		{"metadata", "defaults", "patch", "uid", "-d", `{"name":"n"}`},
		{"metadata", "defaults", "delete", "--no-confirm", "uid"},
		{"metadata", "group", "list"},
		{"metadata", "instance", "list"},
		{"metadata", "peer", "list"},
		// boot (all three types)
		{"boot", "bmc", "list"},
		{"boot", "bmc", "get", "uid"},
		{"boot", "bmc", "add", "-d", `{"name":"n"}`},
		{"boot", "bmc", "set", "uid", "-d", `{"name":"n"}`},
		{"boot", "bmc", "patch", "uid", "-d", `{"name":"n"}`},
		{"boot", "bmc", "delete", "--no-confirm", "uid"},
		{"boot", "config", "list"},
		{"boot", "node", "list"},
		// pcs
		{"pcs", "transition", "list"},
		{"pcs", "transition", "show", "id"},
		{"pcs", "transition", "abort", "id"},
		{"pcs", "transition", "start", "--xname", "x0", "on"},
		{"pcs", "status", "list"},
		{"pcs", "status", "show", "x0"},
	}
	for _, args := range cases {
		name := ""
		for _, a := range args {
			name += a + "_"
		}
		t.Run(name, func(t *testing.T) {
			full := append(args, "--ignore-config")
			res := runOchamiWithRuntime(t, full...)
			if res.err == nil {
				t.Fatalf("expected an error without a base URI, got nil")
			}
			if res.exitCode != cli.CodeConfig {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeConfig, cli.CodeName(cli.CodeConfig))
			}
		})
	}
}

// TestHandleToken_AuthRequired verifies that each listed service command fails
// with CodeAuth when its cluster, configured in the config file, enables auth
// and no token is available.
func TestHandleToken_AuthRequired(t *testing.T) {
	t.Parallel()

	srv := okJSONServer(t)
	defer srv.Close()

	cfg := writeTempConfig(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: `+srv.URL+`
    enable-auth: true
`)

	cases := [][]string{
		{"smd", "group", "get"},
		{"smd", "group", "add", "compute"},
		{"smd", "group", "update", "--description", "d", "compute"},
		{"smd", "group", "member", "get", "compute"},
		{"smd", "group", "member", "add", "compute", "x0"},
		{"smd", "group", "member", "set", "compute", "x0"},
		{"smd", "iface", "add", "x0", "de:ad:be:ef:00:00", "NMN,172.16.0.1"},
		{"smd", "rfe", "get"},
		{"smd", "rfe", "add", "x0", "n", "172.16.0.1", "de:ad:be:ef:00:00"},
		{"smd", "compep", "get"},
		{"smd", "component", "add", "x0", "1"},
		{"bss", "boot", "params", "get"},
		{"bss", "boot", "params", "add", "--mac", "de:ad:be:ef:00:00", "--kernel", "http://k"},
		{"bss", "boot", "params", "set", "--mac", "de:ad:be:ef:00:00", "--kernel", "http://k"},
		{"bss", "boot", "params", "update", "--mac", "de:ad:be:ef:00:00", "--kernel", "http://k"},
		{"metadata", "defaults", "list"},
		{"metadata", "defaults", "add", "-d", `{"name":"n"}`},
		{"metadata", "defaults", "set", "uid", "-d", `{"name":"n"}`},
		{"metadata", "defaults", "patch", "uid", "-d", `{"name":"n"}`},
		{"metadata", "group", "list"},
		{"metadata", "instance", "list"},
		{"metadata", "peer", "list"},
		{"boot", "bmc", "list"},
		{"boot", "bmc", "add", "-d", `{"name":"n"}`},
		{"boot", "config", "list"},
		{"boot", "node", "list"},
		{"pcs", "transition", "list"},
		{"pcs", "status", "list"},
	}
	for _, args := range cases {
		name := ""
		for _, a := range args {
			name += a + "_"
		}
		t.Run(name, func(t *testing.T) {
			full := append([]string{"--config", cfg}, args...)
			res := runOchamiWithRuntime(t, full...)
			if res.err == nil {
				t.Fatalf("expected an auth error, got nil")
			}
			if res.exitCode != cli.CodeAuth {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeAuth, cli.CodeName(cli.CodeAuth))
			}
		})
	}
}

// TestServiceCommands_NetworkError verifies that each listed read command,
// including those that fetch several items, fails with CodeNetwork when its
// service can't be reached.
func TestServiceCommands_NetworkError(t *testing.T) {
	t.Parallel()

	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	cases := []struct {
		args     []string
		wantCode int
	}{
		{args: []string{"smd", "group", "get"}, wantCode: cli.CodeNetwork},
		{args: []string{"smd", "iface", "get"}, wantCode: cli.CodeNetwork},
		{args: []string{"smd", "rfe", "get"}, wantCode: cli.CodeNetwork},
		{args: []string{"smd", "compep", "get"}, wantCode: cli.CodeNetwork},
		{args: []string{"smd", "component", "get"}, wantCode: cli.CodeNetwork},
		{args: []string{"bss", "boot", "params", "get"}, wantCode: cli.CodeNetwork},
		{args: []string{"bss", "boot", "script", "get", "--mac", "de:ad:be:ef:00:00"}, wantCode: cli.CodeNetwork},
		{args: []string{"bss", "hosts", "get"}, wantCode: cli.CodeNetwork},
		{args: []string{"bss", "history", "--xname", "x0"}, wantCode: cli.CodeNetwork},
		{args: []string{"cloud-init", "group", "get", "raw"}, wantCode: cli.CodeNetwork},
		{args: []string{"cloud-init", "node", "get", "meta-data", "x0c0s0b0n0"}, wantCode: cli.CodeNetwork},
		{args: []string{"pcs", "transition", "list"}, wantCode: cli.CodeNetwork},
		{args: []string{"pcs", "status", "list"}, wantCode: cli.CodeNetwork},
		{args: []string{"pcs", "service", "status"}, wantCode: cli.CodeNetwork},
	}
	for _, tc := range cases {
		name := ""
		for _, a := range tc.args {
			name += a + "_"
		}
		t.Run(name, func(t *testing.T) {
			full := append(tc.args, "--ignore-config", "--uri", url, "--token", "t")
			res := runOchamiWithRuntime(t, full...)
			if res.err == nil {
				t.Fatalf("expected a network error, got nil")
			}
			if res.exitCode != tc.wantCode {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, tc.wantCode, cli.CodeName(tc.wantCode))
			}
		})
	}
}

// TestServiceCommands_HTTPError verifies that each listed read command fails
// with CodeHTTP when its service responds with HTTP 500.
func TestServiceCommands_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	cases := [][]string{
		{"smd", "group", "get"},
		{"smd", "iface", "get"},
		{"smd", "rfe", "get"},
		{"smd", "compep", "get"},
		{"smd", "component", "get"},
		{"bss", "boot", "params", "get"},
		{"bss", "boot", "script", "get", "--mac", "de:ad:be:ef:00:00"},
		{"bss", "hosts", "get"},
		{"bss", "history", "--xname", "x0"},
		{"cloud-init", "group", "get", "raw"},
		{"cloud-init", "node", "get", "meta-data", "x0c0s0b0n0"},
		{"pcs", "transition", "list"},
		{"pcs", "status", "list"},
	}
	for _, args := range cases {
		name := ""
		for _, a := range args {
			name += a + "_"
		}
		t.Run(name, func(t *testing.T) {
			full := append(args, "--ignore-config", "--uri", srv.URL, "--token", "t")
			res := runOchamiWithRuntime(t, full...)
			if res.err == nil {
				t.Fatalf("expected an HTTP error, got nil")
			}
			if res.exitCode != cli.CodeHTTP {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
			}
		})
	}
}

// TestCommands_RejectInvalidCACert verifies that each listed command fails with
// CodePayload when --cacert names a file that doesn't exist.
func TestCommands_RejectInvalidCACert(t *testing.T) {
	t.Parallel()

	srv := okJSONServer(t)
	defer srv.Close()

	cases := [][]string{
		{"smd", "group", "get"},
		{"smd", "iface", "get"},
		{"smd", "rfe", "get"},
		{"smd", "compep", "get"},
		{"smd", "component", "get"},
		{"bss", "boot", "params", "get"},
		{"bss", "hosts", "get"},
		{"cloud-init", "group", "get", "raw"},
		{"cloud-init", "node", "get", "meta-data", "x0c0s0b0n0"},
		{"metadata", "defaults", "list"},
		{"metadata", "group", "list"},
		{"metadata", "instance", "list"},
		{"metadata", "peer", "list"},
		{"boot", "bmc", "list"},
		{"boot", "config", "list"},
		{"boot", "node", "list"},
		{"pcs", "transition", "list"},
		{"pcs", "status", "list"},
		{"rcs", "console", "list"},
		{"smd", "group", "add", "compute"},
		{"smd", "rfe", "add", "x0", "n", "172.16.0.1", "de:ad:be:ef:00:00"},
		{"discover", "static", "-d", discoveryPayload},
	}
	for _, args := range cases {
		name := ""
		for _, a := range args {
			name += a + "_"
		}
		t.Run(name, func(t *testing.T) {
			full := append(args, "--ignore-config", "--uri", srv.URL, "--token", "t",
				"--cacert", "/no/such/ca.pem")
			res := runOchamiWithRuntime(t, full...)
			if res.err == nil {
				t.Fatalf("expected an error for invalid --cacert, got nil")
			}
			if res.exitCode != cli.CodePayload {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
			}
		})
	}
}

// TestCommands_MalformedPayload verifies that each listed command that accepts
// -d fails with CodePayload for a payload that isn't valid JSON.
func TestCommands_MalformedPayload(t *testing.T) {
	t.Parallel()

	srv := okJSONServer(t)
	defer srv.Close()

	cases := [][]string{
		{"smd", "iface", "add"},
		{"smd", "rfe", "add"},
		{"smd", "component", "add"},
		{"smd", "compep", "delete", "--no-confirm"},
		{"smd", "iface", "delete", "--no-confirm"},
		{"smd", "rfe", "delete", "--no-confirm"},
		{"cloud-init", "group", "add"},
		{"cloud-init", "group", "set"},
		{"cloud-init", "node", "set"},
		{"cloud-init", "defaults", "set"},
	}
	for _, args := range cases {
		name := ""
		for _, a := range args {
			name += a + "_"
		}
		t.Run(name, func(t *testing.T) {
			full := append(args, "--ignore-config", "--uri", srv.URL, "--token", "t", "-d", "not json")
			res := runOchamiWithRuntime(t, full...)
			if res.err == nil {
				t.Fatalf("expected a payload error, got nil")
			}
			if res.exitCode != cli.CodePayload {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
			}
		})
	}
}

// TestCommands_DataWithExtraArgs verifies that each listed delete command
// accepts -d together with extra positional arguments, succeeds, and logs a
// warning that the extra arguments are ignored.
func TestCommands_DataWithExtraArgs(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	cases := []struct {
		args []string
		data string
	}{
		{[]string{"smd", "iface", "delete", "--no-confirm"}, `[{"ID":"decafc0ffeee"}]`},
		{[]string{"smd", "rfe", "delete", "--no-confirm"}, `{"RedfishEndpoints":[{"ID":"x0"}]}`},
		{[]string{"smd", "compep", "delete", "--no-confirm"}, `[{"ID":"x0"}]`},
		{[]string{"smd", "component", "delete", "--no-confirm"}, `{"Components":[{"ID":"x0"}]}`},
	}
	for _, tc := range cases {
		name := ""
		for _, a := range tc.args {
			name += a + "_"
		}
		t.Run(name, func(t *testing.T) {
			full := append(tc.args, "--ignore-config", "--uri", srv.URL, "--token", "t", "-d", tc.data, "extra-arg")
			res := runOchamiWithRuntime(t, full...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if !strings.Contains(res.stdout, "ignoring extra arguments") {
				t.Errorf("output = %q, want a warning that the extra arguments are ignored", res.stdout)
			}
		})
	}
}

// TestWriteCommands_NetworkError verifies that each listed SMD and cloud-init
// write command, including the multi-item ones, fails with CodeNetwork when its
// service can't be reached.
func TestWriteCommands_NetworkError(t *testing.T) {
	t.Parallel()

	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	cases := [][]string{
		{"smd", "iface", "add", "x0", "de:ad:be:ef:00:00", "NMN,172.16.0.1"},
		{"smd", "rfe", "add", "x0", "n", "172.16.0.1", "de:ad:be:ef:00:00"},
		{"smd", "component", "add", "x0", "1"},
		{"smd", "group", "add", "compute"},
		{"smd", "group", "update", "--description", "d", "compute"},
		{"smd", "group", "delete", "--no-confirm", "compute"},
		{"smd", "iface", "delete", "--no-confirm", "decafc0ffeee"},
		{"smd", "rfe", "delete", "--no-confirm", "x0"},
		{"smd", "compep", "delete", "--no-confirm", "x0"},
		{"smd", "component", "delete", "--no-confirm", "x0"},
		{"cloud-init", "group", "add", "-d", `[{"name":"c"}]`},
		{"cloud-init", "group", "set", "-d", `[{"name":"c"}]`},
		{"cloud-init", "group", "delete", "--no-confirm", "compute"},
		{"cloud-init", "node", "set", "-d", `[{"id":"x0"}]`},
	}
	for _, args := range cases {
		name := ""
		for _, a := range args {
			name += a + "_"
		}
		t.Run(name, func(t *testing.T) {
			full := append(args, "--ignore-config", "--uri", url, "--token", "t")
			res := runOchamiWithRuntime(t, full...)
			if res.err == nil {
				t.Fatalf("expected a network error, got nil")
			}
			if res.exitCode != cli.CodeNetwork {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
			}
		})
	}
}

// TestMetadataBootWrite_NetworkError verifies that the add, set, patch, delete,
// and get commands fail with CodeNetwork for every metadata and boot resource
// type when the service can't be reached.
func TestMetadataBootWrite_NetworkError(t *testing.T) {
	t.Parallel()

	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	var cases [][]string
	for _, typ := range []string{"defaults", "group", "instance", "peer"} {
		cases = append(cases,
			[]string{"metadata", typ, "add", "-d", `{"name":"n"}`},
			[]string{"metadata", typ, "set", "uid", "-d", `{"name":"n"}`},
			[]string{"metadata", typ, "patch", "uid", "-d", `{"name":"n"}`},
			[]string{"metadata", typ, "delete", "--no-confirm", "uid"},
			[]string{"metadata", typ, "get", "uid"},
		)
	}
	for _, typ := range []string{"bmc", "config", "node"} {
		cases = append(cases,
			[]string{"boot", typ, "add", "-d", `{"name":"n"}`},
			[]string{"boot", typ, "set", "uid", "-d", `{"name":"n"}`},
			[]string{"boot", typ, "patch", "uid", "-d", `{"name":"n"}`},
			[]string{"boot", typ, "delete", "--no-confirm", "uid"},
			[]string{"boot", typ, "get", "uid"},
		)
	}
	for _, args := range cases {
		name := ""
		for _, a := range args {
			name += a + "_"
		}
		t.Run(name, func(t *testing.T) {
			full := append(args, "--ignore-config", "--uri", url, "--token", "t")
			res := runOchamiWithRuntime(t, full...)
			if res.err == nil {
				t.Fatalf("expected a network error, got nil")
			}
			if res.exitCode != cli.CodeNetwork {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
			}
		})
	}
}

// TestBatchDelete_MixedFailures verifies that a batch whose item failures
// differ in kind (an HTTP error for one item, a dropped connection for
// another) exits with CodeMixed.
func TestBatchDelete_MixedFailures(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/x0c0s0b0n0"):
			http.Error(w, "not found", http.StatusNotFound)
		case strings.HasSuffix(r.URL.Path, "/x0c0s1b0n0"):
			if conn, _, err := w.(http.Hijacker).Hijack(); err == nil {
				_ = conn.Close()
			}
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "smd", "component", "delete", "--no-confirm",
		"--uri", srv.URL, "--token", "t", "x0c0s0b0n0", "x0c0s1b0n0", "x0c0s2b0n0")
	if res.exitCode != cli.CodeMixed {
		t.Errorf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodeMixed, cli.CodeName(cli.CodeMixed))
	}
}

// TestCommands_RejectInvalidArguments verifies that arguments a service client
// rejects before sending any request (a blank group label or node ID) resolve
// to CodeUsage. Nothing listens on the URI, so a request that was sent anyway
// would fail with CodeNetwork instead.
func TestCommands_RejectInvalidArguments(t *testing.T) {
	t.Parallel()

	cases := [][]string{
		{"smd", "group", "member", "add", "", "x0c0s0b0n0"},
		{"smd", "group", "member", "delete", "--no-confirm", "", "x0c0s0b0n0"},
		{"smd", "group", "member", "get", ""},
		{"smd", "group", "member", "set", "", "x0c0s0b0n0"},
		{"cloud-init", "node", "get", "group", "", "compute"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			full := append(args, "--ignore-config", "--uri", "http://127.0.0.1:1", "--token", "t")
			res := runOchamiWithRuntime(t, full...)
			if res.exitCode != cli.CodeUsage {
				t.Errorf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
			}
		})
	}
}

// TestCommands_PropagateOutputFailures verifies that each listed command fails
// with CodePayload, reporting the writer's error, when its output can't be
// written.
func TestCommands_PropagateOutputFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		args func(string) []string
	}{
		{name: "boot bmc get", body: `{}`, args: func(uri string) []string { return []string{"boot", "bmc", "get", "id", "--uri", uri, "--token", "t"} }},
		{name: "boot bmc list", body: `[]`, args: func(uri string) []string { return []string{"boot", "bmc", "list", "--uri", uri, "--token", "t"} }},
		{name: "boot config get", body: `{}`, args: func(uri string) []string {
			return []string{"boot", "config", "get", "id", "--uri", uri, "--token", "t"}
		}},
		{name: "boot config list", body: `[]`, args: func(uri string) []string { return []string{"boot", "config", "list", "--uri", uri, "--token", "t"} }},
		{name: "boot node get", body: `{}`, args: func(uri string) []string { return []string{"boot", "node", "get", "id", "--uri", uri, "--token", "t"} }},
		{name: "boot node list", body: `[]`, args: func(uri string) []string { return []string{"boot", "node", "list", "--uri", uri, "--token", "t"} }},
		{name: "bss dumpstate", body: `{}`, args: func(uri string) []string { return []string{"bss", "dumpstate", "--uri", uri} }},
		{name: "bss history", body: `[]`, args: func(uri string) []string { return []string{"bss", "history", "--uri", uri} }},
		{name: "bss hosts", body: `[]`, args: func(uri string) []string { return []string{"bss", "hosts", "get", "--uri", uri} }},
		{name: "bss script", body: `#!ipxe`, args: func(uri string) []string {
			return []string{"bss", "boot", "script", "get", "--uri", uri, "--mac", "de:ad:be:ef:00:00"}
		}},
		{name: "bss service version", body: `{"version":"1"}`, args: func(uri string) []string { return []string{"bss", "service", "version", "--uri", uri} }},
		{name: "cloud-init defaults", body: `{}`, args: func(uri string) []string { return []string{"cloud-init", "defaults", "get", "--uri", uri} }},
		{name: "cloud-init group metadata", body: `{"compute":{"name":"compute","meta-data":{"role":"worker"}}}`, args: func(uri string) []string { return []string{"cloud-init", "group", "get", "meta-data", "--uri", uri} }},
		{name: "cloud-init group raw", body: `{"compute":{"name":"compute"}}`, args: func(uri string) []string { return []string{"cloud-init", "group", "get", "raw", "--uri", uri} }},
		{name: "cloud-init node metadata", body: `hostname: node01`, args: func(uri string) []string {
			return []string{"cloud-init", "node", "get", "meta-data", "--uri", uri, "node01"}
		}},
		{name: "cloud-init node user data", body: `#cloud-config`, args: func(uri string) []string {
			return []string{"cloud-init", "node", "get", "user-data", "--uri", uri, "node01"}
		}},
		{name: "cloud-init node vendor data", body: `#cloud-config`, args: func(uri string) []string {
			return []string{"cloud-init", "node", "get", "vendor-data", "--uri", uri, "node01"}
		}},
		{name: "cloud-init service status", body: `{"version":"1"}`, args: func(uri string) []string { return []string{"cloud-init", "service", "status", "--uri", uri} }},
		{name: "cloud-init service api", body: `{"openapi":"3.0.0"}`, args: func(uri string) []string { return []string{"cloud-init", "service", "status", "--api", "--uri", uri} }},
		{name: "cloud-init service version", body: `{"version":"1"}`, args: func(uri string) []string { return []string{"cloud-init", "service", "version", "--uri", uri} }},
		{name: "metadata defaults get", body: `{}`, args: func(uri string) []string {
			return []string{"metadata", "defaults", "get", "id", "--uri", uri, "--token", "t"}
		}},
		{name: "metadata defaults list", body: `[]`, args: func(uri string) []string {
			return []string{"metadata", "defaults", "list", "--uri", uri, "--token", "t"}
		}},
		{name: "metadata group get", body: `{}`, args: func(uri string) []string {
			return []string{"metadata", "group", "get", "id", "--uri", uri, "--token", "t"}
		}},
		{name: "metadata group list", body: `[]`, args: func(uri string) []string { return []string{"metadata", "group", "list", "--uri", uri, "--token", "t"} }},
		{name: "metadata instance get", body: `{}`, args: func(uri string) []string {
			return []string{"metadata", "instance", "get", "id", "--uri", uri, "--token", "t"}
		}},
		{name: "metadata instance list", body: `[]`, args: func(uri string) []string {
			return []string{"metadata", "instance", "list", "--uri", uri, "--token", "t"}
		}},
		{name: "metadata peer get", body: `{}`, args: func(uri string) []string {
			return []string{"metadata", "peer", "get", "id", "--uri", uri, "--token", "t"}
		}},
		{name: "metadata peer list", body: `[]`, args: func(uri string) []string { return []string{"metadata", "peer", "list", "--uri", uri, "--token", "t"} }},
		{name: "pcs status list", body: `{"status":[]}`, args: func(uri string) []string { return []string{"pcs", "status", "list", "--uri", uri} }},
		{name: "pcs status show", body: `{"status":[{"xname":"node","powerState":"on"}]}`, args: func(uri string) []string { return []string{"pcs", "status", "show", "node", "--uri", uri} }},
		{name: "pcs transition list", body: `{"transitions":[]}`, args: func(uri string) []string { return []string{"pcs", "transition", "list", "--uri", uri} }},
		{name: "pcs transition show", body: `{}`, args: func(uri string) []string { return []string{"pcs", "transition", "show", "id", "--uri", uri} }},
		{name: "pcs transition start", body: `{"TransitionID":"id","Operation":"on"}`, args: func(uri string) []string {
			return []string{"pcs", "transition", "start", "--uri", uri, "--token", "t", "--xname", "x0c0s0b0n0", "on"}
		}},
		{name: "smd component get", body: `{"Components":[]}`, args: func(uri string) []string { return []string{"smd", "component", "get", "--uri", uri} }},
		{name: "smd group get", body: `[]`, args: func(uri string) []string { return []string{"smd", "group", "get", "--uri", uri} }},
		{name: "smd group membership", body: `{}`, args: func(uri string) []string { return []string{"smd", "group", "membership", "--uri", uri} }},
		{name: "smd iface get", body: `[]`, args: func(uri string) []string { return []string{"smd", "iface", "get", "--uri", uri} }},
		{name: "smd rfe get", body: `[]`, args: func(uri string) []string { return []string{"smd", "rfe", "get", "--uri", uri} }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				io.WriteString(w, tc.body)
			}))
			defer srv.Close()

			args := append([]string{"--ignore-config"}, tc.args(srv.URL)...)
			res := runOchamiWithOutputWriter(t, commandErrorWriter{}, args...)
			if res.err == nil {
				t.Fatal("expected output error, got nil")
			}
			if res.exitCode != cli.CodePayload {
				t.Errorf("exit code = %d, want %d (%s): %v", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload), res.err)
			}
			if !strings.Contains(res.err.Error(), "injected command output failure") {
				t.Errorf("error = %q, want injected writer failure", res.err)
			}
		})
	}
}

// TestAuthenticatedCommands_RejectMissingTokenBeforeRequest verifies that each
// listed command that sends a token, run against a cluster with enable-auth set
// and no token available, fails with CodeAuth without sending a request.
func TestAuthenticatedCommands_RejectMissingTokenBeforeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		new  func() *cobra.Command
		args []string
	}{
		{name: "boot bmc add", new: bootcmd.NewCmd, args: []string{"bmc", "add", "-d", `{}`}},
		{name: "boot bmc delete", new: bootcmd.NewCmd, args: []string{"bmc", "delete", "--no-confirm", "id"}},
		{name: "boot bmc get", new: bootcmd.NewCmd, args: []string{"bmc", "get", "id"}},
		{name: "boot bmc list", new: bootcmd.NewCmd, args: []string{"bmc", "list"}},
		{name: "boot bmc patch", new: bootcmd.NewCmd, args: []string{"bmc", "patch", "id", "-d", `{}`}},
		{name: "boot bmc set", new: bootcmd.NewCmd, args: []string{"bmc", "set", "id", "-d", `{}`}},
		{name: "boot config add", new: bootcmd.NewCmd, args: []string{"config", "add", "-d", `{}`}},
		{name: "boot config delete", new: bootcmd.NewCmd, args: []string{"config", "delete", "--no-confirm", "id"}},
		{name: "boot config get", new: bootcmd.NewCmd, args: []string{"config", "get", "id"}},
		{name: "boot config list", new: bootcmd.NewCmd, args: []string{"config", "list"}},
		{name: "boot config patch", new: bootcmd.NewCmd, args: []string{"config", "patch", "id", "-d", `{}`}},
		{name: "boot config set", new: bootcmd.NewCmd, args: []string{"config", "set", "id", "-d", `{}`}},
		{name: "boot node add", new: bootcmd.NewCmd, args: []string{"node", "add", "-d", `{}`}},
		{name: "boot node delete", new: bootcmd.NewCmd, args: []string{"node", "delete", "--no-confirm", "id"}},
		{name: "boot node get", new: bootcmd.NewCmd, args: []string{"node", "get", "id"}},
		{name: "boot node list", new: bootcmd.NewCmd, args: []string{"node", "list"}},
		{name: "boot node patch", new: bootcmd.NewCmd, args: []string{"node", "patch", "id", "-d", `{}`}},
		{name: "boot node set", new: bootcmd.NewCmd, args: []string{"node", "set", "id", "-d", `{}`}},

		{name: "bss image set", new: bsscmd.NewCmd, args: []string{"boot", "image", "set", "--mac", "de:ad:be:ef:00:00", "image"}},
		{name: "bss params add", new: bsscmd.NewCmd, args: []string{"boot", "params", "add", "-d", `{}`}},
		{name: "bss params delete", new: bsscmd.NewCmd, args: []string{"boot", "params", "delete", "--no-confirm", "-d", `{}`}},
		{name: "bss params get", new: bsscmd.NewCmd, args: []string{"boot", "params", "get"}},
		{name: "bss params set", new: bsscmd.NewCmd, args: []string{"boot", "params", "set", "-d", `{}`}},
		{name: "bss params update", new: bsscmd.NewCmd, args: []string{"boot", "params", "update", "-d", `{}`}},

		{name: "cloud-init group add", new: cloudinitcmd.NewCmd, args: []string{"group", "add", "-d", `[]`}},
		{name: "cloud-init group delete", new: cloudinitcmd.NewCmd, args: []string{"group", "delete", "--no-confirm", "group"}},
		{name: "cloud-init group config", new: cloudinitcmd.NewCmd, args: []string{"group", "get", "config"}},
		{name: "cloud-init group metadata", new: cloudinitcmd.NewCmd, args: []string{"group", "get", "meta-data"}},
		{name: "cloud-init group raw", new: cloudinitcmd.NewCmd, args: []string{"group", "get", "raw"}},
		{name: "cloud-init group render", new: cloudinitcmd.NewCmd, args: []string{"group", "render", "group", "node"}},
		{name: "cloud-init group set", new: cloudinitcmd.NewCmd, args: []string{"group", "set", "-d", `[]`}},
		{name: "cloud-init node group", new: cloudinitcmd.NewCmd, args: []string{"node", "get", "group", "node", "group"}},
		{name: "cloud-init node metadata", new: cloudinitcmd.NewCmd, args: []string{"node", "get", "meta-data", "node"}},
		{name: "cloud-init node user data", new: cloudinitcmd.NewCmd, args: []string{"node", "get", "user-data", "node"}},
		{name: "cloud-init node vendor data", new: cloudinitcmd.NewCmd, args: []string{"node", "get", "vendor-data", "node"}},
		{name: "cloud-init node set", new: cloudinitcmd.NewCmd, args: []string{"node", "set", "-d", `[]`}},

		{name: "metadata defaults add", new: metadatacmd.NewCmd, args: []string{"defaults", "add", "-d", `{}`}},
		{name: "metadata defaults delete", new: metadatacmd.NewCmd, args: []string{"defaults", "delete", "--no-confirm", "id"}},
		{name: "metadata defaults get", new: metadatacmd.NewCmd, args: []string{"defaults", "get", "id"}},
		{name: "metadata defaults list", new: metadatacmd.NewCmd, args: []string{"defaults", "list"}},
		{name: "metadata defaults patch", new: metadatacmd.NewCmd, args: []string{"defaults", "patch", "id", "-d", `{}`}},
		{name: "metadata defaults set", new: metadatacmd.NewCmd, args: []string{"defaults", "set", "id", "-d", `{}`}},
		{name: "metadata group add", new: metadatacmd.NewCmd, args: []string{"group", "add", "-d", `{}`}},
		{name: "metadata group delete", new: metadatacmd.NewCmd, args: []string{"group", "delete", "--no-confirm", "id"}},
		{name: "metadata group get", new: metadatacmd.NewCmd, args: []string{"group", "get", "id"}},
		{name: "metadata group list", new: metadatacmd.NewCmd, args: []string{"group", "list"}},
		{name: "metadata group patch", new: metadatacmd.NewCmd, args: []string{"group", "patch", "id", "-d", `{}`}},
		{name: "metadata group set", new: metadatacmd.NewCmd, args: []string{"group", "set", "id", "-d", `{}`}},
		{name: "metadata instance add", new: metadatacmd.NewCmd, args: []string{"instance", "add", "-d", `{}`}},
		{name: "metadata instance delete", new: metadatacmd.NewCmd, args: []string{"instance", "delete", "--no-confirm", "id"}},
		{name: "metadata instance get", new: metadatacmd.NewCmd, args: []string{"instance", "get", "id"}},
		{name: "metadata instance list", new: metadatacmd.NewCmd, args: []string{"instance", "list"}},
		{name: "metadata instance patch", new: metadatacmd.NewCmd, args: []string{"instance", "patch", "id", "-d", `{}`}},
		{name: "metadata instance set", new: metadatacmd.NewCmd, args: []string{"instance", "set", "id", "-d", `{}`}},
		{name: "metadata peer add", new: metadatacmd.NewCmd, args: []string{"peer", "add", "-d", `{}`}},
		{name: "metadata peer delete", new: metadatacmd.NewCmd, args: []string{"peer", "delete", "--no-confirm", "id"}},
		{name: "metadata peer get", new: metadatacmd.NewCmd, args: []string{"peer", "get", "id"}},
		{name: "metadata peer list", new: metadatacmd.NewCmd, args: []string{"peer", "list"}},
		{name: "metadata peer patch", new: metadatacmd.NewCmd, args: []string{"peer", "patch", "id", "-d", `{}`}},
		{name: "metadata peer set", new: metadatacmd.NewCmd, args: []string{"peer", "set", "id", "-d", `{}`}},

		{name: "smd compep get", new: smdcmd.NewCmd, args: []string{"compep", "get"}},
		{name: "smd component add", new: smdcmd.NewCmd, args: []string{"component", "add", "node", "1"}},
		{name: "smd group add", new: smdcmd.NewCmd, args: []string{"group", "add", "group"}},
		{name: "smd group get", new: smdcmd.NewCmd, args: []string{"group", "get"}},
		{name: "smd group membership", new: smdcmd.NewCmd, args: []string{"group", "membership"}},
		{name: "smd iface add", new: smdcmd.NewCmd, args: []string{"iface", "add", "node", "de:ad:be:ef:00:00", "NMN,172.16.0.1"}},
		{name: "smd rfe add", new: smdcmd.NewCmd, args: []string{"rfe", "add", "bmc", "bmc.example", "172.16.0.1", "de:ad:be:ef:00:00"}},
		{name: "smd rfe get", new: smdcmd.NewCmd, args: []string{"rfe", "get"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requestMade := false
			srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				requestMade = true
			}))
			defer srv.Close()

			var output bytes.Buffer
			rt := cli.NewTestRuntime(strings.NewReader(""), &output, &output)
			rt.WithConfig(config.Config{
				DefaultCluster: "secure",
				Clusters: []config.Cluster{{
					Name: "secure",
					Cluster: config.ClusterConfig{
						URI:        srv.URL,
						EnableAuth: true,
					},
				}},
			})

			cmd := tc.new()
			if cmd.PersistentFlags().Lookup("cluster") == nil {
				cmd.PersistentFlags().String("cluster", "", "test cluster")
			}
			if cmd.PersistentFlags().Lookup("cluster-uri") == nil {
				cmd.PersistentFlags().String("cluster-uri", "", "test cluster URI")
			}
			cmd.SetContext(cli.ContextWithRuntime(t.Context(), rt))
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs(append(tc.args, "--cluster", "secure", "--uri", srv.URL))

			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected missing-token error, got nil")
			}
			if cli.ExitCode(err) != cli.CodeAuth {
				t.Errorf("exit code = %d, want %d (%s): %v", cli.ExitCode(err), cli.CodeAuth, cli.CodeName(cli.CodeAuth), err)
			}
			if requestMade {
				t.Error("service received a request despite missing authentication")
			}
		})
	}
}

// TestLeafCommands_RequireInvocationRuntime verifies that each listed command,
// built and run outside the root command without a runtime on its context,
// fails with CodeConfig instead of panicking.
func TestLeafCommands_RequireInvocationRuntime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		new  func() *cobra.Command
		args []string
	}{
		{name: "boot bmc add", new: bootcmd.NewCmd, args: []string{"bmc", "add", "-d", `{}`}},
		{name: "boot bmc delete", new: bootcmd.NewCmd, args: []string{"bmc", "delete", "--no-confirm", "id"}},
		{name: "boot bmc get", new: bootcmd.NewCmd, args: []string{"bmc", "get", "id"}},
		{name: "boot bmc list", new: bootcmd.NewCmd, args: []string{"bmc", "list"}},
		{name: "boot bmc patch", new: bootcmd.NewCmd, args: []string{"bmc", "patch", "id", "-d", `{}`}},
		{name: "boot bmc set", new: bootcmd.NewCmd, args: []string{"bmc", "set", "id", "-d", `{}`}},
		{name: "boot config add", new: bootcmd.NewCmd, args: []string{"config", "add", "-d", `{}`}},
		{name: "boot config delete", new: bootcmd.NewCmd, args: []string{"config", "delete", "--no-confirm", "id"}},
		{name: "boot config get", new: bootcmd.NewCmd, args: []string{"config", "get", "id"}},
		{name: "boot config list", new: bootcmd.NewCmd, args: []string{"config", "list"}},
		{name: "boot config patch", new: bootcmd.NewCmd, args: []string{"config", "patch", "id", "-d", `{}`}},
		{name: "boot config set", new: bootcmd.NewCmd, args: []string{"config", "set", "id", "-d", `{}`}},
		{name: "boot node add", new: bootcmd.NewCmd, args: []string{"node", "add", "-d", `{}`}},
		{name: "boot node delete", new: bootcmd.NewCmd, args: []string{"node", "delete", "--no-confirm", "id"}},
		{name: "boot node get", new: bootcmd.NewCmd, args: []string{"node", "get", "id"}},
		{name: "boot node list", new: bootcmd.NewCmd, args: []string{"node", "list"}},
		{name: "boot node patch", new: bootcmd.NewCmd, args: []string{"node", "patch", "id", "-d", `{}`}},
		{name: "boot node set", new: bootcmd.NewCmd, args: []string{"node", "set", "id", "-d", `{}`}},
		{name: "boot service status", new: bootcmd.NewCmd, args: []string{"service", "status"}},

		{name: "bss image set", new: bsscmd.NewCmd, args: []string{"boot", "image", "set", "--mac", "de:ad:be:ef:00:00", "image"}},
		{name: "bss params add", new: bsscmd.NewCmd, args: []string{"boot", "params", "add", "-d", `{}`}},
		{name: "bss params delete", new: bsscmd.NewCmd, args: []string{"boot", "params", "delete", "--no-confirm", "-d", `{}`}},
		{name: "bss params get", new: bsscmd.NewCmd, args: []string{"boot", "params", "get"}},
		{name: "bss params set", new: bsscmd.NewCmd, args: []string{"boot", "params", "set", "-d", `{}`}},
		{name: "bss params update", new: bsscmd.NewCmd, args: []string{"boot", "params", "update", "-d", `{}`}},
		{name: "bss script get", new: bsscmd.NewCmd, args: []string{"boot", "script", "get", "--mac", "de:ad:be:ef:00:00"}},
		{name: "bss dumpstate", new: bsscmd.NewCmd, args: []string{"dumpstate"}},
		{name: "bss history", new: bsscmd.NewCmd, args: []string{"history"}},
		{name: "bss hosts", new: bsscmd.NewCmd, args: []string{"hosts", "get"}},
		{name: "bss service status", new: bsscmd.NewCmd, args: []string{"service", "status"}},
		{name: "bss service version", new: bsscmd.NewCmd, args: []string{"service", "version"}},
		{name: "bss deprecated status", new: bsscmd.NewCmd, args: []string{"status"}},

		{name: "cloud-init defaults get", new: cloudinitcmd.NewCmd, args: []string{"defaults", "get"}},
		{name: "cloud-init defaults set", new: cloudinitcmd.NewCmd, args: []string{"defaults", "set", "-d", `{}`}},
		{name: "cloud-init group add", new: cloudinitcmd.NewCmd, args: []string{"group", "add", "-d", `[]`}},
		{name: "cloud-init group delete", new: cloudinitcmd.NewCmd, args: []string{"group", "delete", "--no-confirm", "group"}},
		{name: "cloud-init group config", new: cloudinitcmd.NewCmd, args: []string{"group", "get", "config"}},
		{name: "cloud-init group metadata", new: cloudinitcmd.NewCmd, args: []string{"group", "get", "meta-data"}},
		{name: "cloud-init group raw", new: cloudinitcmd.NewCmd, args: []string{"group", "get", "raw"}},
		{name: "cloud-init group render", new: cloudinitcmd.NewCmd, args: []string{"group", "render", "group", "node"}},
		{name: "cloud-init group set", new: cloudinitcmd.NewCmd, args: []string{"group", "set", "-d", `[]`}},
		{name: "cloud-init node group", new: cloudinitcmd.NewCmd, args: []string{"node", "get", "group", "node", "group"}},
		{name: "cloud-init node metadata", new: cloudinitcmd.NewCmd, args: []string{"node", "get", "meta-data", "node"}},
		{name: "cloud-init node user data", new: cloudinitcmd.NewCmd, args: []string{"node", "get", "user-data", "node"}},
		{name: "cloud-init node vendor data", new: cloudinitcmd.NewCmd, args: []string{"node", "get", "vendor-data", "node"}},
		{name: "cloud-init node set", new: cloudinitcmd.NewCmd, args: []string{"node", "set", "-d", `[]`}},
		{name: "cloud-init service status", new: cloudinitcmd.NewCmd, args: []string{"service", "status"}},
		{name: "cloud-init service version", new: cloudinitcmd.NewCmd, args: []string{"service", "version"}},

		{name: "config set", new: configcmd.NewCmd, args: []string{"set", "log.level", "debug"}},
		{name: "config show", new: configcmd.NewCmd, args: []string{"show"}},
		{name: "config unset", new: configcmd.NewCmd, args: []string{"unset", "log.level"}},
		{name: "config cluster delete", new: configcmd.NewCmd, args: []string{"cluster", "delete", "cluster"}},
		{name: "config cluster set", new: configcmd.NewCmd, args: []string{"cluster", "set", "cluster", "cluster.uri", "https://example.com"}},
		{name: "config cluster show", new: configcmd.NewCmd, args: []string{"cluster", "show", "cluster"}},
		{name: "config cluster unset", new: configcmd.NewCmd, args: []string{"cluster", "unset", "cluster", "cluster.uri"}},

		{name: "metadata defaults add", new: metadatacmd.NewCmd, args: []string{"defaults", "add", "-d", `{}`}},
		{name: "metadata defaults delete", new: metadatacmd.NewCmd, args: []string{"defaults", "delete", "--no-confirm", "id"}},
		{name: "metadata defaults get", new: metadatacmd.NewCmd, args: []string{"defaults", "get", "id"}},
		{name: "metadata defaults list", new: metadatacmd.NewCmd, args: []string{"defaults", "list"}},
		{name: "metadata defaults patch", new: metadatacmd.NewCmd, args: []string{"defaults", "patch", "id", "-d", `{}`}},
		{name: "metadata defaults set", new: metadatacmd.NewCmd, args: []string{"defaults", "set", "id", "-d", `{}`}},
		{name: "metadata group add", new: metadatacmd.NewCmd, args: []string{"group", "add", "-d", `{}`}},
		{name: "metadata group delete", new: metadatacmd.NewCmd, args: []string{"group", "delete", "--no-confirm", "id"}},
		{name: "metadata group get", new: metadatacmd.NewCmd, args: []string{"group", "get", "id"}},
		{name: "metadata group list", new: metadatacmd.NewCmd, args: []string{"group", "list"}},
		{name: "metadata group patch", new: metadatacmd.NewCmd, args: []string{"group", "patch", "id", "-d", `{}`}},
		{name: "metadata group set", new: metadatacmd.NewCmd, args: []string{"group", "set", "id", "-d", `{}`}},
		{name: "metadata instance add", new: metadatacmd.NewCmd, args: []string{"instance", "add", "-d", `{}`}},
		{name: "metadata instance delete", new: metadatacmd.NewCmd, args: []string{"instance", "delete", "--no-confirm", "id"}},
		{name: "metadata instance get", new: metadatacmd.NewCmd, args: []string{"instance", "get", "id"}},
		{name: "metadata instance list", new: metadatacmd.NewCmd, args: []string{"instance", "list"}},
		{name: "metadata instance patch", new: metadatacmd.NewCmd, args: []string{"instance", "patch", "id", "-d", `{}`}},
		{name: "metadata instance set", new: metadatacmd.NewCmd, args: []string{"instance", "set", "id", "-d", `{}`}},
		{name: "metadata peer add", new: metadatacmd.NewCmd, args: []string{"peer", "add", "-d", `{}`}},
		{name: "metadata peer delete", new: metadatacmd.NewCmd, args: []string{"peer", "delete", "--no-confirm", "id"}},
		{name: "metadata peer get", new: metadatacmd.NewCmd, args: []string{"peer", "get", "id"}},
		{name: "metadata peer list", new: metadatacmd.NewCmd, args: []string{"peer", "list"}},
		{name: "metadata peer patch", new: metadatacmd.NewCmd, args: []string{"peer", "patch", "id", "-d", `{}`}},
		{name: "metadata peer set", new: metadatacmd.NewCmd, args: []string{"peer", "set", "id", "-d", `{}`}},
		{name: "metadata service status", new: metadatacmd.NewCmd, args: []string{"service", "status"}},

		{name: "pcs service status", new: pcscmd.NewCmd, args: []string{"service", "status"}},
		{name: "pcs status list", new: pcscmd.NewCmd, args: []string{"status", "list"}},
		{name: "pcs status show", new: pcscmd.NewCmd, args: []string{"status", "show", "node"}},
		{name: "pcs transition abort", new: pcscmd.NewCmd, args: []string{"transition", "abort", "id"}},
		{name: "pcs transition list", new: pcscmd.NewCmd, args: []string{"transition", "list"}},
		{name: "pcs transition monitor", new: pcscmd.NewCmd, args: []string{"transition", "monitor", "id"}},
		{name: "pcs transition show", new: pcscmd.NewCmd, args: []string{"transition", "show", "id"}},
		{name: "pcs transition start", new: pcscmd.NewCmd, args: []string{"transition", "start", "--xname", "node", "on"}},

		{name: "rcs console list", new: rcscmd.NewCmd, args: []string{"console", "list"}},
		{name: "rcs console show", new: rcscmd.NewCmd, args: []string{"console", "show", "node"}},
		{name: "rcs service status", new: rcscmd.NewCmd, args: []string{"service", "status"}},

		{name: "smd compep get", new: smdcmd.NewCmd, args: []string{"compep", "get"}},
		{name: "smd component add", new: smdcmd.NewCmd, args: []string{"component", "add", "node", "1"}},
		{name: "smd component get", new: smdcmd.NewCmd, args: []string{"component", "get"}},
		{name: "smd group add", new: smdcmd.NewCmd, args: []string{"group", "add", "group"}},
		{name: "smd group get", new: smdcmd.NewCmd, args: []string{"group", "get"}},
		{name: "smd group membership", new: smdcmd.NewCmd, args: []string{"group", "membership"}},
		{name: "smd iface add", new: smdcmd.NewCmd, args: []string{"iface", "add", "node", "de:ad:be:ef:00:00", "NMN,172.16.0.1"}},
		{name: "smd iface get", new: smdcmd.NewCmd, args: []string{"iface", "get"}},
		{name: "smd rfe add", new: smdcmd.NewCmd, args: []string{"rfe", "add", "bmc", "bmc.example", "172.16.0.1", "de:ad:be:ef:00:00"}},
		{name: "smd rfe get", new: smdcmd.NewCmd, args: []string{"rfe", "get"}},
		{name: "smd service status", new: smdcmd.NewCmd, args: []string{"service", "status"}},
		{name: "smd deprecated status", new: smdcmd.NewCmd, args: []string{"status"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd := tc.new()
			if cmd.PersistentFlags().Lookup("config") == nil {
				cmd.PersistentFlags().String("config", "", "test config path")
			}
			if cmd.PersistentFlags().Lookup("system") == nil {
				cmd.PersistentFlags().Bool("system", false, "test system config")
			}
			if cmd.PersistentFlags().Lookup("user") == nil {
				cmd.PersistentFlags().Bool("user", false, "test user config")
			}
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs(tc.args)

			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected missing-runtime error, got nil")
			}
			if cli.ExitCode(err) != cli.CodeConfig {
				t.Errorf("exit code = %d, want %d (%s): %v", cli.ExitCode(err), cli.CodeConfig, cli.CodeName(cli.CodeConfig), err)
			}
			if !strings.Contains(err.Error(), "CLI runtime is unavailable") {
				t.Errorf("error = %q, want runtime-unavailable context", err)
			}
		})
	}
}

// TestCommands_TransportFailures verifies that each listed command fails with
// CodeNetwork when its service can't be reached.
func TestCommands_TransportFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		wantCode int
		args     func(string) []string
	}{
		{
			name:     "cloud-init defaults get",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "defaults", "get", "--uri", uri}
			},
		},
		{
			name:     "cloud-init defaults set",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "defaults", "set", "--uri", uri, "-d", `{}`}
			},
		},
		{
			name:     "cloud-init node user data",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "node", "get", "user-data", "--uri", uri, "--token", "t", "x0c0s0b0n0"}
			},
		},
		{
			name:     "cloud-init node vendor data",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "node", "get", "vendor-data", "--uri", uri, "--token", "t", "x0c0s0b0n0"}
			},
		},
		{
			name:     "cloud-init node group",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "node", "get", "group", "--uri", uri, "--token", "t", "x0c0s0b0n0", "compute"}
			},
		},
		{
			name:     "cloud-init service version",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "service", "version", "--uri", uri}
			},
		},
		{
			name:     "cloud-init service api",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "cloud-init", "service", "status", "--api", "--uri", uri}
			},
		},
		{
			name:     "bss dumpstate",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "bss", "dumpstate", "--uri", uri}
			},
		},
		{
			name:     "bss service status",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "bss", "service", "status", "--uri", uri}
			},
		},
		{
			name:     "bss service version",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "bss", "service", "version", "--uri", uri}
			},
		},
		{
			name:     "pcs transition show",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "pcs", "transition", "show", "--uri", uri, "transition-id"}
			},
		},
		{
			name:     "pcs transition abort",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "pcs", "transition", "abort", "--uri", uri, "transition-id"}
			},
		},
		{
			name:     "pcs transition start",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "pcs", "transition", "start", "--uri", uri, "--xname", "x0c0s0b0n0", "on"}
			},
		},
		{
			name:     "pcs status show",
			wantCode: cli.CodeNetwork,
			args: func(uri string) []string {
				return []string{"--ignore-config", "pcs", "status", "show", "--uri", uri, "x0c0s0b0n0"}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := runOchamiWithRuntime(t, tc.args("http://127.0.0.1:1")...)
			if res.err == nil {
				t.Fatal("expected a transport error, got nil")
			}
			if res.exitCode != tc.wantCode {
				t.Errorf("exit code = %d, want %d (%s): %v", res.exitCode, tc.wantCode, cli.CodeName(tc.wantCode), res.err)
			}
		})
	}
}

// TestResourceCommands_RejectMalformedSuccessBodies verifies that every boot
// and metadata get and list command fails with CodePayload when a successful
// response can't be decoded.
func TestResourceCommands_RejectMalformedSuccessBodies(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{`)
	}))
	defer srv.Close()

	for _, resource := range bootResourceTypes {
		for _, verb := range []string{"get", "list"} {
			t.Run("boot/"+resource+"/"+verb, func(t *testing.T) {
				args := []string{"--ignore-config", "boot", resource, verb}
				if verb == "get" {
					args = append(args, "some-uid")
				}
				args = append(args, "--uri", srv.URL, "--token", "t")
				res := runOchamiWithRuntime(t, args...)
				if res.err == nil || res.exitCode != cli.CodePayload {
					t.Fatalf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
				}
				if !strings.Contains(res.err.Error(), "failed to unmarshal response") {
					t.Errorf("error = %q, want malformed-response context", res.err)
				}
			})
		}
	}

	for _, resource := range metadataTypes {
		for _, verb := range []string{"get", "list"} {
			t.Run("metadata/"+resource+"/"+verb, func(t *testing.T) {
				args := []string{"--ignore-config", "metadata", resource, verb}
				if verb == "get" {
					args = append(args, "some-uid")
				}
				args = append(args, "--uri", srv.URL, "--token", "t")
				res := runOchamiWithRuntime(t, args...)
				if res.err == nil || res.exitCode != cli.CodePayload {
					t.Fatalf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
				}
				if !strings.Contains(res.err.Error(), "failed to unmarshal response") {
					t.Errorf("error = %q, want malformed-response context", res.err)
				}
			})
		}
	}
}

// TestRootCommand_RequiresInvocationRuntime verifies that the root command
// fails with CodeConfig when its context carries no runtime.
func TestRootCommand_RequiresInvocationRuntime(t *testing.T) {
	t.Parallel()

	// Create a command without runtime in context
	rootCmd := NewRootCmd()
	// Explicitly set context without runtime
	rootCmd.SetContext(context.Background())
	rootCmd.SetArgs([]string{"version"})

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error when runtime is missing, got nil")
	}

	if exitCode := cli.ExitCode(err); exitCode != cli.CodeConfig {
		t.Errorf("exit code = %d, want %d (%s)", exitCode, cli.CodeConfig, cli.CodeName(cli.CodeConfig))
	}
}
