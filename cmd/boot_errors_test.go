// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// boot_errors_test.go covers the error arms of the "boot" command group; see
// boot_test.go for the success paths these mirror.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestBootList_HTTPError verifies that an unsuccessful HTTP response from the
// boot service resolves to CodeHTTP for the "list" subcommands.
func TestBootList_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	for _, tc := range bootListCases {
		t.Run(tc.name, func(t *testing.T) {
			args := append(tc.args, "--ignore-config", "--uri", srv.URL, "--token", "faketoken")
			res := runOchamiWithRuntime(t, args...)
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodeHTTP {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
			}
		})
	}
}

// TestBootGet_HTTPError verifies that an unsuccessful HTTP response from a
// "<type> get" resolves to CodeHTTP for each resource type.
func TestBootGet_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	for _, typ := range bootResourceTypes {
		t.Run(typ, func(t *testing.T) {
			res := runOchamiWithRuntime(t, "--ignore-config", "boot", typ, "get", "some-uid", "--uri", srv.URL, "--token", "t")
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodeHTTP {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
			}
		})
	}
}

// TestBootConfigDelete_NoArgs verifies that "boot config delete" with no UID
// arguments is a usage error (MinimumNArgs(1)).
func TestBootConfigDelete_NoArgs(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "--ignore-config", "boot", "config", "delete", "--uri", "http://127.0.0.1:0", "--no-confirm")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestBootList_NetworkError verifies a closed port resolves to CodeNetwork for
// each boot type's list.
func TestBootList_NetworkError(t *testing.T) {
	t.Parallel()

	for _, typ := range bootResourceTypes {
		t.Run(typ, func(t *testing.T) {
			url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

			res := runOchamiWithRuntime(t, "boot", typ, "list", "--ignore-config", "--uri", url, "--token", "t")
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodeNetwork {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
			}
		})
	}
}

// TestBootSet_HTTPError verifies a failing set resolves to CodeHTTP across boot
// types.
func TestBootSet_HTTPError(t *testing.T) {
	t.Parallel()

	for _, typ := range bootResourceTypes {
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
			if res.exitCode != cli.CodeHTTP {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
			}
		})
	}
}

// TestBootPatch_HTTPError verifies a failing patch resolves to CodeHTTP across
// boot types.
func TestBootPatch_HTTPError(t *testing.T) {
	t.Parallel()

	for _, typ := range bootResourceTypes {
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
			if res.exitCode != cli.CodeHTTP {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
			}
		})
	}
}

// TestBootDelete_Abort verifies answering "n" aborts deletion without contacting
// the server across boot types.
func TestBootDelete_Abort(t *testing.T) {
	t.Parallel()

	for _, typ := range bootResourceTypes {
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
			if res.exitCode != cli.CodeDeclined {
				t.Fatalf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodeDeclined, cli.CodeName(cli.CodeDeclined))
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

	for _, typ := range bootResourceTypes {
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
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
			}
		})
	}
}

// TestBootAdd_MultiItemAggregate verifies a multi-item add against a failing
// server aggregates per-item errors into CodeHTTP.
func TestBootAdd_MultiItemAggregate(t *testing.T) {
	t.Parallel()

	for _, typ := range bootResourceTypes {
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
			if res.exitCode != cli.CodeHTTP {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
			}
		})
	}
}

// TestBootDelete_HTTPError verifies a failing delete resolves to CodeHTTP
// across boot types (per-item aggregation).
func TestBootDelete_HTTPError(t *testing.T) {
	t.Parallel()

	for _, typ := range bootResourceTypes {
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
			if res.exitCode != cli.CodeHTTP {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
			}
		})
	}
}

// TestBootBmcAdd_MalformedResponse verifies that "boot bmc add" fails with
// CodePayload when the service's success response can't be decoded.
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
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestBootBmcSet_MalformedResponse verifies that "boot bmc set" fails with
// CodePayload when the service's success response can't be decoded.
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
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestBootBmcSet_EnvelopeMalformedResponse verifies that "boot bmc set -e"
// fails with CodePayload when the service's success response can't be decoded.
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
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestBootBmcPatch_MalformedResponse verifies that "boot bmc patch" fails with
// CodePayload when the service's success response can't be decoded.
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
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestBootNodeAdd_MalformedResponse verifies that "boot node add" fails with
// CodePayload when the service's success response can't be decoded.
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
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestBootNodePatch_MalformedResponse verifies that "boot node patch" fails
// with CodePayload when the service's success response can't be decoded.
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
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestBootNodeSet_MalformedResponse verifies that "boot node set" fails with
// CodePayload when the service's success response can't be decoded.
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
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestBootConfigAdd_MalformedResponse verifies that "boot config add" fails
// with CodePayload when the service's success response can't be decoded.
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
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestBootConfigPatch_MalformedResponse verifies that "boot config patch" fails
// with CodePayload when the service's success response can't be decoded.
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
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestBootConfigSet_MalformedResponse verifies that "boot config set" fails
// with CodePayload when the service's success response can't be decoded.
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
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}
