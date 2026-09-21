// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// commands_errors_test.go validates exact exit codes for error arms shared
// across command families: missing base URI (CodeConfig), missing/invalid token
// (CodeAuth/CodePayload), invalid CA cert, network/HTTP transport failures,
// malformed/extra-arg payload handling, and arguments a client rejects before
// sending a request (CodeUsage).

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestGetClient_NoBaseURI verifies that each listed command fails with
// CodeConfig when no base URI is configured for its service.
func TestGetClient_NoBaseURI(t *testing.T) {
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
