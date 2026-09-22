// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// boot_test.go provides end-to-end tests for the "boot" command group, whose
// client wraps the upstream boot-service library. Because the library controls
// the exact request paths, these tests assert exit-code behavior for
// success/HTTP-failure cases rather than exact request routing. HTTP-failure
// and rejection-path cases are covered in boot_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// bootResourceTypes are the boot-service resource types exposed under
// "boot <type> ...", shared by the success and error-path tests below.
var bootResourceTypes = []string{"config", "node", "bmc"}

// bootListCase is one "boot <type> list" table-test case.
type bootListCase struct {
	name string
	args []string
}

// bootListCases enumerates the "boot <type> list" subcommands, shared by the
// success and error-path tests below.
var bootListCases = func() []bootListCase {
	cases := make([]bootListCase, len(bootResourceTypes))
	for i, typ := range bootResourceTypes {
		cases[i] = bootListCase{typ + " list", []string{"boot", typ, "list"}}
	}
	return cases
}()

// bootListSuccess is a table-driven check that a "list" subcommand exits
// successfully when the service returns an empty JSON array.
func TestBootList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	for _, tc := range bootListCases {
		t.Run(tc.name, func(t *testing.T) {
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

// TestBootServiceStatus verifies "boot service status" exits successfully when
// the health endpoint responds OK.
func TestBootServiceStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
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

// TestBootGet_Success verifies "<type> get <uid>" exits successfully for each
// boot-service resource type.
func TestBootGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	for _, typ := range bootResourceTypes {
		t.Run(typ, func(t *testing.T) {
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

// TestBootDelete_NoConfirm verifies "<type> delete --no-confirm <uid>" exits
// successfully for each boot-service resource type.
func TestBootDelete_NoConfirm(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	for _, typ := range bootResourceTypes {
		t.Run(typ, func(t *testing.T) {
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
