// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// cloud_init_group_test.go exercises the "cloud-init group" verbs (get
// config/meta-data/raw, add, delete, render) end-to-end against an
// httptest.Server, covering the success-path branches that the happy paths in
// cloud_init_test.go do not reach: output format variants, payload input
// variants, decode/render logic, and the delete confirmation prompt. Error arms
// are covered in cloud_init_group_errors_test.go.

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
	groups := map[string]any{
		"compute": map[string]any{"name": "compute", "meta-data": map[string]any{"foo": "bar"}},
	}
	srv := ciGroupServer(t, groups, http.StatusOK)
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchami(t, "cloud-init", "group", "get", "raw", "--ignore-config",
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
	groups := map[string]any{
		"compute": map[string]any{"name": "compute", "meta-data": map[string]any{"role": "worker"}},
	}
	srv := ciGroupServer(t, groups, http.StatusOK)
	defer srv.Close()

	res := runOchami(t, "cloud-init", "group", "get", "meta-data", "--ignore-config",
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
	content := "#cloud-config\nfoo: bar\n"
	groups := map[string]any{
		"compute": map[string]any{
			"name": "compute",
			"file": map[string]any{"content": b64(content), "encoding": "base64"},
		},
	}
	srv := ciGroupServer(t, groups, http.StatusOK)
	defer srv.Close()

	res := runOchami(t, "cloud-init", "group", "get", "config", "--ignore-config",
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
	res := runOchami(t, "cloud-init", "group", "get", "config", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "--headers", "always", "compute")
	if res.err != nil {
		t.Fatalf("headers=always: unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "group=compute") {
		t.Errorf("headers=always: stdout = %q, want a header line", res.stdout)
	}

	// --headers never omits header lines.
	res = runOchami(t, "cloud-init", "group", "get", "config", "--ignore-config",
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
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, `[{"name":"compute"}]`,
		"cloud-init", "group", "add", "--ignore-config", "--uri", srv.URL, "--token", "t")
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
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "y\n",
		"cloud-init", "group", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
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
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "group", "delete", "--ignore-config",
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

	res := runOchami(t, "cloud-init", "group", "render", "--ignore-config",
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

	res := runOchami(t, "cloud-init", "group", "render", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "--extra-vars", `{"myvar":"hello"}`, "compute", "x0c0s0b0n0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "hello") {
		t.Errorf("stdout = %q, want rendered extra var", res.stdout)
	}
}
