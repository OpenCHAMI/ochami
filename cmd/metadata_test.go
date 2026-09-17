// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// metadata_test.go exercises representative "metadata" subcommands across the
// four resource types (defaults, group, instance, peer). The metadata client
// wraps an upstream library, so these tests assert exit-code behavior for
// success/HTTP-failure rather than exact request routing (routing for the
// metadata client is covered by its own package tests).

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// okJSONServer returns a server that responds 200 with an empty JSON body to
// every request, and a client-targetable URL.
func okJSONServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
}

// TestMetadataListSuccess verifies that "<type> list" exits successfully for
// each metadata resource type.
func TestMetadataListSuccess(t *testing.T) {
	for _, typ := range []string{"defaults", "group", "instance", "peer"} {
		t.Run(typ, func(t *testing.T) {
			srv := okJSONServer(t)
			defer srv.Close()

			res := runOchami(t, "metadata", typ, "list", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if res.exitCode != cli.CodeSuccess {
				t.Errorf("exit code = %d, want %d (CodeSuccess)", res.exitCode, cli.CodeSuccess)
			}
		})
	}
}

// TestMetadataListHTTPError verifies that an unsuccessful HTTP response resolves
// to a non-success exit code for each metadata resource type's "list".
func TestMetadataListHTTPError(t *testing.T) {
	for _, typ := range []string{"defaults", "group", "instance", "peer"} {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
			}))
			defer srv.Close()

			res := runOchami(t, "metadata", typ, "list", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

// TestMetadataGetSuccess verifies that "<type> get <uid>" exits successfully for
// each metadata resource type.
func TestMetadataGetSuccess(t *testing.T) {
	for _, typ := range []string{"defaults", "group", "instance", "peer"} {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchami(t, "metadata", typ, "get", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if res.exitCode != cli.CodeSuccess {
				t.Errorf("exit code = %d, want %d (CodeSuccess)", res.exitCode, cli.CodeSuccess)
			}
		})
	}
}

// TestMetadataGetHTTPError verifies that an unsuccessful HTTP response from a
// "<type> get" resolves to a non-success exit code for each resource type.
func TestMetadataGetHTTPError(t *testing.T) {
	for _, typ := range []string{"defaults", "group", "instance", "peer"} {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "not found", http.StatusNotFound)
			}))
			defer srv.Close()

			res := runOchami(t, "metadata", typ, "get", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}
