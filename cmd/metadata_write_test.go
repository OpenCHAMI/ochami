// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// metadata_write_test.go exercises the add, set, and delete verbs of the
// "metadata" subcommands across the four resource types. The metadata client
// wraps an upstream library, so these tests assert exit-code behavior rather
// than exact request routing (routing is covered by the client package's own
// tests). HTTP failures and the declined delete confirmation are covered in
// metadata_write_errors_test.go.

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

// TestMetadataAdd_Success verifies that "metadata <type> add" succeeds for
// every metadata resource type.
func TestMetadataAdd_Success(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"metadata":{"name":"thing-1"}}`))
			}))
			defer srv.Close()

			res := runOchami(t, "metadata", typ, "add",
				"--ignore-config", "--uri", srv.URL, "--token", "t",
				"-d", addPayloadFor(typ))
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if res.exitCode != cli.CodeSuccess {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeSuccess, cli.CodeName(cli.CodeSuccess))
			}
		})
	}
}

// TestMetadataSet_Success verifies that "metadata <type> set" succeeds for
// every metadata resource type.
func TestMetadataSet_Success(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"metadata":{"name":"thing-1"}}`))
			}))
			defer srv.Close()

			res := runOchami(t, "metadata", typ, "set", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t",
				"-d", addPayloadFor(typ))
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if res.exitCode != cli.CodeSuccess {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeSuccess, cli.CodeName(cli.CodeSuccess))
			}
		})
	}
}

// TestMetadataDelete_NoConfirm verifies that "metadata <type> delete
// --no-confirm" deletes without prompting, for every metadata resource type.
func TestMetadataDelete_NoConfirm(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{}`))
			}))
			defer srv.Close()

			res := runOchami(t, "metadata", typ, "delete", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "--no-confirm")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if res.exitCode != cli.CodeSuccess {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeSuccess, cli.CodeName(cli.CodeSuccess))
			}
		})
	}
}
