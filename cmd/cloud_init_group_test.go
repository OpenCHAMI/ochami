// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// cloud_init_group_test.go exercises the branchy "cloud-init group" verbs
// (get config/meta-data/raw, add, set, delete, render) end-to-end against an
// httptest.Server. It focuses on the recurring branch families that the happy
// paths in cloud_init_test.go do not reach: HTTP-error mapping, per-item error
// aggregation, payload input variants, output format variants, decode/render
// logic, and the delete confirmation prompt.

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
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
				_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
			}
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
		}
	}))
}

// b64 returns the base64 encoding of s (used to build cloud-config content).
func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

// TestCloudInitGroupGetRawFormats verifies "get raw" formats output as JSON and
// YAML from a group map returned by the server.
func TestCloudInitGroupGetRawFormats(t *testing.T) {
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

// TestCloudInitGroupGetHTTPError verifies an unsuccessful HTTP response on the
// all-groups fetch resolves to CodeHTTP.
func TestCloudInitGroupGetHTTPError(t *testing.T) {
	srv := ciGroupServer(t, nil, http.StatusInternalServerError)
	defer srv.Close()

	res := runOchami(t, "cloud-init", "group", "get", "raw", "--ignore-config",
		"--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestCloudInitGroupGetByIDHTTPError verifies an unsuccessful HTTP response on a
// per-group fetch (args form) resolves to CodeHTTP via the per-item error
// aggregation ("completed with errors").
func TestCloudInitGroupGetByIDHTTPError(t *testing.T) {
	srv := ciGroupServer(t, nil, http.StatusNotFound)
	defer srv.Close()

	res := runOchami(t, "cloud-init", "group", "get", "raw", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "compute")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestCloudInitGroupGetNetworkError verifies pointing at a closed port resolves
// to CodeNetwork.
func TestCloudInitGroupGetNetworkError(t *testing.T) {
	srv := ciGroupServer(t, nil, http.StatusOK)
	url := srv.URL
	srv.Close() // close immediately so the connection is refused

	res := runOchami(t, "cloud-init", "group", "get", "raw", "--ignore-config",
		"--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestCloudInitGroupGetMetadata verifies "get meta-data" extracts and formats
// the meta-data map for each group.
func TestCloudInitGroupGetMetadata(t *testing.T) {
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

// TestCloudInitGroupGetConfig verifies "get config" base64-decodes the group's
// cloud-config content and prints it.
func TestCloudInitGroupGetConfig(t *testing.T) {
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

// TestCloudInitGroupGetConfigHeaders verifies the --headers always/never modes
// of "get config" over multiple groups.
func TestCloudInitGroupGetConfigHeaders(t *testing.T) {
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

// TestCloudInitGroupAddStdin verifies "group add" reads payload from stdin when
// -d is not supplied.
func TestCloudInitGroupAddStdin(t *testing.T) {
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

// TestCloudInitGroupAddMalformedPayload verifies malformed inline payload data
// resolves to CodePayload.
func TestCloudInitGroupAddMalformedPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "group", "add", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "-d", `not json`)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (CodePayload)", res.exitCode, cli.CodePayload)
	}
}

// TestCloudInitGroupAddHTTPError verifies a failing POST resolves to CodeHTTP
// via the per-item aggregation.
func TestCloudInitGroupAddHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "group", "add", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "-d", `[{"name":"compute"}]`)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestCloudInitGroupSetHTTPError verifies a failing PUT resolves to CodeHTTP.
func TestCloudInitGroupSetHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "group", "set", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "-d", `[{"name":"compute"}]`)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestCloudInitGroupDeleteByArgsConfirm verifies answering "y" to the delete
// prompt issues DELETEs for each named group.
func TestCloudInitGroupDeleteByArgsConfirm(t *testing.T) {
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
	if deletes != 2 {
		t.Errorf("DELETE count = %d, want 2", deletes)
	}
}

// TestCloudInitGroupDeleteAbort verifies answering "n" aborts without a request.
func TestCloudInitGroupDeleteAbort(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "n\n",
		"cloud-init", "group", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t", "compute")
	if res.err != nil {
		t.Fatalf("unexpected error on abort: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0 (user declined)", deletes)
	}
}

// TestCloudInitGroupDeleteByData verifies "delete -d <payload>" derives the
// group names to delete from the payload.
func TestCloudInitGroupDeleteByData(t *testing.T) {
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

// TestCloudInitGroupDeleteNoArgsUsage verifies delete with neither -d nor args
// is a usage error.
func TestCloudInitGroupDeleteNoArgsUsage(t *testing.T) {
	res := runOchami(t, "cloud-init", "group", "delete", "--ignore-config",
		"--uri", "http://127.0.0.1:1", "--token", "t", "--no-confirm")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestCloudInitGroupDeleteHTTPError verifies a failing DELETE resolves to
// CodeHTTP via the per-item aggregation.
func TestCloudInitGroupDeleteHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "group", "delete", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "--no-confirm", "compute")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestCloudInitGroupRender verifies the full render path: fetch the group's
// jinja config, fetch node meta-data, and render the template to stdout.
func TestCloudInitGroupRender(t *testing.T) {
	tmpl := "## template: jinja\n#cloud-config\nhostname: {{ ds.meta_data.hostname }}\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "compute.yaml"):
			_, _ = w.Write([]byte(tmpl)) //nolint:errcheck // test response writes are observed by the client
		case strings.HasSuffix(r.URL.Path, "meta-data"):
			_, _ = w.Write([]byte("hostname: node01\n")) //nolint:errcheck // test response writes are observed by the client
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

// TestCloudInitGroupRenderWithExtraVars verifies --extra-vars are merged into
// the render context.
func TestCloudInitGroupRenderWithExtraVars(t *testing.T) {
	tmpl := "## template: jinja\n#cloud-config\nx: {{ myvar }}\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "compute.yaml"):
			_, _ = w.Write([]byte(tmpl)) //nolint:errcheck // test response writes are observed by the client
		case strings.HasSuffix(r.URL.Path, "meta-data"):
			_, _ = w.Write([]byte("hostname: node01\n")) //nolint:errcheck // test response writes are observed by the client
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

// TestCloudInitGroupRenderHTTPError verifies an unsuccessful HTTP response on
// the group-config fetch resolves to CodeHTTP.
func TestCloudInitGroupRenderHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "group", "render", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "compute", "x0c0s0b0n0")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestCloudInitGroupRenderMetadataHTTPError verifies an unsuccessful HTTP
// response on the node meta-data fetch (after a successful config fetch)
// resolves to CodeHTTP.
func TestCloudInitGroupRenderMetadataHTTPError(t *testing.T) {
	tmpl := "## template: jinja\n#cloud-config\nhostname: {{ ds.meta_data.hostname }}\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "compute.yaml"):
			_, _ = w.Write([]byte(tmpl)) //nolint:errcheck // test response writes are observed by the client
		default:
			http.Error(w, "bad", http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "group", "render", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "compute", "x0c0s0b0n0")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestCloudInitGroupGetByIDNetworkError verifies the by-id network-error arm of
// getGroupData (closed port with args) surfaces a non-success exit.
func TestCloudInitGroupGetByIDNetworkError(t *testing.T) {
	srv := ciGroupServer(t, nil, http.StatusOK)
	url := srv.URL
	srv.Close()

	res := runOchami(t, "cloud-init", "group", "get", "raw", "--ignore-config",
		"--uri", url, "--token", "t", "compute")
	if res.err == nil {
		t.Fatal("expected a network error, got nil")
	}
	if res.exitCode == cli.CodeSuccess {
		t.Errorf("exit code = %d, want a non-success code", res.exitCode)
	}
}

// TestCloudInitGroupRenderMalformedExtraVars verifies malformed --extra-vars is
// a payload error.
func TestCloudInitGroupRenderMalformedExtraVars(t *testing.T) {
	tmpl := "## template: jinja\n#cloud-config\nx: {{ ds.meta_data.hostname }}\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "compute.yaml"):
			_, _ = w.Write([]byte(tmpl)) //nolint:errcheck // test response writes are observed by the client
		case strings.HasSuffix(r.URL.Path, "meta-data"):
			_, _ = w.Write([]byte("hostname: node01\n")) //nolint:errcheck // test response writes are observed by the client
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "group", "render", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "--extra-vars", `not json`, "compute", "x0c0s0b0n0")
	if res.err == nil {
		t.Fatal("expected a payload error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (CodePayload)", res.exitCode, cli.CodePayload)
	}
}
