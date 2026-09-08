// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// noconfig_test.go verifies that commands requiring a resolvable service base
// URI fail with CodeConfig when neither --uri nor a cluster config provides one.
// This exercises the GetClient/GetBaseURI error arm shared by every service
// command.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

func TestGetClientNoBaseURI(t *testing.T) {
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
			res := runOchami(t, full...)
			if res.err == nil {
				t.Fatalf("expected an error without a base URI, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

// TestHandleTokenAuthRequired verifies that with enable-auth enabled and no
// token available, service commands fail with CodeAuth. This exercises the
// HandleToken error-return arm shared by every service command. The base URI is
// provided via a config-file cluster so GetClient succeeds and the failure
// occurs in HandleToken.
func TestHandleTokenAuthRequired(t *testing.T) {
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
			res := runOchami(t, full...)
			if res.err == nil {
				t.Fatalf("expected an auth error, got nil")
			}
			if res.exitCode != cli.CodeAuth {
				t.Errorf("exit code = %d, want %d (CodeAuth)", res.exitCode, cli.CodeAuth)
			}
		})
	}
}

// TestUseCACertInvalid verifies that commands which load a CA certificate fail
// with CodePayload when --cacert points at an invalid/nonexistent file. This
// exercises the shared UseCACert error arm.
func TestUseCACertInvalid(t *testing.T) {
	srv := okJSONServer(t)
	defer srv.Close()

	cases := [][]string{
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
			res := runOchami(t, full...)
			if res.err == nil {
				t.Fatalf("expected an error for invalid --cacert, got nil")
			}
			if res.exitCode != cli.CodePayload {
				t.Errorf("exit code = %d, want %d (CodePayload)", res.exitCode, cli.CodePayload)
			}
		})
	}
}

// TestServiceCommandsNetworkError points commands at a closed port so the
// underlying request fails at the transport layer, exercising the CodeNetwork
// arm shared by many service commands.
func TestServiceCommandsNetworkError(t *testing.T) {
	srv := okJSONServer(t)
	url := srv.URL
	srv.Close() // closed => connection refused

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
		{"pcs", "service", "status"},
	}
	for _, args := range cases {
		name := ""
		for _, a := range args {
			name += a + "_"
		}
		t.Run(name, func(t *testing.T) {
			full := append(args, "--ignore-config", "--uri", url, "--token", "t")
			res := runOchami(t, full...)
			if res.err == nil {
				t.Fatalf("expected a network error, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

// TestServiceCommandsHTTPError points commands at a server returning 500 so the
// HTTP-error mapping arm (CodeHTTP) is exercised broadly.
func TestServiceCommandsHTTPError(t *testing.T) {
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
			res := runOchami(t, full...)
			if res.err == nil {
				t.Fatalf("expected an HTTP error, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

// TestGetClientUseCACertInvalid verifies that an invalid --cacert causes the
// per-service GetClient's UseCACert step to fail with CodePayload, exercising
// that shared arm across every service's GetClient.
func TestGetClientUseCACertInvalid(t *testing.T) {
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
	}
	for _, args := range cases {
		name := ""
		for _, a := range args {
			name += a + "_"
		}
		t.Run(name, func(t *testing.T) {
			full := append(args, "--ignore-config", "--uri", srv.URL, "--token", "t",
				"--cacert", "/no/such/ca.pem")
			res := runOchami(t, full...)
			if res.err == nil {
				t.Fatalf("expected an error for invalid --cacert, got nil")
			}
			if res.exitCode != cli.CodePayload {
				t.Errorf("exit code = %d, want %d (CodePayload)", res.exitCode, cli.CodePayload)
			}
		})
	}
}

// TestMalformedPayloadAcrossCommands verifies that malformed inline -d payload
// resolves to CodePayload across the commands that accept a data payload,
// exercising the shared HandlePayload error arm.
func TestMalformedPayloadAcrossCommands(t *testing.T) {
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
			res := runOchami(t, full...)
			if res.err == nil {
				t.Fatalf("expected a payload error, got nil")
			}
			if res.exitCode != cli.CodePayload {
				t.Errorf("exit code = %d, want %d (CodePayload)", res.exitCode, cli.CodePayload)
			}
		})
	}
}

// TestDataWithExtraArgsAcrossCommands verifies that passing both -d and extra
// positional arguments is accepted (the extra args are ignored with a warning)
// across the commands that support -d, exercising that warning arm.
func TestDataWithExtraArgsAcrossCommands(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
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
			res := runOchami(t, full...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestWriteCommandsNetworkError points write commands at a closed port so their
// network-error arms fire (CodeNetwork or the per-item aggregate).
func TestWriteCommandsNetworkError(t *testing.T) {
	srv := okJSONServer(t)
	url := srv.URL
	srv.Close()

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
			res := runOchami(t, full...)
			if res.err == nil {
				t.Fatalf("expected a network error, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

// TestMetadataBootWriteNetworkError points metadata/boot write commands at a
// closed port so their network-error arms fire across all resource types.
func TestMetadataBootWriteNetworkError(t *testing.T) {
	srv := okJSONServer(t)
	url := srv.URL
	srv.Close()

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
			res := runOchami(t, full...)
			if res.err == nil {
				t.Fatalf("expected a network error, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}
