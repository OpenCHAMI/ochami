// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// boot_test.go provides end-to-end tests for the "boot" command group, whose
// client wraps the upstream boot-service library. Because the library controls
// the exact request paths, these tests assert exit-code behavior for
// success/HTTP-failure cases rather than exact request routing.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// bootListSuccess is a table-driven check that a "list" subcommand exits
// successfully when the service returns an empty JSON array.
func TestBootListSuccess(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"config list", []string{"boot", "config", "list"}},
		{"node list", []string{"boot", "node", "list"}},
		{"bmc list", []string{"boot", "bmc", "list"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			args := append(tc.args, "--ignore-config", "--uri", srv.URL, "--token", "faketoken")
			res := runOchami(t, args...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if res.exitCode != cli.CodeSuccess {
				t.Errorf("exit code = %d, want %d (CodeSuccess)", res.exitCode, cli.CodeSuccess)
			}
		})
	}
}

// TestBootListHTTPError verifies that an unsuccessful HTTP response from the
// boot service resolves to a non-success exit code for the "list" subcommands.
func TestBootListHTTPError(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"config list", []string{"boot", "config", "list"}},
		{"node list", []string{"boot", "node", "list"}},
		{"bmc list", []string{"boot", "bmc", "list"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "boom", http.StatusInternalServerError)
			}))
			defer srv.Close()

			args := append(tc.args, "--ignore-config", "--uri", srv.URL, "--token", "faketoken")
			res := runOchami(t, args...)
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

// TestBootServiceStatus verifies "boot service status" exits successfully when
// the health endpoint responds OK.
func TestBootServiceStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "boot", "service", "status", "--ignore-config", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if res.exitCode != cli.CodeSuccess {
		t.Errorf("exit code = %d, want %d (CodeSuccess)", res.exitCode, cli.CodeSuccess)
	}
}

// TestBootConfigDeleteNoArgs verifies that "boot config delete" with no UID
// arguments is a usage error (MinimumNArgs(1)).
func TestBootConfigDeleteNoArgs(t *testing.T) {
	res := runOchami(t, "boot", "config", "delete", "--ignore-config", "--uri", "http://127.0.0.1:0", "--no-confirm")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestBootGetSuccess verifies "<type> get <uid>" exits successfully for each
// boot-service resource type.
func TestBootGetSuccess(t *testing.T) {
	for _, typ := range []string{"config", "node", "bmc"} {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchami(t, "boot", typ, "get", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if res.exitCode != cli.CodeSuccess {
				t.Errorf("exit code = %d, want %d (CodeSuccess)", res.exitCode, cli.CodeSuccess)
			}
		})
	}
}

// TestBootGetHTTPError verifies that an unsuccessful HTTP response from a
// "<type> get" resolves to a non-success exit code for each resource type.
func TestBootGetHTTPError(t *testing.T) {
	for _, typ := range []string{"config", "node", "bmc"} {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "not found", http.StatusNotFound)
			}))
			defer srv.Close()

			res := runOchami(t, "boot", typ, "get", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

// TestBootDeleteNoConfirm verifies "<type> delete --no-confirm <uid>" exits
// successfully for each boot-service resource type.
func TestBootDeleteNoConfirm(t *testing.T) {
	for _, typ := range []string{"config", "node", "bmc"} {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchami(t, "boot", typ, "delete", "--no-confirm", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if res.exitCode != cli.CodeSuccess {
				t.Errorf("exit code = %d, want %d (CodeSuccess)", res.exitCode, cli.CodeSuccess)
			}
		})
	}
}
