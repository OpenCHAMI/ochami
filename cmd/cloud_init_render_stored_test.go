// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestCloudInitRender_StoredGroupSources verifies plain and base64 group content
// uses only the authenticated raw-group endpoint, without inferring a node or
// merging stored group metadata into the supplied or empty rendering context.
func TestCloudInitRender_StoredGroupSources(t *testing.T) {
	t.Parallel()
	for _, encoding := range []string{"plain", "base64"} {
		for _, metadata := range []string{"empty", "yaml", "json", "stdin", "extra-vars"} {
			t.Run(encoding+"/"+metadata, func(t *testing.T) {
				t.Parallel()
				content := "{{ds.meta_data.hostname|default('none')}}|{{ds.meta_data['instance-id']|default('none')}}|{{role}}\n"
				if encoding == "base64" {
					content = base64.StdEncoding.EncodeToString([]byte(content))
				}
				var requests atomic.Int32
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if r.Method != http.MethodGet || r.URL.Path != "/admin/groups/compute" {
						t.Errorf("request = %s %s, want GET /admin/groups/compute", r.Method, r.URL.Path)
						http.NotFound(w, r)
						return
					}
					if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
						t.Errorf("authorization = %q, want Bearer test-token", got)
					}
					writeJSONResponse(t, w, map[string]any{
						"name":      "compute",
						"meta-data": map[string]any{"hostname": "stored-value", "instance-id": "stored-node"},
						"file":      map[string]any{"content": content, "encoding": encoding},
					})
				}))
				t.Cleanup(srv.Close)
				args := []string{"--ignore-config", "cloud-init", "render", "--group", "compute", "--uri", srv.URL, "--token", "test-token"}
				stdin, want := "", "none|none|compute"
				extra := `{"role":"compute"}`
				if metadata == "extra-vars" {
					extra = `{"role":"compute","ds":{"meta_data":{"hostname":"extra-host","instance-id":"extra-node"}}}`
					want = "extra-host|extra-node|compute"
				} else if metadata != "empty" {
					data := "hostname: file-host\ninstance-id: file-node\n"
					if metadata == "json" {
						data = `{"hostname":"file-host","instance-id":"file-node"}`
					}
					path := writeRenderInput(t, "metadata", data)
					if metadata == "stdin" {
						path, stdin = "-", data
					}
					args = append(args, "--metadata", path)
					want = "file-host|file-node|compute"
				}
				args = append(args, "--extra-vars", extra)
				res := runOchamiWithInputAndRuntime(t, stdin, args...)
				if res.err != nil || res.stdout != want || requests.Load() != 1 {
					t.Fatalf("stdout = %q, error = %v, requests = %d, want %q and one request", res.stdout, res.err, requests.Load(), want)
				}
			})
		}
	}
}

// TestCloudInitRender_StoredGroupReference verifies both storage encodings render
// the multi-group fixture exactly as the pinned Python reference.
func TestCloudInitRender_StoredGroupReference(t *testing.T) {
	t.Parallel()
	read := func(name string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("..", "pkg", "cloudinit", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	for _, encoding := range []string{"plain", "base64"} {
		t.Run(encoding, func(t *testing.T) {
			t.Parallel()
			content := read("cloud-init-original.j2")
			if encoding == "base64" {
				content = base64.StdEncoding.EncodeToString([]byte(content))
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/admin/groups/compute" {
					t.Errorf("unexpected request path %q", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				writeJSONResponse(t, w, map[string]any{"name": "compute", "file": map[string]any{"content": content, "encoding": encoding}})
			}))
			t.Cleanup(srv.Close)
			res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "render", "--group", "compute", "--uri", srv.URL, "--token", "t",
				"--metadata", filepath.Join("..", "pkg", "cloudinit", "testdata", "cloud-init-original.metadata.json"))
			if want := read("cloud-init-original.yaml"); res.err != nil || res.stdout != want {
				t.Fatalf("stdout = %q, error = %v, want %q without error", res.stdout, res.err, want)
			}
		})
	}
}

