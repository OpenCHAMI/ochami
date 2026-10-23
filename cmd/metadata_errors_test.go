// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// metadata_errors_test.go covers the error arms of the "metadata" command
// group; see metadata_test.go (including the okJSONServer helper) for the
// success paths these mirror.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestMetadataList_HTTPError verifies that an unsuccessful HTTP response
// resolves to CodeHTTP for each metadata resource type's "list".
func TestMetadataList_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	for _, typ := range []string{"defaults", "group", "instance", "peer"} {
		t.Run(typ, func(t *testing.T) {
			res := runOchamiWithRuntime(t, "--ignore-config", "metadata", typ, "list", "--uri", srv.URL, "--token", "t")
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodeHTTP {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
			}
		})
	}
}

// TestMetadataGet_HTTPError verifies that an unsuccessful HTTP response from a
// "<type> get" resolves to CodeHTTP for each resource type.
func TestMetadataGet_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	for _, typ := range []string{"defaults", "group", "instance", "peer"} {
		t.Run(typ, func(t *testing.T) {
			res := runOchamiWithRuntime(t, "--ignore-config", "metadata", typ, "get", "some-uid", "--uri", srv.URL, "--token", "t")
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodeHTTP {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
			}
		})
	}
}

// TestMetadataList_NetworkError verifies a closed port resolves to a network
// error (CodeNetwork) for each type's list.
func TestMetadataList_NetworkError(t *testing.T) {
	t.Parallel()

	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

			res := runOchamiWithRuntime(t, "metadata", typ, "list", "--ignore-config", "--uri", url, "--token", "t")
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodeNetwork {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
			}
		})
	}
}

// TestMetadataSet_HTTPError verifies a failing set resolves to CodeHTTP.
func TestMetadataSet_HTTPError(t *testing.T) {
	t.Parallel()

	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "bad", http.StatusBadRequest)
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "metadata", typ, "set", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "-d", addPayloadFor(typ))
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodeHTTP {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
			}
		})
	}
}

// TestMetadataPatch_HTTPError verifies a failing patch resolves to CodeHTTP.
func TestMetadataPatch_HTTPError(t *testing.T) {
	t.Parallel()

	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "bad", http.StatusBadRequest)
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "metadata", typ, "patch", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "-d", addPayloadFor(typ))
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodeHTTP {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
			}
		})
	}
}

// TestMetadataAdd_MalformedPayload verifies malformed inline payload resolves to
// a non-success exit code across metadata types.
func TestMetadataAdd_MalformedPayload(t *testing.T) {
	t.Parallel()

	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "metadata", typ, "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
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

// TestMetadataAdd_MultiItemAggregate verifies a multi-item add against a
// failing server aggregates per-item errors into CodeHTTP.
func TestMetadataAdd_MultiItemAggregate(t *testing.T) {
	t.Parallel()

	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "bad", http.StatusBadRequest)
			}))
			defer srv.Close()

			payload := "[" + addPayloadFor(typ) + "," + addPayloadFor(typ) + "]"
			res := runOchamiWithRuntime(t, "metadata", typ, "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
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

// TestMetadataServiceStatus_HTTPError verifies that "metadata service status"
// fails with CodeHTTP for an HTTP 503 response.
func TestMetadataServiceStatus_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "metadata", "service", "status",
		"--uri", srv.URL, "--token", "t")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestMetadataAdd_HTTPError verifies that "metadata <type> add" fails with
// CodeHTTP for an unsuccessful HTTP response, for every metadata resource type.
func TestMetadataAdd_HTTPError(t *testing.T) {
	t.Parallel()

	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "bad request", http.StatusBadRequest)
			}))
			defer srv.Close()

			t.Parallel()
			res := runOchamiWithRuntime(t, "metadata", "--ignore-config", typ, "add",
				"--uri", srv.URL, "--token", "t",
				"-d", addPayloadFor(typ))
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodeHTTP {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
			}
		})
	}
}

// TestMetadataDelete_AbortsOnNo verifies that answering "n" at the confirmation
// prompt aborts deletion without contacting the server, for every metadata
// resource type.
func TestMetadataDelete_AbortsOnNo(t *testing.T) {
	t.Parallel()

	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			var contacted bool
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Only record calls to the delete verb (the client may issue a
				// preliminary request); any DELETE means confirmation failed to abort.
				if r.Method == http.MethodDelete {
					contacted = true
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			res := runOchamiWithInputAndRuntime(t, "n\n", "--ignore-config", "metadata", typ, "delete", "some-uid",
				"--uri", srv.URL, "--token", "t")
			if res.exitCode != cli.CodeDeclined {
				t.Fatalf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodeDeclined, cli.CodeName(cli.CodeDeclined))
			}
			if !strings.Contains(res.stdout, "Really delete?") {
				t.Errorf("output = %q, want the confirmation prompt", res.stdout)
			}
			if contacted {
				t.Error("server received a DELETE despite user declining confirmation")
			}
		})
	}
}

// TestMetadataDelete_HTTPError verifies that an unsuccessful HTTP response to
// a delete fails the command for every metadata resource type.
func TestMetadataDelete_HTTPError(t *testing.T) {
	t.Parallel()

	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "not found", http.StatusNotFound)
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "metadata", "--ignore-config", typ, "delete", "some-uid",
				"--uri", srv.URL, "--token", "t", "--no-confirm")
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodeHTTP {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
			}
		})
	}
}
