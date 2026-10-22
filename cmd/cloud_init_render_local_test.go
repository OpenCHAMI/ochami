// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/cloudinit"
)

func writeRenderInput(t *testing.T, name, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestCloudInitRender_LocalReference verifies all Python fixtures match
// exactly when templates and JSON or YAML metadata come from local files.
func TestCloudInitRender_LocalReference(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "pkg", "cloudinit", "testdata")
	read := func(name string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	var manifest struct {
		Fixtures []struct {
			Name, Template, Metadata, Expected string
		} `json:"fixtures"`
	}
	if err := json.Unmarshal(read("reference.json"), &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Fixtures) != 5 {
		t.Fatalf("fixture count = %d, want 5", len(manifest.Fixtures))
	}
	for _, fixture := range manifest.Fixtures {
		for _, metadataFormat := range []string{"json", "yaml"} {
			t.Run(fixture.Name+"/"+metadataFormat, func(t *testing.T) {
				t.Parallel()
				metadataPath := filepath.Join(root, fixture.Metadata)
				if metadataFormat == "yaml" {
					var node yaml.Node
					if err := yaml.Unmarshal(read(fixture.Metadata), &node); err != nil {
						t.Fatal(err)
					}
					data, err := yaml.Marshal(&node)
					if err != nil {
						t.Fatal(err)
					}
					metadataPath = writeRenderInput(t, "metadata.yaml", string(data))
				}
				res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "render",
					"--template", filepath.Join(root, fixture.Template), "--metadata", metadataPath)
				if res.err != nil {
					t.Fatal(res.err)
				}
				if want := string(read(fixture.Expected)); res.stdout != want {
					t.Fatalf("stdout = %q, want %q", res.stdout, want)
				}
			})
		}
	}
}

// TestCloudInitRender_LocalInputSources verifies omitted metadata, empty
// templates, variable replacement, and each source's exclusive use of stdin.
func TestCloudInitRender_LocalInputSources(t *testing.T) {
	t.Parallel()
	const template = "{{ds.meta_data.hostname|default('none')}}|{{label|default('none')}}\n"
	for _, tc := range []struct {
		name, stdinSource, metadata, extra, inputFormat, want string
		emptyTemplate                                         bool
	}{
		{name: "no metadata", want: "none|none"},
		{name: "empty template", emptyTemplate: true},
		{name: "template stdin", stdinSource: "template", metadata: "hostname: node01", want: "node01|none"},
		{name: "metadata stdin", stdinSource: "metadata", metadata: "hostname: node01", want: "node01|none"},
		{name: "extra vars stdin", stdinSource: "extra", extra: `{"label":"compute"}`, want: "none|compute"},
		{name: "JSON override", metadata: "hostname: node01", extra: `{"ds":{"meta_data":{"hostname":"replacement"}},"label":"compute"}`, want: "replacement|compute"},
		{name: "YAML override", metadata: `{"hostname":"node01"}`, extra: "ds: {meta_data: {hostname: replacement}}\nlabel: compute", inputFormat: "yaml", want: "replacement|compute"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			source := template
			if tc.emptyTemplate {
				source = ""
			}
			templatePath := writeRenderInput(t, "template.j2", source)
			stdin := ""
			if tc.stdinSource == "template" {
				templatePath, stdin = "-", source
			}
			args := []string{"--ignore-config", "cloud-init", "render", "--template", templatePath,
				"--uri", "http://127.0.0.1:1"}
			if tc.metadata != "" {
				path := writeRenderInput(t, "metadata", tc.metadata)
				if tc.stdinSource == "metadata" {
					path, stdin = "-", tc.metadata
				}
				args = append(args, "--metadata", path)
			}
			if tc.extra != "" {
				value := tc.extra
				if tc.stdinSource == "extra" {
					value, stdin = "@-", tc.extra
				}
				args = append(args, "--extra-vars", value)
			}
			if tc.inputFormat != "" {
				args = append(args, "--format-input", tc.inputFormat)
			}
			res := runOchamiWithInputAndRuntime(t, stdin, args...)
			if res.err != nil || res.stdout != tc.want {
				t.Fatalf("stdout = %q, error = %v, want %q without error", res.stdout, res.err, tc.want)
			}
		})
	}
}

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

// TestCloudInitRender_HybridMetadata verifies a local template fetches
// only node metadata and sends the configured authorization token.
func TestCloudInitRender_HybridMetadata(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/admin/impersonation/node01/meta-data" {
			t.Errorf("request = %s %s, want GET /admin/impersonation/node01/meta-data", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("authorization = %q, want Bearer test-token", got)
		}
		io.WriteString(w, "hostname: node01\n")
	}))
	t.Cleanup(srv.Close)
	path := writeRenderInput(t, "template.j2", "{{ds.meta_data.hostname}}|{{role}}\n")
	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "render", "--template", path, "--node", "node01",
		"--uri", srv.URL, "--token", "test-token", "--extra-vars", `{"role":"compute"}`)
	if res.err != nil || res.stdout != "node01|compute" || requests.Load() != 1 {
		t.Fatalf("stdout = %q, error = %v, requests = %d, want node01|compute and one request", res.stdout, res.err, requests.Load())
	}
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
