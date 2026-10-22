// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/cloudinit"
)

// TestCloudInitRender_LocalUsageErrors verifies invalid modes and competing
// stdin sources return CodeUsage before opening files or reading input.
func TestCloudInitRender_LocalUsageErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "missing template source"},
		{name: "node without template source", args: []string{"--node", "node01"}},
		{name: "too many remote arguments", args: []string{"compute", "node01", "extra"}},
		{name: "empty template flag", args: []string{"--template="}},
		{name: "two arguments with template", args: []string{"--template", "missing", "compute", "node01"}},
		{name: "both template sources", args: []string{"--template", "missing", "--group", "compute", "--node", "node01"}},
		{name: "empty group flag", args: []string{"--group=", "--node", "node01"}},
		{name: "empty node flag", args: []string{"--group", "compute", "--node="}},
		{name: "group metadata and extras stdin", args: []string{"--group", "compute", "--node", "node01", "--metadata", "-", "--extra-vars", "@-"}},
		{name: "metadata only", args: []string{"--metadata", "missing"}},
		{name: "empty metadata flag", args: []string{"--template", "missing", "--metadata="}},
		{name: "node and local metadata", args: []string{"--template", "missing", "--metadata", "missing", "--node", "node01"}},
		{name: "template and metadata stdin", args: []string{"--template", "-", "--metadata", "-"}},
		{name: "template and extras stdin", args: []string{"--template", "-", "--extra-vars", "@-"}},
		{name: "metadata and extras stdin", args: []string{"--template", "missing", "--metadata", "-", "--extra-vars", "@-"}},
		{name: "three stdin sources", args: []string{"--template", "-", "--metadata", "-", "--extra-vars", "@-"}},
		{name: "hybrid competing stdin", args: []string{"--template", "-", "--extra-vars", "@-", "--node", "node01"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			args := append([]string{"--ignore-config", "cloud-init", "render"}, tc.args...)
			var stdout, stderr bytes.Buffer
			reader := &renderCountingReader{}
			rt := cli.NewTestRuntime(reader, &stdout, &stderr)
			root := NewRootCmd()
			root.SetContext(cli.ContextWithRuntime(t.Context(), rt))
			root.SetArgs(args)
			err := root.Execute()
			if err == nil || cli.ExitCode(err) != cli.CodeUsage {
				t.Fatalf("error = %v, exit code = %d, want %d (%s)", err, cli.ExitCode(err), cli.CodeUsage, cli.CodeName(cli.CodeUsage))
			}
			if reader.reads != 0 || stdout.Len() != 0 {
				t.Fatalf("stdin reads = %d, stdout = %q, want neither", reader.reads, stdout.String())
			}
		})
	}
}

type renderCountingReader struct{ reads int }

func (r *renderCountingReader) Read([]byte) (int, error) {
	r.reads++
	return 0, io.EOF
}

// TestCloudInitRender_LocalFailures verifies local source and rendering
// failures return CodePayload without stdout and retain structured causes.
func TestCloudInitRender_LocalFailures(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, template, metadata, extra  string
		missingTemplate, missingMetadata bool
		stage                            cloudinit.RenderStage
	}{
		{name: "missing template", missingTemplate: true},
		{name: "missing metadata", template: "prefix", missingMetadata: true},
		{name: "malformed metadata", template: "prefix", metadata: "x: [", stage: cloudinit.StageMetadata},
		{name: "malformed extra vars", template: "prefix", extra: "{", stage: cloudinit.StageExtraVars},
		{name: "compile", template: "prefix{% if", stage: cloudinit.StageCompile},
		{name: "execute", template: "prefix{% include 'missing' %}", stage: cloudinit.StageExecute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "missing")
			if !tc.missingTemplate {
				path = writeRenderInput(t, "template.j2", tc.template)
			}
			args := []string{"--ignore-config", "cloud-init", "render", "--template", path}
			if tc.missingMetadata {
				args = append(args, "--metadata", filepath.Join(t.TempDir(), "missing"))
			} else if tc.metadata != "" {
				args = append(args, "--metadata", writeRenderInput(t, "metadata", tc.metadata))
			}
			if tc.extra != "" {
				args = append(args, "--extra-vars", tc.extra)
			}
			var stdout, stderr bytes.Buffer
			rt := cli.NewTestRuntime(strings.NewReader(""), &stdout, &stderr)
			root := NewRootCmd()
			root.SetContext(cli.ContextWithRuntime(t.Context(), rt))
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetArgs(args)
			err := root.Execute()
			if err == nil || cli.ExitCode(err) != cli.CodePayload {
				t.Fatalf("error = %v, exit code = %d, want %d (%s)", err, cli.ExitCode(err), cli.CodePayload, cli.CodeName(cli.CodePayload))
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if tc.stage != "" {
				var renderErr *cloudinit.RenderError
				if !errors.As(err, &renderErr) || renderErr.Stage != tc.stage {
					t.Fatalf("error = %v, want RenderError stage %s", err, tc.stage)
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("error = %v, want missing file cause", err)
			}
		})
	}
}

// TestCloudInitRender_LocalOutputFailures verifies local rendering returns
// CodePayload and preserves writer errors and short-write causes.
func TestCloudInitRender_LocalOutputFailures(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		writer io.Writer
		want   error
	}{
		{name: "writer error", writer: &failingOutputWriter{}, want: errTestWriter},
		{name: "short write", writer: renderShortWriter{}, want: io.ErrShortWrite},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := writeRenderInput(t, "template.j2", "rendered")
			res := runOchamiWithOutputWriter(t, tc.writer, "--ignore-config", "cloud-init", "render", "--template", path)
			if !errors.Is(res.err, tc.want) {
				t.Fatalf("error = %v, want %v", res.err, tc.want)
			}
			if res.exitCode != cli.CodePayload {
				t.Fatalf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
			}
		})
	}
}

// TestCloudInitRender_HybridFailures verifies metadata HTTP failures and
// unreadable local templates retain CodeHTTP and CodePayload respectively.
func TestCloudInitRender_HybridFailures(t *testing.T) {
	t.Parallel()
	for _, missing := range []bool{false, true} {
		name := "metadata HTTP error"
		if missing {
			name = "missing template"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.URL.Path != "/admin/impersonation/node01/meta-data" {
					t.Errorf("unexpected request path %q", r.URL.Path)
				}
				http.Error(w, "unavailable", http.StatusInternalServerError)
			}))
			t.Cleanup(srv.Close)
			path := writeRenderInput(t, "template.j2", "prefix{{ds.meta_data.hostname}}")
			wantCode, wantRequests := cli.CodeHTTP, int32(1)
			if missing {
				path = filepath.Join(t.TempDir(), "missing")
				wantCode, wantRequests = cli.CodePayload, 0
			}
			res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "render", "--template", path, "--node", "node01", "--uri", srv.URL, "--token", "t")
			if res.err == nil || res.exitCode != wantCode {
				t.Fatalf("error = %v, exit code = %d, want %d (%s)", res.err, res.exitCode, wantCode, cli.CodeName(wantCode))
			}
			if got := requests.Load(); got != wantRequests {
				t.Fatalf("requests = %d, want %d", got, wantRequests)
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
