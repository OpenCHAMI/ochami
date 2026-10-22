// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// cloud_init_group_test.go exercises the "cloud-init group" verbs (get
// config/meta-data/raw, add, set, delete, render) end-to-end against an
// httptest.Server: the request each verb issues, output format variants,
// payload input variants, decode/render logic, and the delete confirmation
// prompt. Error arms are covered in cloud_init_group_errors_test.go.

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// ciGroupServer builds an httptest.Server that serves cloud-init group data.
// The groups map is keyed by group name; the "all groups" endpoint
// (GET /admin/groups) returns the whole map, and GET /admin/groups/<name>
// returns a single group. Any status override applies to every request.
func ciGroupServer(t *testing.T, groups map[string]any, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status != 0 && status != http.StatusOK {
			http.Error(w, "boom", status)
			return
		}
		switch {
		case r.URL.Path == "/admin/groups":
			writeJSONResponse(t, w, groups)
		case strings.HasPrefix(r.URL.Path, "/admin/groups/"):
			name := strings.TrimPrefix(r.URL.Path, "/admin/groups/")
			if g, ok := groups[name]; ok {
				writeJSONResponse(t, w, g)
			} else {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{}`))
			}
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
		}
	}))
}

// b64 returns the base64 encoding of s (used to build cloud-config content).
func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

// TestCloudInitGroupGet_RawFormats verifies "get raw" formats output as JSON and
// YAML from a group map returned by the server.
func TestCloudInitGroupGet_RawFormats(t *testing.T) {
	t.Parallel()

	groups := map[string]any{
		"compute": map[string]any{"name": "compute", "meta-data": map[string]any{"foo": "bar"}},
	}
	srv := ciGroupServer(t, groups, http.StatusOK)
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "get", "raw",
			"--uri", srv.URL, "--token", "t", "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
		if !strings.Contains(res.stdout, "compute") {
			t.Errorf("format %s: stdout = %q, want it to contain group name", f, res.stdout)
		}
	}
}

// TestCloudInitGroupGet_Metadata verifies "get meta-data" extracts and formats
// the meta-data map for each group.
func TestCloudInitGroupGet_Metadata(t *testing.T) {
	t.Parallel()

	groups := map[string]any{
		"compute": map[string]any{"name": "compute", "meta-data": map[string]any{"role": "worker"}},
	}
	srv := ciGroupServer(t, groups, http.StatusOK)
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "get", "meta-data",
		"--uri", srv.URL, "--token", "t", "-F", "json")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "worker") {
		t.Errorf("stdout = %q, want it to contain the meta-data value", res.stdout)
	}
}

// TestCloudInitGroupGet_Config verifies "get config" base64-decodes the group's
// cloud-config content and prints it.
func TestCloudInitGroupGet_Config(t *testing.T) {
	t.Parallel()

	content := "#cloud-config\nfoo: bar\n"
	groups := map[string]any{
		"compute": map[string]any{
			"name": "compute",
			"file": map[string]any{"content": b64(content), "encoding": "base64"},
		},
	}
	srv := ciGroupServer(t, groups, http.StatusOK)
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "get", "config",
		"--uri", srv.URL, "--token", "t", "compute")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "foo: bar") {
		t.Errorf("stdout = %q, want decoded cloud-config content", res.stdout)
	}
}

// TestCloudInitGroupGet_ConfigHeaders verifies the --headers always/never modes
// of "get config" over multiple groups.
func TestCloudInitGroupGet_ConfigHeaders(t *testing.T) {
	t.Parallel()

	groups := map[string]any{
		"compute": map[string]any{
			"name": "compute",
			"file": map[string]any{"content": b64("#cloud-config\na: 1\n"), "encoding": "base64"},
		},
		"storage": map[string]any{
			"name": "storage",
			"file": map[string]any{"content": b64("#cloud-config\nb: 2\n"), "encoding": "base64"},
		},
	}
	srv := ciGroupServer(t, groups, http.StatusOK)
	defer srv.Close()

	// --headers always prints a header line even for a single group.
	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "get", "config",
		"--uri", srv.URL, "--token", "t", "--headers", "always", "compute")
	if res.err != nil {
		t.Fatalf("headers=always: unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "group=compute") {
		t.Errorf("headers=always: stdout = %q, want a header line", res.stdout)
	}

	// --headers never omits header lines.
	res = runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "get", "config",
		"--uri", srv.URL, "--token", "t", "--headers", "never", "compute", "storage")
	if res.err != nil {
		t.Fatalf("headers=never: unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if strings.Contains(res.stdout, "group=") {
		t.Errorf("headers=never: stdout = %q, want no header lines", res.stdout)
	}
}

// TestCloudInitGroupAdd_Stdin verifies "group add" reads payload from stdin when
// -d is not supplied.
func TestCloudInitGroupAdd_Stdin(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithInputAndRuntime(t, `[{"name":"compute"}]`,
		"--ignore-config", "cloud-init", "group", "add", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost || gotPath != "/admin/groups" {
		t.Errorf("request = %s %s, want POST /admin/groups", gotMethod, gotPath)
	}
}

// TestCloudInitGroupDelete_ByArgsConfirm verifies answering "y" to the delete
// prompt issues DELETEs for each named group.
func TestCloudInitGroupDelete_ByArgsConfirm(t *testing.T) {
	t.Parallel()

	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInputAndRuntime(t, "y\n",
		"--ignore-config", "cloud-init", "group", "delete", "--uri", srv.URL, "--token", "t",
		"compute", "storage")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "Really delete?") {
		t.Errorf("output = %q, want the confirmation prompt", res.stdout)
	}
	if deletes != 2 {
		t.Errorf("DELETE count = %d, want 2", deletes)
	}
}

// TestCloudInitGroupDelete_ByData verifies "delete -d <payload>" derives the
// group names to delete from the payload.
func TestCloudInitGroupDelete_ByData(t *testing.T) {
	t.Parallel()

	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "delete",
		"--uri", srv.URL, "--token", "t", "--no-confirm", "-d", `[{"name":"compute"},{"name":"storage"}]`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 2 {
		t.Errorf("DELETE count = %d, want 2", deletes)
	}
}

// TestCloudInitGroupRender_Success verifies the full render path: fetch the group's
// jinja config, fetch node meta-data, and render the template to stdout.
func TestCloudInitGroupRender_Success(t *testing.T) {
	t.Parallel()

	tmpl := "## template: jinja\n#cloud-config\nhostname: {{ ds.meta_data.hostname }}\n"
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
		"--uri", srv.URL, "--token", "t", "compute", "x0c0s0b0n0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "node01") {
		t.Errorf("stdout = %q, want rendered hostname", res.stdout)
	}
}

// TestCloudInitGroupRender_WithExtraVars verifies --extra-vars are merged into
// the render context.
func TestCloudInitGroupRender_WithExtraVars(t *testing.T) {
	t.Parallel()

	tmpl := "## template: jinja\n#cloud-config\nx: {{ myvar }}\n"
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
		"--uri", srv.URL, "--token", "t", "--extra-vars", `{"myvar":"hello"}`, "compute", "x0c0s0b0n0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "hello") {
		t.Errorf("stdout = %q, want rendered extra var", res.stdout)
	}
}

// TestCloudInitGroupGet_ConfigEdgeCases verifies that "cloud-init group get
// config" succeeds for a group with empty plain content and fails with
// CodePayload for content that isn't valid base64.
func TestCloudInitGroupGet_ConfigEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		group    string
		wantCode int
	}{
		{name: "empty content", group: `{"name":"compute","file":{"content":"","encoding":"plain"}}`, wantCode: cli.CodeSuccess},
		{name: "invalid base64", group: `{"name":"compute","file":{"content":"%%%","encoding":"base64"}}`, wantCode: cli.CodePayload},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				io.WriteString(w, tc.group)
			}))
			defer srv.Close()
			res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "group", "get", "config",
				"compute", "--uri", srv.URL, "--token", "t")
			if res.exitCode != tc.wantCode || (tc.wantCode == cli.CodeSuccess && res.err != nil) || (tc.wantCode != cli.CodeSuccess && res.err == nil) {
				t.Fatalf("result = (err %v, exit %d), want exit %d (%s)", res.err, res.exitCode, tc.wantCode, cli.CodeName(tc.wantCode))
			}
		})
	}
}

// TestCloudInitGroupGet_Success verifies "cloud-init group get raw" issues GET
// /admin/groups.
func TestCloudInitGroupGet_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "group", "get", "raw", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodGet || gotPath != "/admin/groups" {
		t.Errorf("request = %s %s, want GET /admin/groups", gotMethod, gotPath)
	}
}

// TestCloudInitGroupAdd_Success verifies "cloud-init group add -d <payload>" issues
// POST /admin/groups.
func TestCloudInitGroupAdd_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "group", "add", "--uri", srv.URL, "--token", "t",
		"-d", `[{"name":"compute"}]`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost || gotPath != "/admin/groups" {
		t.Errorf("request = %s %s, want POST /admin/groups", gotMethod, gotPath)
	}
}

// TestCloudInitGroupSet_Success verifies "cloud-init group set -d <payload>"
// issues a PUT.
func TestCloudInitGroupSet_Success(t *testing.T) {
	t.Parallel()

	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "group", "set", "--uri", srv.URL, "--token", "t",
		"-d", `[{"name":"compute"}]`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
}

// TestCloudInitGroupDelete_NoConfirm verifies "cloud-init group delete
// --no-confirm <name>" issues DELETE under /admin/groups.
func TestCloudInitGroupDelete_NoConfirm(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "group", "delete", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "compute")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/admin/groups/compute" {
		t.Errorf("path = %q, want /admin/groups/compute", gotPath)
	}
}

// TestCloudInitGroupRender_EmptyConfig verifies that "cloud-init group render"
// succeeds without rendering anything, and logs a warning, when the group's
// cloud-config is empty.
func TestCloudInitGroupRender_EmptyConfig(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Empty body for the group-config fetch => nothing to render.
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "group", "render", "--uri", srv.URL, "--token", "t",
		"compute", "x0c0s0b0n0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "cloud-config for group compute was empty") {
		t.Errorf("output = %q, want a warning about the empty cloud-config", res.stdout)
	}
}

// TestCloudInitGroupGet_ConfigAndMetaData verifies that "cloud-init group get
// config" and "cloud-init group get meta-data" succeed against a service that
// returns no groups.
func TestCloudInitGroupGet_ConfigAndMetaData(t *testing.T) {
	t.Parallel()

	for _, subcommand := range []string{"config", "meta-data"} {
		t.Run(subcommand, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{}`)
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "cloud-init", "group", "get", subcommand,
				"--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

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

// TestCloudInitGroupRender_SortedInputSources verifies complete output, with
// mappings in sorted key order, for inline, file, and stdin extra variables in
// each supported input format.
func TestCloudInitGroupRender_SortedInputSources(t *testing.T) {
	t.Parallel()
	const template = "## template: jinja\n#cloud-config\nhostname: {{ds.meta_data.hostname}}\ntext: {{text}}\nkeys: {% for k in values %}{{k}};{% endfor %}\n"
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
				if want := "#cloud-config\nhostname: override\ntext: <&é>\nkeys: a;z;\n"; res.stdout != want {
					t.Fatalf("stdout = %q, want %q", res.stdout, want)
				}
			})
		}
	}
}

// TestCloudInitGroupRender_ReferenceTemplate verifies the multi-group mount
// template matches the complete cloud-init output through the HTTP command path.
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
