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

	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	for _, tc := range bootListCases {
		t.Run(tc.name, func(t *testing.T) {
			args := append(tc.args, "--ignore-config", "--uri", srv.URL, "--token", "faketoken")
			res := runOchamiWithRuntime(t, args...)
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

	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "boot", "service", "status", "--uri", srv.URL)
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

	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	for _, typ := range bootResourceTypes {
		t.Run(typ, func(t *testing.T) {
			res := runOchamiWithRuntime(t, "--ignore-config", "boot", typ, "get", "some-uid", "--uri", srv.URL, "--token", "t")
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

	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	for _, typ := range bootResourceTypes {
		t.Run(typ, func(t *testing.T) {
			res := runOchamiWithRuntime(t, "--ignore-config", "boot", typ, "delete", "--no-confirm", "some-uid",
				"--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if res.exitCode != cli.CodeSuccess {
				t.Errorf("exit code = %d, want %d (CodeSuccess)", res.exitCode, cli.CodeSuccess)
			}
		})
	}
}

// bootTypes are the three boot resource types that share a common verb surface.
var bootTypes = []string{"config", "node", "bmc"}

// bootEnvelopePayload returns a minimal envelope-API (metadata+spec) payload for
// the given boot resource type.
func bootEnvelopePayload(typ string) string {
	switch typ {
	case "config":
		return `{"metadata":{"name":"compute-boot"},"spec":{"hosts":["x0c0s0b0n0"],"kernel":"http://s3/vmlinuz"}}`
	case "node":
		return `{"metadata":{"name":"node-1"},"spec":{"xname":"x0c0s0b0n0"}}`
	case "bmc":
		return `{"metadata":{"name":"bmc-1"},"spec":{"xname":"x0c0s0b0"}}`
	default:
		return `{"metadata":{"name":"thing-1"},"spec":{}}`
	}
}

// TestBootList_Formats verifies list output-format variants across boot types.
func TestBootList_Formats(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		for _, f := range []string{"json", "json-pretty", "yaml"} {
			t.Run(typ+"/"+f, func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
				}))
				defer srv.Close()

				res := runOchamiWithRuntime(t, "boot", typ, "list", "--ignore-config", "--uri", srv.URL, "--token", "t", "-F", f)
				if res.err != nil {
					t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
				}
			})
		}
	}
}

// TestBootGet_Formats verifies get output-format variants across boot types.
func TestBootGet_Formats(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		for _, f := range []string{"json", "json-pretty", "yaml"} {
			t.Run(typ+"/"+f, func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
				}))
				defer srv.Close()

				res := runOchamiWithRuntime(t, "boot", typ, "get", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t", "-F", f)
				if res.err != nil {
					t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
				}
			})
		}
	}
}

// TestBootList_NetworkError verifies a closed port resolves to a non-success exit
// code for each boot type's list.
func TestBootList_NetworkError(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			url := srv.URL
			srv.Close()

			res := runOchamiWithRuntime(t, "boot", typ, "list", "--ignore-config", "--uri", url, "--token", "t")
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

// TestBootGet_HTTPError verifies a failing get resolves to a non-success exit
// code across boot types.
func TestBootGet_HTTPErrorAllTypes(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "not found", http.StatusNotFound)
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "boot", typ, "get", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

// TestBootAdd_Envelope verifies the envelope (advanced) API path of "add -e"
// across boot types.
func TestBootAdd_Envelope(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "boot", typ, "add", "-e",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "-d", bootEnvelopePayload(typ))
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootAdd_Stdin verifies add reads payload from stdin when -d is not supplied
// (simple API path) across boot types.
func TestBootAdd_Stdin(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInputAndRuntime(t, bootAddPayload(typ),
				"boot", typ, "add", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootSet_Envelope verifies the envelope API path of "set -e" across boot
// types.
func TestBootSet_Envelope(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "boot", typ, "set", "some-uid", "-e",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "-d", bootEnvelopePayload(typ))
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootSet_HTTPError verifies a failing set resolves to a non-success exit
// code across boot types.
func TestBootSet_HTTPError(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "bad", http.StatusBadRequest)
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "boot", typ, "set", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "-d", bootAddPayload(typ))
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

// TestBootPatch_HTTPError verifies a failing patch resolves to a non-success exit
// code across boot types.
func TestBootPatch_HTTPError(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "bad", http.StatusBadRequest)
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "boot", typ, "patch", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "-d", bootAddPayload(typ))
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

// TestBootDelete_ConfirmYes verifies answering "y" at the confirmation prompt
// proceeds with deletion across boot types.
func TestBootDelete_ConfirmYes(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInputAndRuntime(t, "y\n", "boot", typ, "delete", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootDelete_Abort verifies answering "n" aborts deletion without contacting
// the server across boot types.
func TestBootDelete_Abort(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			var deleted bool
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					deleted = true
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			res := runOchamiWithInputAndRuntime(t, "n\n", "boot", typ, "delete", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error on abort: %v (exit %d)", res.err, res.exitCode)
			}
			if deleted {
				t.Error("server received a DELETE despite user declining confirmation")
			}
		})
	}
}

