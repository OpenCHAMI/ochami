// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestCloudInitGroupGet_HTTPError verifies an unsuccessful HTTP response on the
// all-groups fetch resolves to CodeHTTP.
func TestCloudInitGroupGet_HTTPError(t *testing.T) {
	srv := ciGroupServer(t, nil, http.StatusInternalServerError)
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "get", "raw",
		"--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestCloudInitGroupGet_ByIDHTTPError verifies an unsuccessful HTTP response on a
// per-group fetch (args form) resolves to CodeHTTP via the per-item error
// aggregation ("completed with errors").
func TestCloudInitGroupGet_ByIDHTTPError(t *testing.T) {
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

// TestCloudInitGroupGet_NetworkError verifies pointing at a closed port resolves
// to CodeNetwork.
func TestCloudInitGroupGet_NetworkError(t *testing.T) {
	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "get", "raw",
		"--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}

// TestCloudInitGroupAdd_MalformedPayload verifies malformed inline payload data
// resolves to CodePayload.
func TestCloudInitGroupAdd_MalformedPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "add",
		"--uri", srv.URL, "--token", "t", "-d", `not json`)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestCloudInitGroupAdd_HTTPError verifies a failing POST resolves to CodeHTTP
// via the per-item aggregation.
func TestCloudInitGroupAdd_HTTPError(t *testing.T) {
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
	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "group", "delete",
		"--uri", "http://127.0.0.1:1", "--token", "t", "--no-confirm")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestCloudInitGroupDelete_HTTPError verifies a failing DELETE resolves to
// CodeHTTP via the per-item aggregation.
func TestCloudInitGroupDelete_HTTPError(t *testing.T) {
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
