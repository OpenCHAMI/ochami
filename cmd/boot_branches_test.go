// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// boot_branches_test.go extends the "boot" verb coverage to the branch families
// the happy paths do not reach: output-format variants on list/get, the
// envelope (advanced) API path on add/set, network error mapping, stdin payload
// input, patch/set HTTP errors, and the delete confirm-yes/abort paths. The
// three boot resource types (config, node, bmc) share a common verb surface.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

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

// TestBootListFormats verifies list output-format variants across boot types.
func TestBootListFormats(t *testing.T) {
	for _, typ := range bootTypes {
		for _, f := range []string{"json", "json-pretty", "yaml"} {
			t.Run(typ+"/"+f, func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
				}))
				defer srv.Close()

				res := runOchami(t, "boot", typ, "list", "--ignore-config", "--uri", srv.URL, "--token", "t", "-F", f)
				if res.err != nil {
					t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
				}
			})
		}
	}
}

// TestBootGetFormats verifies get output-format variants across boot types.
func TestBootGetFormats(t *testing.T) {
	for _, typ := range bootTypes {
		for _, f := range []string{"json", "json-pretty", "yaml"} {
			t.Run(typ+"/"+f, func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
				}))
				defer srv.Close()

				res := runOchami(t, "boot", typ, "get", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t", "-F", f)
				if res.err != nil {
					t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
				}
			})
		}
	}
}

// TestBootListNetworkError verifies a closed port resolves to a non-success exit
// code for each boot type's list.
func TestBootListNetworkError(t *testing.T) {
	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			url := srv.URL
			srv.Close()

			res := runOchami(t, "boot", typ, "list", "--ignore-config", "--uri", url, "--token", "t")
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

// TestBootGetHTTPError verifies a failing get resolves to a non-success exit
// code across boot types.
func TestBootGetHTTPErrorAllTypes(t *testing.T) {
	for _, typ := range bootTypes {
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

// TestBootAddEnvelope verifies the envelope (advanced) API path of "add -e"
// across boot types.
func TestBootAddEnvelope(t *testing.T) {
	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchami(t, "boot", typ, "add", "-e",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "-d", bootEnvelopePayload(typ))
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootAddStdin verifies add reads payload from stdin when -d is not supplied
// (simple API path) across boot types.
func TestBootAddStdin(t *testing.T) {
	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInput(t, bootAddPayload(typ),
				"boot", typ, "add", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootSetEnvelope verifies the envelope API path of "set -e" across boot
// types.
func TestBootSetEnvelope(t *testing.T) {
	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchami(t, "boot", typ, "set", "some-uid", "-e",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "-d", bootEnvelopePayload(typ))
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootSetHTTPError verifies a failing set resolves to a non-success exit
// code across boot types.
func TestBootSetHTTPError(t *testing.T) {
	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "bad", http.StatusBadRequest)
			}))
			defer srv.Close()

			res := runOchami(t, "boot", typ, "set", "some-uid",
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

// TestBootPatchHTTPError verifies a failing patch resolves to a non-success exit
// code across boot types.
func TestBootPatchHTTPError(t *testing.T) {
	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "bad", http.StatusBadRequest)
			}))
			defer srv.Close()

			res := runOchami(t, "boot", typ, "patch", "some-uid",
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

// TestBootDeleteConfirmYes verifies answering "y" at the confirmation prompt
// proceeds with deletion across boot types.
func TestBootDeleteConfirmYes(t *testing.T) {
	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInput(t, "y\n", "boot", typ, "delete", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootDeleteAbort verifies answering "n" aborts deletion without contacting
// the server across boot types.
func TestBootDeleteAbort(t *testing.T) {
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

			res := runOchamiWithInput(t, "n\n", "boot", typ, "delete", "some-uid",
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

// TestBootAddMalformedPayload verifies malformed inline payload resolves to a
// non-success exit code across boot types.
func TestBootAddMalformedPayload(t *testing.T) {
	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
			}))
			defer srv.Close()

			res := runOchami(t, "boot", typ, "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
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

// TestBootAddMultiItemAggregate verifies a multi-item add against a failing
// server aggregates per-item errors into a non-success exit code.
func TestBootAddMultiItemAggregate(t *testing.T) {
	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "bad", http.StatusBadRequest)
			}))
			defer srv.Close()

			payload := "[" + bootAddPayload(typ) + "," + bootAddPayload(typ) + "]"
			res := runOchami(t, "boot", typ, "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
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

// TestBootDeleteHTTPError verifies a failing delete resolves to a non-success
// exit code across boot types (per-item aggregation).
func TestBootDeleteHTTPError(t *testing.T) {
	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "not found", http.StatusNotFound)
			}))
			defer srv.Close()

			res := runOchami(t, "boot", typ, "delete", "some-uid",
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

// TestBootSetStdin verifies "set <uid>" reads payload from stdin when -d is not
// supplied across boot types.
func TestBootSetStdin(t *testing.T) {
	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInput(t, bootAddPayload(typ),
				"boot", typ, "set", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootPatchStdin verifies "patch <uid>" reads payload from stdin when -d is
// not supplied across boot types.
func TestBootPatchStdin(t *testing.T) {
	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInput(t, bootAddPayload(typ),
				"boot", typ, "patch", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootAddEnvelopeStdin verifies the envelope API path reads from stdin when
// -d is not supplied across boot types.
func TestBootAddEnvelopeStdin(t *testing.T) {
	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInput(t, bootEnvelopePayload(typ),
				"boot", typ, "add", "-e", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootSetEnvelopeStdin verifies the envelope set path reads from stdin when
// -d is not supplied across boot types.
func TestBootSetEnvelopeStdin(t *testing.T) {
	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInput(t, bootEnvelopePayload(typ),
				"boot", typ, "set", "some-uid", "-e", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootPatchKeyval verifies the key-value patch path (--set/--unset) across
// boot types.
func TestBootPatchKeyval(t *testing.T) {
	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchami(t, "boot", typ, "patch", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t",
				"--set", "description=new", "--unset", "obsolete")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootPatchRFC6902 verifies the rfc6902 patch-method path across boot types.
func TestBootPatchRFC6902(t *testing.T) {
	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchami(t, "boot", typ, "patch", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t",
				"--patch-method", "rfc6902", "-d", `[{"op":"replace","path":"/description","value":"new"}]`)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestBootPatchStdinData verifies patch reads from stdin when -d is not given
// across boot types.
func TestBootPatchStdinData(t *testing.T) {
	for _, typ := range bootTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInput(t, `{"description":"new"}`,
				"boot", typ, "patch", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}
