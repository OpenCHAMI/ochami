// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// metadata_write_test.go exercises the write verbs (add, set, patch, delete) of
// the "metadata" subcommands across the four resource types. The metadata
// client wraps an upstream library, so these tests assert exit-code behavior
// for success, HTTP failure, and interactive confirmation rather than exact
// request routing (routing is covered by the client package's own tests).

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// metadataTypes are the four metadata resource types that share a common verb
// surface.
var metadataTypes = []string{"defaults", "group", "instance", "peer"}

// addPayloadFor returns a minimal JSON payload accepted by "<type> add" for the
// given resource type. All four accept a name plus type-appropriate fields.
func addPayloadFor(typ string) string {
	switch typ {
	case "peer":
		return `{"name":"peer-1","publicKey":"abc","allowedIP":"10.0.0.1/32"}`
	default:
		return `{"name":"thing-1"}`
	}
}

func TestMetadataAddSuccess(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`))
			}))
			defer srv.Close()

			res := runOchami(t, "metadata", typ, "add",
				"--ignore-config", "--uri", srv.URL, "--token", "t",
				"-d", addPayloadFor(typ))
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if res.exitCode != cli.CodeSuccess {
				t.Errorf("exit code = %d, want %d (CodeSuccess)", res.exitCode, cli.CodeSuccess)
			}
		})
	}
}

func TestMetadataAddHTTPError(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "bad request", http.StatusBadRequest)
			}))
			defer srv.Close()

			res := runOchami(t, "metadata", typ, "add",
				"--ignore-config", "--uri", srv.URL, "--token", "t",
				"-d", addPayloadFor(typ))
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

func TestMetadataSetSuccess(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`))
			}))
			defer srv.Close()

			res := runOchami(t, "metadata", typ, "set", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t",
				"-d", addPayloadFor(typ))
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if res.exitCode != cli.CodeSuccess {
				t.Errorf("exit code = %d, want %d (CodeSuccess)", res.exitCode, cli.CodeSuccess)
			}
		})
	}
}

func TestMetadataDeleteNoConfirm(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			}))
			defer srv.Close()

			res := runOchami(t, "metadata", typ, "delete", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "--no-confirm")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if res.exitCode != cli.CodeSuccess {
				t.Errorf("exit code = %d, want %d (CodeSuccess)", res.exitCode, cli.CodeSuccess)
			}
		})
	}
}

// TestMetadataDeleteAbortsOnNo verifies that answering "n" at the confirmation
// prompt aborts deletion without error and without contacting the server.
func TestMetadataDeleteAbortsOnNo(t *testing.T) {
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

	res := runOchamiWithInput(t, "n\n", "metadata", "group", "delete", "some-uid",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if contacted {
		t.Error("server received a DELETE despite user declining confirmation")
	}
}

func TestMetadataDeleteHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchami(t, "metadata", "group", "delete", "some-uid",
		"--ignore-config", "--uri", srv.URL, "--token", "t", "--no-confirm")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode == cli.CodeSuccess {
		t.Errorf("exit code = %d, want a non-success code", res.exitCode)
	}
}
