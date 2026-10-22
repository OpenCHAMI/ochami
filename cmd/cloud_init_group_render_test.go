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
	"testing"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/cloudinit"
)

func cloudInitRenderServer(t *testing.T, template, metadata string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "compute.yaml"):
			io.WriteString(w, template)
		case strings.HasSuffix(r.URL.Path, "meta-data"):
			io.WriteString(w, metadata)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestCloudInitGroupRender_OrderedInputSources verifies complete output with
// inline, file, and stdin extra variables in each supported input format.
func TestCloudInitGroupRender_OrderedInputSources(t *testing.T) {
	t.Parallel()
	const template = "#cloud-config\nhostname: {{ds.meta_data.hostname}}\ntext: {{text}}\nkeys: {% for k in values %}{{k}};{% endfor %}\n"
	for _, inputFormat := range []string{"json", "json-pretty", "yaml"} {
		for _, source := range []string{"inline", "file", "stdin"} {
			t.Run(inputFormat+"/"+source, func(t *testing.T) {
				t.Parallel()
				srv := cloudInitRenderServer(t, template, "hostname: original\n")
				payload := `{"values":{"z":1,"a":2},"text":"<&é>","ds":{"meta_data":{"hostname":"override"}}}`
				if inputFormat == "yaml" {
					payload = "values: {z: 1, a: 2}\ntext: <&é>\nds: {meta_data: {hostname: override}}\n"
				}
				value, stdin := payload, ""
				switch source {
				case "file":
					path := filepath.Join(t.TempDir(), "vars")
					if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
						t.Fatal(err)
					}
					value = "@" + path
				case "stdin":
					value, stdin = "@-", payload
				}
				res := runOchamiWithInputAndRuntime(t, stdin, "--ignore-config", "cloud-init", "group", "render",
					"compute", "node01", "--uri", srv.URL, "--token", "t", "--format-input", inputFormat, "--extra-vars", value)
				if res.err != nil {
					t.Fatal(res.err)
				}
				if want := "#cloud-config\nhostname: override\ntext: <&é>\nkeys: z;a;"; res.stdout != want {
					t.Fatalf("stdout = %q, want %q", res.stdout, want)
				}
			})
		}
	}
}

// TestCloudInitGroupRender_ReferenceTemplate verifies the multi-group mount
// template matches the complete Python output through the HTTP command path.
func TestCloudInitGroupRender_ReferenceTemplate(t *testing.T) {
	t.Parallel()
	read := func(name string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("..", "pkg", "cloudinit", "testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	srv := cloudInitRenderServer(t, read("cloud-init-original.j2"), read("cloud-init-original.metadata.json"))
	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "group", "render", "compute", "node01", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatal(res.err)
	}
	if want := read("cloud-init-original.yaml"); res.stdout != want {
		t.Fatalf("stdout = %q, want %q", res.stdout, want)
	}
}

// TestCloudInitGroupRender_TemplateFailures verifies compilation and execution
// failures preserve their library error and return CodePayload without stdout.
func TestCloudInitGroupRender_TemplateFailures(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, template string
		stage          cloudinit.RenderStage
	}{
		{name: "compile", template: "prefix{% if", stage: cloudinit.StageCompile},
		{name: "execute", template: "prefix{% include 'missing' %}", stage: cloudinit.StageExecute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := cloudInitRenderServer(t, tc.template, "{}")
			var stdout, stderr bytes.Buffer
			rt := cli.NewTestRuntime(strings.NewReader(""), &stdout, &stderr)
			root := NewRootCmd()
			root.SetContext(cli.ContextWithRuntime(t.Context(), rt))
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetArgs([]string{"--ignore-config", "cloud-init", "group", "render", "compute", "node01", "--uri", srv.URL, "--token", "t"})
			err := root.Execute()
			var renderErr *cloudinit.RenderError
			if !errors.As(err, &renderErr) || renderErr.Stage != tc.stage {
				t.Fatalf("error = %v, want RenderError stage %s", err, tc.stage)
			}
			if code := cli.ExitCode(err); code != cli.CodePayload {
				t.Fatalf("exit code = %d, want %d (%s)", code, cli.CodePayload, cli.CodeName(cli.CodePayload))
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
		})
	}
}

type renderShortWriter struct{}

func (renderShortWriter) Write([]byte) (int, error) { return 0, nil }

// TestCloudInitGroupRender_OutputFailures verifies writer errors and short
// writes remain inspectable and resolve to CodePayload.
func TestCloudInitGroupRender_OutputFailures(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		writer io.Writer
		want   error
	}{
		{name: "failure", writer: &failingOutputWriter{}, want: errTestWriter},
		{name: "short write", writer: renderShortWriter{}, want: io.ErrShortWrite},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := cloudInitRenderServer(t, "rendered", "{}")
			res := runOchamiWithOutputWriter(t, tc.writer, "--ignore-config", "cloud-init", "group", "render", "compute", "node01", "--uri", srv.URL, "--token", "t")
			if !errors.Is(res.err, tc.want) {
				t.Fatalf("error = %v, want %v", res.err, tc.want)
			}
			if res.exitCode != cli.CodePayload {
				t.Fatalf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
			}
		})
	}
}
