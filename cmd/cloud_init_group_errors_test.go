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
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/cloudinit"
)

// TestCloudInitGroupGet_ByIDHTTPError verifies an unsuccessful HTTP response on a
// per-group fetch (args form) resolves to CodeHTTP via the per-item error
// aggregation ("completed with errors").
func TestCloudInitGroupGet_ByIDHTTPError(t *testing.T) {
	t.Parallel()

	srv := ciGroupServer(t, nil, http.StatusNotFound)
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "get", "raw",
		"--uri", srv.URL, "--token", "t", "compute")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestCloudInitGroupAdd_HTTPError verifies a failing POST resolves to CodeHTTP
// via the per-item aggregation.
func TestCloudInitGroupAdd_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "add",
		"--uri", srv.URL, "--token", "t", "-d", `[{"name":"compute"}]`)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestCloudInitGroupSet_HTTPError verifies a failing PUT resolves to CodeHTTP.
func TestCloudInitGroupSet_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "set",
		"--uri", srv.URL, "--token", "t", "-d", `[{"name":"compute"}]`)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestCloudInitGroupDelete_Abort verifies answering "n" aborts without a request.
func TestCloudInitGroupDelete_Abort(t *testing.T) {
	t.Parallel()

	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInputAndRuntime(t, "n\n",
		"--ignore-config", "cloud-init", "group", "delete", "--uri", srv.URL, "--token", "t", "compute")
	if res.exitCode != cli.CodeDeclined {
		t.Fatalf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodeDeclined, cli.CodeName(cli.CodeDeclined))
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0 (user declined)", deletes)
	}
}

// TestCloudInitGroupDelete_NoArgsUsage verifies delete with neither -d nor args
// is a usage error.
func TestCloudInitGroupDelete_NoArgsUsage(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "delete",
		"--uri", "http://127.0.0.1:1", "--token", "t", "--no-confirm")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestCloudInitGroupDelete_DataWithArgs verifies that delete with -d and group
// name arguments is a usage error whose message says the combination is
// rejected, not that the arguments are ignored.
func TestCloudInitGroupDelete_DataWithArgs(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "delete",
		"--uri", "http://127.0.0.1:1", "--token", "t", "--no-confirm",
		"-d", `[{"name":"compute"}]`, "extra-arg")
	if res.err == nil || res.exitCode != cli.CodeUsage {
		t.Fatalf("err=%v exit=%d, want %d (%s)", res.err, res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
	if want := "-d cannot be used with arguments, got [extra-arg]"; !strings.Contains(res.err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", res.err.Error(), want)
	}
}

// TestCloudInitGroupDelete_HTTPError verifies a failing DELETE resolves to
// CodeHTTP via the per-item aggregation.
func TestCloudInitGroupDelete_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "delete",
		"--uri", srv.URL, "--token", "t", "--no-confirm", "compute")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestCloudInitGroupRender_HTTPError verifies an unsuccessful HTTP response on
// the group-config fetch resolves to CodeHTTP.
func TestCloudInitGroupRender_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "render",
		"--uri", srv.URL, "--token", "t", "compute", "x0c0s0b0n0")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestCloudInitGroupRender_MetadataHTTPError verifies an unsuccessful HTTP
// response on the node meta-data fetch (after a successful config fetch)
// resolves to CodeHTTP.
func TestCloudInitGroupRender_MetadataHTTPError(t *testing.T) {
	t.Parallel()

	tmpl := "## template: jinja\n#cloud-config\nhostname: {{ ds.meta_data.hostname }}\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "compute.yaml"):
			w.Write([]byte(tmpl))
		default:
			http.Error(w, "bad", http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "render",
		"--uri", srv.URL, "--token", "t", "compute", "x0c0s0b0n0")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestCloudInitGroupGet_ByIDNetworkError verifies that "cloud-init group get
// raw <name>" fails with CodeNetwork when the service can't be reached.
func TestCloudInitGroupGet_ByIDNetworkError(t *testing.T) {
	t.Parallel()

	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "get", "raw",
		"--uri", url, "--token", "t", "compute")
	if res.err == nil {
		t.Fatal("expected a network error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}

// TestCloudInitGroupRender_MalformedExtraVars verifies malformed --extra-vars is
// a payload error.
func TestCloudInitGroupRender_MalformedExtraVars(t *testing.T) {
	t.Parallel()

	tmpl := "## template: jinja\n#cloud-config\nx: {{ ds.meta_data.hostname }}\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "compute.yaml"):
			w.Write([]byte(tmpl))
		case strings.HasSuffix(r.URL.Path, "meta-data"):
			w.Write([]byte("hostname: node01\n"))
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "render",
		"--uri", srv.URL, "--token", "t", "--extra-vars", `not json`, "compute", "x0c0s0b0n0")
	if res.err == nil {
		t.Fatal("expected a payload error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestCloudInitGroupRender_PayloadErrors verifies that "cloud-init group
// render" fails with CodePayload for malformed node meta-data or a malformed
// template.
func TestCloudInitGroupRender_PayloadErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		template string
		metadata string
	}{
		{name: "malformed metadata", template: `{{ ds.meta_data.hostname }}`, metadata: `: invalid`},
		{name: "malformed template", template: `{% if`, metadata: `hostname: node01`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "compute.yaml") {
					io.WriteString(w, tc.template)
					return
				}
				io.WriteString(w, tc.metadata)
			}))
			defer srv.Close()
			res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "group", "render", "compute", "node01",
				"--uri", srv.URL, "--token", "t")
			if res.err == nil || res.exitCode != cli.CodePayload {
				t.Fatalf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
			}
		})
	}
}

// TestCloudInitGroupRender_LegacyInputs verifies the original command rejects
// new source-selection flags and missing or blank arguments as CodeUsage.
func TestCloudInitGroupRender_LegacyInputs(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "template flag", args: []string{"compute", "node01", "--template", "missing"}},
		{name: "metadata flag", args: []string{"compute", "node01", "--metadata", "missing"}},
		{name: "metadata stdin flag", args: []string{"compute", "node01", "--metadata", "-"}},
		{name: "group flag", args: []string{"compute", "node01", "--group", "compute"}},
		{name: "node flag", args: []string{"compute", "node01", "--node", "node01"}},
		{name: "no arguments"},
		{name: "missing node", args: []string{"compute"}},
		{name: "empty node", args: []string{"compute", ""}},
		{name: "blank node", args: []string{"compute", " "}},
		{name: "empty group", args: []string{"", "node01"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			args := append([]string{"--ignore-config", "cloud-init", "group", "render"}, tc.args...)
			res := runOchamiWithRuntime(t, args...)
			if res.err == nil || res.exitCode != cli.CodeUsage {
				t.Fatalf("error = %v, exit code = %d, want %d (%s)", res.err, res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
			}
		})
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
