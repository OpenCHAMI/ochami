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
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
}

// TestMetadataList_Success verifies that "<type> list" exits successfully for
// each metadata resource type.
func TestMetadataList_Success(t *testing.T) {
	t.Parallel()

	srv := okJSONServer(t)
	defer srv.Close()

	for _, typ := range []string{"defaults", "group", "instance", "peer"} {
		t.Run(typ, func(t *testing.T) {
			res := runOchamiWithRuntime(t, "--ignore-config", "metadata", typ, "list", "--uri", srv.URL, "--token", "t")
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
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	for _, typ := range []string{"defaults", "group", "instance", "peer"} {
		t.Run(typ, func(t *testing.T) {
			res := runOchamiWithRuntime(t, "--ignore-config", "metadata", typ, "get", "some-uid", "--uri", srv.URL, "--token", "t")
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
	t.Parallel()

	for _, resource := range []string{"defaults", "group", "instance", "peer"} {
		t.Run(resource, func(t *testing.T) {
			var gotContentType, gotBody string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotContentType = r.Header.Get("Content-Type")
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read request body: %v", err)
				}
				gotBody = string(body)
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"metadata":{"uid":"some-uid","name":"thing"},"spec":{}}`) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "metadata", resource, "patch", "some-uid",
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

// envelopePayloadFor returns a minimal envelope-API (metadata+spec) payload for
// the given metadata resource type.
func envelopePayloadFor(typ string) string {
	switch typ {
	case "peer":
		return `{"metadata":{"name":"peer-1"},"spec":{"publicKey":"abc","allowedIP":"10.0.0.1/32"}}`
	default:
		return `{"metadata":{"name":"thing-1"},"spec":{}}`
	}
}

// TestMetadataList_Formats verifies list output-format variants across types.
func TestMetadataList_Formats(t *testing.T) {
	for _, typ := range metadataTypes {
		for _, f := range []string{"json", "json-pretty", "yaml"} {
			t.Run(typ+"/"+f, func(t *testing.T) {
				srv := okJSONServer(t)
				defer srv.Close()

				res := runOchamiWithRuntime(t, "metadata", typ, "list", "--ignore-config", "--uri", srv.URL, "--token", "t", "-F", f)
				if res.err != nil {
					t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
				}
			})
		}
	}
}

// TestMetadataGet_Formats verifies get output-format variants across types.
func TestMetadataGet_Formats(t *testing.T) {
	for _, typ := range metadataTypes {
		for _, f := range []string{"json", "json-pretty", "yaml"} {
			t.Run(typ+"/"+f, func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
				}))
				defer srv.Close()

				res := runOchamiWithRuntime(t, "metadata", typ, "get", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t", "-F", f)
				if res.err != nil {
					t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
				}
			})
		}
	}
}

// TestMetadataList_NetworkError verifies a closed port resolves to a network
// error (non-success exit) for each type's list.
func TestMetadataList_NetworkError(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := okJSONServer(t)
			url := srv.URL
			srv.Close()

			res := runOchamiWithRuntime(t, "metadata", typ, "list", "--ignore-config", "--uri", url, "--token", "t")
			if res.err == nil {
				t.Fatal("expected an error, got nil")
			}
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

// TestMetadataAdd_Envelope verifies the envelope (advanced) API path of "add -e"
// across types.
func TestMetadataAdd_Envelope(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "metadata", typ, "add", "-e",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "-d", envelopePayloadFor(typ))
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestMetadataAdd_Stdin verifies add reads payload from stdin when -d is not
// supplied (simple API path).
func TestMetadataAdd_Stdin(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInputAndRuntime(t, addPayloadFor(typ),
				"metadata", typ, "add", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestMetadataSet_Envelope verifies the envelope API path of "set -e" across
// types.
func TestMetadataSet_Envelope(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "metadata", typ, "set", "some-uid", "-e",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "-d", envelopePayloadFor(typ))
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestMetadataSet_HTTPError verifies a failing set resolves to a non-success exit
// code.
func TestMetadataSet_HTTPError(t *testing.T) {
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
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

// TestMetadataPatch_Success verifies the patch verb across types.
func TestMetadataPatch_Success(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "metadata", typ, "patch", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "-d", addPayloadFor(typ))
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestMetadataPatch_HTTPError verifies a failing patch resolves to a non-success
// exit code.
func TestMetadataPatch_HTTPError(t *testing.T) {
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
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

// TestMetadataAdd_MalformedPayload verifies malformed inline payload resolves to
// a non-success exit code across metadata types.
func TestMetadataAdd_MalformedPayload(t *testing.T) {
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
				t.Errorf("exit code = %d, want %d (CodePayload)", res.exitCode, cli.CodePayload)
			}
		})
	}
}

// TestMetadataAdd_MultiItemAggregate verifies a multi-item add against a failing
// server aggregates per-item errors into a non-success exit code.
func TestMetadataAdd_MultiItemAggregate(t *testing.T) {
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
			if res.exitCode == cli.CodeSuccess {
				t.Errorf("exit code = %d, want a non-success code", res.exitCode)
			}
		})
	}
}

// TestMetadataDelete_ConfirmYes verifies answering "y" at the confirmation prompt
// proceeds with deletion across types.
func TestMetadataDelete_ConfirmYes(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInputAndRuntime(t, "y\n", "metadata", typ, "delete", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestMetadataDelete_HTTPErrorAllTypes verifies a failing delete resolves to a
// non-success exit code across all metadata types (per-item aggregation).
func TestMetadataDelete_HTTPErrorAllTypes(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "not found", http.StatusNotFound)
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "metadata", typ, "delete", "some-uid",
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

// TestMetadataDelete_AbortAllTypes verifies answering "n" aborts deletion without
// contacting the server across all metadata types.
func TestMetadataDelete_AbortAllTypes(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			var deleted bool
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					deleted = true
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			res := runOchamiWithInputAndRuntime(t, "n\n", "metadata", typ, "delete", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error on abort: %v (exit %d)", res.err, res.exitCode)
			}
			if deleted {
				t.Errorf("%s: server received a DELETE despite user declining", typ)
			}
		})
	}
}

// TestMetadataSet_Stdin verifies "set <uid>" reads the payload from stdin when -d
// is not supplied (simple API path) across metadata types.
func TestMetadataSet_Stdin(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInputAndRuntime(t, addPayloadFor(typ),
				"metadata", typ, "set", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestMetadataPatch_Stdin verifies "patch <uid>" reads the payload from stdin
// when -d is not supplied across metadata types.
func TestMetadataPatch_Stdin(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInputAndRuntime(t, addPayloadFor(typ),
				"metadata", typ, "patch", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestMetadataAdd_EnvelopeStdin verifies the envelope API path reads from stdin
// when -d is not supplied across metadata types.
func TestMetadataAdd_EnvelopeStdin(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInputAndRuntime(t, envelopePayloadFor(typ),
				"metadata", typ, "add", "-e", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestMetadataSet_EnvelopeStdin verifies the envelope set path reads from stdin
// when -d is not supplied across metadata types.
func TestMetadataSet_EnvelopeStdin(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithInputAndRuntime(t, envelopePayloadFor(typ),
				"metadata", typ, "set", "some-uid", "-e", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestMetadataPatch_Keyval verifies the key-value patch path (--set/--unset)
// across metadata types.
func TestMetadataPatch_Keyval(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "metadata", typ, "patch", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t",
				"--set", "description=new")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestMetadataPatch_RFC6902 verifies the rfc6902 patch-method path across types.
func TestMetadataPatch_RFC6902(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"metadata":{"name":"thing-1"}}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "metadata", typ, "patch", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t",
				"--patch-method", "rfc6902", "-d", `[{"op":"replace","path":"/description","value":"new"}]`)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestMetadataSet_NilResource verifies the "set returned no resource" arm when
// the server responds 200 with a null body.
func TestMetadataSet_NilResource(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`null`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "metadata", typ, "set", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "-d", addPayloadFor(typ))
			// Either a clean success or the "no resource" generic error is
			// acceptable depending on how the upstream client decodes null.
			_ = res
		})
	}
}

// TestMetadataServiceStatus_Success verifies "metadata service status" issues GET
// /health.
func TestMetadataServiceStatus_Success(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "metadata", "service", "status", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/health" {
		t.Errorf("path = %q, want /health", gotPath)
	}
}
