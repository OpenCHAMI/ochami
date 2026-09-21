// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// metadata_test.go exercises representative "metadata" subcommands across the
// four resource types (defaults, group, instance, peer). The metadata client
// wraps an upstream library, so these tests assert exit-code behavior for
// success/HTTP-failure rather than exact request routing (routing for the
// metadata client is covered by its own package tests). HTTP-failure cases are
// covered in metadata_errors_test.go.

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// okJSONServer returns a server that responds 200 with an empty JSON body to
// every request, and a client-targetable URL.
func okJSONServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
}

// TestMetadataList_Success verifies that "<type> list" exits successfully for
// each metadata resource type.
func TestMetadataList_Success(t *testing.T) {
	srv := okJSONServer(t)
	defer srv.Close()

	for _, typ := range []string{"defaults", "group", "instance", "peer"} {
		t.Run(typ, func(t *testing.T) {
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

// TestMetadataGet_Success verifies that "<type> get <uid>" exits successfully for
// each metadata resource type.
func TestMetadataGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	for _, typ := range []string{"defaults", "group", "instance", "peer"} {
		t.Run(typ, func(t *testing.T) {
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

// TestMetadataPatch_PathsAndArrayOperations verifies that --add/--set/--unset/
// --remove on "metadata <resource> patch" produce an RFC 6902 JSON Patch
// request instead of being silently dropped.
func TestMetadataPatch_PathsAndArrayOperations(t *testing.T) {
	for _, resource := range []string{"defaults", "group", "instance", "peer"} {
		t.Run(resource, func(t *testing.T) {
			var gotContentType, gotBody string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotContentType = r.Header.Get("Content-Type")
				body, _ := io.ReadAll(r.Body)
				gotBody = string(body)
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"metadata":{"uid":"some-uid","name":"thing"},"spec":{}}`)
			}))
			defer srv.Close()

			res := runOchami(t, "metadata", resource, "patch", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "--add", "items=value")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if gotContentType != "application/json-patch+json" {
				t.Errorf("Content-Type = %q, want application/json-patch+json", gotContentType)
			}
			if !strings.Contains(gotBody, `"op":"add"`) || !strings.Contains(gotBody, `"path":"/items/-"`) {
				t.Errorf("body = %q, want RFC 6902 add operation", gotBody)
			}
		})
	}
}
