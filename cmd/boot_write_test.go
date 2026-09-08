// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// boot_write_test.go extends the "boot" command-group coverage to the write
// verbs (add, set, patch). As with boot_test.go, the boot-service client wraps
// an upstream library that controls exact routing, so these tests assert
// exit-code behavior for success and HTTP-failure rather than exact paths.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// bootAddPayload returns a minimal add payload for each boot resource type.
func bootAddPayload(typ string) string {
	switch typ {
	case "config":
		return `{"name":"compute-boot","hosts":["x0c0s0b0n0"],"kernel":"http://s3/vmlinuz"}`
	case "node":
		return `{"name":"node-1","xname":"x0c0s0b0n0"}`
	case "bmc":
		return `{"name":"bmc-1","xname":"x0c0s0b0"}`
	default:
		return `{"name":"thing-1"}`
	}
}

func TestBootAddSuccess(t *testing.T) {
	for _, typ := range []string{"config", "node", "bmc"} {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchami(t, "boot", typ, "add",
				"--ignore-config", "--uri", srv.URL, "--token", "faketoken",
				"-d", bootAddPayload(typ))
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if res.exitCode != cli.CodeSuccess {
				t.Errorf("exit code = %d, want %d (CodeSuccess)", res.exitCode, cli.CodeSuccess)
			}
		})
	}
}

func TestBootAddHTTPError(t *testing.T) {
	for _, typ := range []string{"config", "node", "bmc"} {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "bad request", http.StatusBadRequest)
			}))
			defer srv.Close()

			res := runOchami(t, "boot", typ, "add",
				"--ignore-config", "--uri", srv.URL, "--token", "faketoken",
				"-d", bootAddPayload(typ))
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

func TestBootSetSuccess(t *testing.T) {
	for _, typ := range []string{"config", "node", "bmc"} {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchami(t, "boot", typ, "set", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "faketoken",
				"-d", bootAddPayload(typ))
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if res.exitCode != cli.CodeSuccess {
				t.Errorf("exit code = %d, want %d (CodeSuccess)", res.exitCode, cli.CodeSuccess)
			}
		})
	}
}

func TestBootPatchSuccess(t *testing.T) {
	for _, typ := range []string{"config", "node", "bmc"} {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchami(t, "boot", typ, "patch", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "faketoken",
				"-d", bootAddPayload(typ))
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if res.exitCode != cli.CodeSuccess {
				t.Errorf("exit code = %d, want %d (CodeSuccess)", res.exitCode, cli.CodeSuccess)
			}
		})
	}
}