// TestBootAdd_MalformedPayload verifies malformed inline payload resolves to a
// non-success exit code across boot types.
func TestBootAdd_MalformedPayload(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "boot", typ, "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
				"-d", `not json`)
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodePayload {
				t.Errorf("exit code = %d, want %d (CodePayload)", res.exitCode, cli.CodePayload)
			}
		})
	}
}

// TestBootAdd_MultiItemAggregate verifies a multi-item add against a failing
// server aggregates per-item errors into a non-success exit code.
func TestBootAdd_MultiItemAggregate(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "bad", http.StatusBadRequest)
			}))
			defer srv.Close()

			payload := "[" + bootAddPayload(typ) + "," + bootAddPayload(typ) + "]"
			res := runOchamiWithRuntime(t, "boot", typ, "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
				"-d", payload)
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

// TestBootDelete_HTTPError verifies a failing delete resolves to a non-success
// exit code across boot types (per-item aggregation).
func TestBootDelete_HTTPError(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "not found", http.StatusNotFound)
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "boot", typ, "delete", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "--no-confirm")
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

// TestBootSet_Stdin verifies "set <uid>" reads payload from stdin when -d is not
// supplied across boot types.
func TestBootSet_Stdin(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInputAndRuntime(t, bootAddPayload(typ),
				"boot", typ, "set", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootPatch_Stdin verifies "patch <uid>" reads payload from stdin when -d is
// not supplied across boot types.
func TestBootPatch_Stdin(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInputAndRuntime(t, bootAddPayload(typ),
				"boot", typ, "patch", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootAdd_EnvelopeStdin verifies the envelope API path reads from stdin when
// -d is not supplied across boot types.
func TestBootAdd_EnvelopeStdin(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInputAndRuntime(t, bootEnvelopePayload(typ),
				"boot", typ, "add", "-e", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootSet_EnvelopeStdin verifies the envelope set path reads from stdin when
// -d is not supplied across boot types.
func TestBootSet_EnvelopeStdin(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInputAndRuntime(t, bootEnvelopePayload(typ),
				"boot", typ, "set", "some-uid", "-e", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootPatch_Keyval verifies the key-value patch path (--set/--unset) across
// boot types.
func TestBootPatch_Keyval(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "boot", typ, "patch", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t",
				"--set", "description=new", "--unset", "obsolete")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootPatch_RFC6902 verifies the rfc6902 patch-method path across boot types.
func TestBootPatch_RFC6902(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "boot", typ, "patch", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t",
				"--patch-method", "rfc6902", "-d", `[{"op":"replace","path":"/description","value":"new"}]`)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootPatch_StdinData verifies patch reads from stdin when -d is not given
// across boot types.
func TestBootPatch_StdinData(t *testing.T) {
	t.Parallel()

	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInputAndRuntime(t, `{"description":"new"}`,
				"boot", typ, "patch", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// in boot BMC add command.
func TestBootBmcAdd_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"BMCs":[{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "bmc", "add", "-d", `{}`)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestBootBmcAdd_HTTPError verifies HTTP error handling.
func TestBootBmcAdd_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "bmc", "add", "-d", `{}`)

	if res.err == nil {
		t.Fatal("expected HTTP response error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestBootBmcGet_MalformedResponse verifies handling of malformed responses
// in boot BMC get command.
func TestBootBmcGet_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "bmc", "get", "bmc-id")

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestBootBmcList_HTTPError verifies HTTP error handling.
func TestBootBmcList_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "bmc", "list")

	if res.err == nil {
		t.Fatal("expected HTTP response error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestBootBmcSet_MalformedResponse verifies handling of malformed responses
// in boot BMC set command (simple API).
func TestBootBmcSet_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"BMCs":[{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	args := []string{"--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "bmc", "set", "x0c0s1b0n0", "-d", `{"xname":"test"}`}
	res := runOchamiWithRuntime(t, args...)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestBootBmcSet_EnvelopeMalformedResponse verifies handling of malformed responses
// in boot BMC set command (envelope API).
func TestBootBmcSet_EnvelopeMalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"BMCs":[{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	args := []string{"--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "bmc", "set", "-e", "x0c0s1b0n0", "-d", `{"spec":{"xname":"test"}}`}
	res := runOchamiWithRuntime(t, args...)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestBootBmcPatch_MalformedResponse verifies handling of malformed responses
// in boot BMC patch command.
func TestBootBmcPatch_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"BMCs":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "bmc", "patch", "x0c0s1b0n0", "-d", `{}`)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestBootConfigGet_MalformedResponse verifies handling of malformed responses
// in boot config get command.
func TestBootConfigGet_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"BootConfigParams":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "config", "get", "x0c0s1b0n0")

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestBootConfigList_HTTPError verifies HTTP error handling.
func TestBootConfigList_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "config", "list")

	if res.err == nil {
		t.Fatal("expected HTTP response error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestBootNodeGet_MalformedResponse verifies handling of malformed responses
// in boot node get command.
func TestBootNodeGet_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"Nodes":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "node", "get", "node-id")

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestBootNodeList_HTTPError verifies HTTP error handling.
func TestBootNodeList_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	args := []string{"--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "node", "list"}
	res := runOchamiWithRuntime(t, args...)

	if res.err == nil {
		t.Fatal("expected HTTP response error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestBootNodeAdd_MalformedResponse verifies handling of malformed responses
// in boot node add command.
func TestBootNodeAdd_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"Nodes":[{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	args := []string{"--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "node", "add", "-d", `{}`}
	res := runOchamiWithRuntime(t, args...)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestBootNodePatch_MalformedResponse verifies handling of malformed responses
// in boot node patch command.
func TestBootNodePatch_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"Nodes":[{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	args := []string{"--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "node", "patch", "x0c0s1b0n0", "-d", `{}`}
	res := runOchamiWithRuntime(t, args...)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestBootNodeSet_MalformedResponse verifies handling of malformed responses
// in boot node set command.
func TestBootNodeSet_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"Nodes":[{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	args := []string{"--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "node", "set", "x0c0s1b0n0", "-d", `{}`}
	res := runOchamiWithRuntime(t, args...)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestBootConfigAdd_MalformedResponse verifies handling of malformed responses
// in boot config add command.
func TestBootConfigAdd_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"BootConfigParams":[{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	args := []string{"--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "config", "add", "-d", `{}`}
	res := runOchamiWithRuntime(t, args...)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestBootConfigPatch_MalformedResponse verifies handling of malformed responses
// in boot config patch command.
func TestBootConfigPatch_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"BootConfigParams":[{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	args := []string{"--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "config", "patch", "x0c0s1b0n0", "-d", `{}`}
	res := runOchamiWithRuntime(t, args...)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestBootConfigSet_MalformedResponse verifies handling of malformed responses
// in boot config set command.
func TestBootConfigSet_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"BootConfigParams":[{"ID":`)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	args := []string{"--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"boot", "config", "set", "x0c0s1b0n0", "-d", `{}`}
	res := runOchamiWithRuntime(t, args...)

	if res.err == nil {
		t.Fatal("expected malformed response error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}
