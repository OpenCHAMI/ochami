// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

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