// TestCloudInitRender_StoredGroupFailures verifies HTTP and invalid stored
// document failures retain CodeHTTP or CodePayload and publish no stdout.
func TestCloudInitRender_StoredGroupFailures(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, body       string
		status, wantCode int
	}{
		{name: "not found", status: http.StatusNotFound, wantCode: cli.CodeHTTP},
		{name: "HTTP failure", status: http.StatusInternalServerError, wantCode: cli.CodeHTTP},
		{name: "empty response", wantCode: cli.CodePayload},
		{name: "malformed JSON", body: "{", wantCode: cli.CodePayload},
		{name: "null response", body: "null", wantCode: cli.CodePayload},
		{name: "array response", body: "[]", wantCode: cli.CodePayload},
		{name: "invalid content type", body: `{"file":{"content":1,"encoding":"plain"}}`, wantCode: cli.CodePayload},
		{name: "unknown encoding", body: `{"file":{"content":"prefix","encoding":"gzip"}}`, wantCode: cli.CodePayload},
		{name: "missing encoding", body: `{"file":{"content":"prefix"}}`, wantCode: cli.CodePayload},
		{name: "invalid base64", body: `{"file":{"content":"%%%","encoding":"base64"}}`, wantCode: cli.CodePayload},
		{name: "template failure", body: `{"file":{"content":"prefix{% include 'missing' %}","encoding":"plain"}}`, wantCode: cli.CodePayload},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/admin/groups/compute" {
					t.Errorf("unexpected request path %q", r.URL.Path)
				}
				if tc.status != 0 {
					w.WriteHeader(tc.status)
				}
				io.WriteString(w, tc.body)
			}))
			t.Cleanup(srv.Close)
			var stdout bytes.Buffer
			res := runOchamiWithOutputWriter(t, &stdout, "--ignore-config", "cloud-init", "render", "--group", "compute", "--uri", srv.URL, "--token", "t")
			if res.err == nil || res.exitCode != tc.wantCode {
				t.Fatalf("error = %v, exit code = %d, want %d (%s)", res.err, res.exitCode, tc.wantCode, cli.CodeName(tc.wantCode))
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
		})
	}
}

// TestCloudInitRender_EmptyStoredGroup verifies a group without template content
// succeeds with a warning and does not attempt to fetch node metadata.
func TestCloudInitRender_EmptyStoredGroup(t *testing.T) {
	t.Parallel()
	for _, body := range []string{`{"name":"compute"}`, `{"name":"compute","file":{"content":"","encoding":"plain"}}`} {
		t.Run(body, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/admin/groups/compute" {
					t.Errorf("unexpected request path %q", r.URL.Path)
				}
				io.WriteString(w, body)
			}))
			t.Cleanup(srv.Close)
			res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "render", "--group", "compute", "--uri", srv.URL, "--token", "t")
			if res.err != nil || !strings.Contains(res.stdout, "stored cloud-config for group compute was empty") {
				t.Fatalf("output = %q, error = %v, want empty-template warning without error", res.stdout, res.err)
			}
		})
	}
}

// TestCloudInitRender_NodeSelectionDoesNotFallback verifies an unsuccessful
// node-specific selection returns CodeHTTP without requesting stored content.
func TestCloudInitRender_NodeSelectionDoesNotFallback(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/admin/impersonation/node01/compute.yaml" {
			t.Errorf("unexpected request path %q", r.URL.Path)
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "render", "--group", "compute", "--node", "node01", "--uri", srv.URL, "--token", "t")
	if res.err == nil || res.exitCode != cli.CodeHTTP {
		t.Fatalf("error = %v, exit code = %d, want %d (%s)", res.err, res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
	if requests.Load() != 1 {
		t.Fatalf("requests = %d, want 1", requests.Load())
	}
}
