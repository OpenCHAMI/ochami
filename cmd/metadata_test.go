// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// metadata_test.go exercises representative "metadata" subcommands across the
// four resource types (defaults, group, instance, peer). The metadata client
// wraps an upstream library, so these tests assert exit-code behavior rather
// than exact request routing (routing for the metadata client is covered by its
// own package tests). HTTP-failure cases are covered in
// metadata_errors_test.go.

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
		w.Write([]byte(`[]`))
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
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeSuccess, cli.CodeName(cli.CodeSuccess))
			}
		})
	}
}

// TestMetadataGet_Success verifies that "<type> get <uid>" exits successfully for
// each metadata resource type.
func TestMetadataGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	for _, typ := range []string{"defaults", "group", "instance", "peer"} {
		t.Run(typ, func(t *testing.T) {
			res := runOchami(t, "metadata", typ, "get", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if res.exitCode != cli.CodeSuccess {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeSuccess, cli.CodeName(cli.CodeSuccess))
			}
		})
	}
}

// TestMetadataPatch_PathsAndArrayOperations verifies that --add on "metadata
// <resource> patch" produces an RFC 6902 JSON Patch request instead of being
// silently dropped.
func TestMetadataPatch_PathsAndArrayOperations(t *testing.T) {
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
				io.WriteString(w, `{"metadata":{"uid":"some-uid","name":"thing"},"spec":{}}`)
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
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`[{"metadata":{"name":"thing-1"}}]`))
				}))
				defer srv.Close()

				res := runOchami(t, "metadata", typ, "list", "--ignore-config", "--uri", srv.URL, "--token", "t", "-F", f)
				if res.err != nil {
					t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
				}
				assertFormattedOutput(t, f, res.stdout, "name", "thing-1")
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
					w.Write([]byte(`{"metadata":{"name":"thing-1"}}`))
				}))
				defer srv.Close()

				res := runOchami(t, "metadata", typ, "get", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t", "-F", f)
				if res.err != nil {
					t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
				}
				assertFormattedOutput(t, f, res.stdout, "name", "thing-1")
			})
		}
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
				w.Write([]byte(`{"metadata":{"name":"thing-1"}}`))
			}))
			defer srv.Close()

			res := runOchami(t, "metadata", typ, "add", "-e",
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
				w.Write([]byte(`{"metadata":{"name":"thing-1"}}`))
			}))
			defer srv.Close()

			res := runOchamiWithInput(t, addPayloadFor(typ),
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
				w.Write([]byte(`{"metadata":{"name":"thing-1"}}`))
			}))
			defer srv.Close()

			res := runOchami(t, "metadata", typ, "set", "some-uid", "-e",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "-d", envelopePayloadFor(typ))
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestMetadataPatch_Success verifies that "metadata <type> patch" with a
// payload succeeds for every metadata resource type.
func TestMetadataPatch_Success(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"metadata":{"name":"thing-1"}}`))
			}))
			defer srv.Close()

			res := runOchami(t, "metadata", typ, "patch", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "-d", addPayloadFor(typ))
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestMetadataDelete_ConfirmYes verifies that "metadata <type> delete" prompts
// for confirmation and, on "y", sends the DELETE, for every metadata resource
// type.
func TestMetadataDelete_ConfirmYes(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			var deleted bool
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					deleted = true
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{}`))
			}))
			defer srv.Close()

			res := runOchamiWithInput(t, "y\n", "metadata", typ, "delete", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if !strings.Contains(res.stdout, "Really delete?") {
				t.Errorf("output = %q, want the confirmation prompt", res.stdout)
			}
			if !deleted {
				t.Error("server received no DELETE after the user confirmed")
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
				w.Write([]byte(`{"metadata":{"name":"thing-1"}}`))
			}))
			defer srv.Close()

			res := runOchamiWithInput(t, addPayloadFor(typ),
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
				w.Write([]byte(`{"metadata":{"name":"thing-1"}}`))
			}))
			defer srv.Close()

			res := runOchamiWithInput(t, addPayloadFor(typ),
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
				w.Write([]byte(`{"metadata":{"name":"thing-1"}}`))
			}))
			defer srv.Close()

			res := runOchamiWithInput(t, envelopePayloadFor(typ),
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
				w.Write([]byte(`{"metadata":{"name":"thing-1"}}`))
			}))
			defer srv.Close()

			res := runOchamiWithInput(t, envelopePayloadFor(typ),
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
				w.Write([]byte(`{"metadata":{"name":"thing-1"}}`))
			}))
			defer srv.Close()

			res := runOchami(t, "metadata", typ, "patch", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t",
				"--set", "description=new", "--unset", "obsolete")
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
				w.Write([]byte(`{"metadata":{"name":"thing-1"}}`))
			}))
			defer srv.Close()

			res := runOchami(t, "metadata", typ, "patch", "some-uid", "--ignore-config", "--uri", srv.URL, "--token", "t",
				"--patch-method", "rfc6902", "-d", `[{"op":"replace","path":"/description","value":"new"}]`)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestMetadataSet_NilResource verifies that a 200 response with a null body
// is accepted: the generated client decodes null into an empty resource, so
// the command's "set returned no resource" check isn't reached over HTTP.
func TestMetadataSet_NilResource(t *testing.T) {
	for _, typ := range metadataTypes {
		t.Run(typ, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`null`))
			}))
			defer srv.Close()

			res := runOchami(t, "metadata", typ, "set", "some-uid",
				"--ignore-config", "--uri", srv.URL, "--token", "t", "-d", addPayloadFor(typ))
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestMetadataGroupList_Success verifies that "metadata group list" exits
// successfully when the service returns a valid list response.
func TestMetadataGroupList_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchami(t, "metadata", "group", "list", "--ignore-config", "--uri", srv.URL, "--token", "faketoken")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if res.exitCode != cli.CodeSuccess {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeSuccess, cli.CodeName(cli.CodeSuccess))
	}
}
