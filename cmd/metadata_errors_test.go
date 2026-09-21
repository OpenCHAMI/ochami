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
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestMetadataList_HTTPError verifies that an unsuccessful HTTP response
// resolves to CodeHTTP for each metadata resource type's "list".
func TestMetadataList_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	for _, typ := range []string{"defaults", "group", "instance", "peer"} {
		t.Run(typ, func(t *testing.T) {
			res := runOchami(t, "metadata", typ, "list", "--ignore-config", "--uri", srv.URL, "--token", "t")
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	for _, typ := range []string{"defaults", "group", "instance", "peer"} {
		t.Run(typ, func(t *testing.T) {
			res := runOchami(t, "metadata", typ, "get", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode != cli.CodeHTTP {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
			}
		})
	}
}
