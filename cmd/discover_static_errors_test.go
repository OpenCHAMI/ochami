// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// discover_static_errors_test.go exercises the error-arm branches of "ochami
// discover static": HTTP failures (with and without --overwrite),
// malformed/missing payload and config, 409-then-failing-fallback cases, and
// transport-level (network) failures across the plain, discovery-version v1,
// and --overwrite paths. The discoveryPayload fixture is in
// discover_static_test.go.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestDiscoverStatic_HTTPError verifies that when SMD rejects the discovery
// writes, "discover static" resolves to the CodeHTTP exit code (the "completed
// with errors" aggregate).
func TestDiscoverStatic_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "discover", "static", "-d", discoveryPayload,
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestDiscoverStatic_MalformedPayload verifies malformed inline payload resolves
// to CodePayload.
func TestDiscoverStatic_MalformedPayload(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "discover", "static", "-d", `not json`,
		"--ignore-config", "--uri", "http://127.0.0.1:1", "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestDiscoverStatic_NoConfig verifies that without a resolvable base URI the
// command fails with CodeConfig.
func TestDiscoverStatic_NoConfig(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "discover", "static", "-d", discoveryPayload, "--ignore-config", "--token", "t")
	if res.err == nil {
		t.Fatal("expected a config error, got nil")
	}
	if res.exitCode != cli.CodeConfig {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeConfig, cli.CodeName(cli.CodeConfig))
	}
}

// TestDiscoverStatic_OverwriteHTTPError verifies that with --overwrite, a
// non-409 HTTP error on the redfish POST resolves to the CodeHTTP aggregate.
func TestDiscoverStatic_OverwriteHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "RedfishEndpoints") && r.Method == http.MethodPost {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "discover", "static", "-d", discoveryPayload, "--overwrite",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestDiscoverStatic_OverwritePutFails verifies that with --overwrite, when the
// redfish POST returns 409 but the fallback PUT also fails, the command reports
// the CodeHTTP aggregate.
func TestDiscoverStatic_OverwritePutFails(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "RedfishEndpoints") {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"detail":"exists"}`))
				return
			}
			// PUT fallback also fails.
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "discover", "static", "-d", discoveryPayload, "--overwrite",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestDiscoverStatic_V1OverwritePatchFails verifies that with --overwrite and
// discovery-version v1, when the ethernet-interface POST returns 409 but the
// fallback PATCH also fails, the command reports the CodeHTTP aggregate.
func TestDiscoverStatic_V1OverwritePatchFails(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "EthernetInterfaces") {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"detail":"exists"}`))
				return
			}
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "discover", "static", "-d", discoveryPayload, "--overwrite",
		"--discovery-version", "1",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestDiscoverStatic_OverwriteGroupPatchFails verifies that with --overwrite,
// when the group POST returns 409 but the fallback PATCH also fails, the
// command reports the CodeHTTP aggregate.
func TestDiscoverStatic_OverwriteGroupPatchFails(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "groups") {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"detail":"exists"}`))
				return
			}
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "discover", "static", "-d", discoveryPayload, "--overwrite",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestDiscoverStatic_ComponentHTTPError verifies that a failing component POST
// (non-overwrite) surfaces the CodeHTTP aggregate.
func TestDiscoverStatic_ComponentHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "State/Components") {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "discover", "static", "-d", discoveryPayload,
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestDiscoverStatic_OverwriteComponentError verifies that with --overwrite, a
// failing component PUT surfaces the CodeHTTP aggregate.
func TestDiscoverStatic_OverwriteComponentError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "State/Components") {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "discover", "static", "-d", discoveryPayload, "--overwrite",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestDiscoverStatic_V1IfaceError verifies the discovery-version v1 non-overwrite
// path surfaces an ethernet-interface POST error as the CodeHTTP aggregate.
func TestDiscoverStatic_V1IfaceError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "EthernetInterfaces") {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "discover", "static", "-d", discoveryPayload, "--discovery-version", "1",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestDiscoverStatic_GroupError verifies a failing group POST (non-overwrite)
// surfaces the CodeHTTP aggregate.
func TestDiscoverStatic_GroupError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "groups") {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "discover", "static", "-d", discoveryPayload,
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestDiscoverStatic_NetworkError verifies that "discover static" fails with
// CodeNetwork when SMD can't be reached.
func TestDiscoverStatic_NetworkError(t *testing.T) {
	t.Parallel()

	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchamiWithRuntime(t, "discover", "static", "-d", discoveryPayload,
		"--ignore-config", "--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}

// TestDiscoverStatic_V1NetworkError verifies that "discover static
// --discovery-version 1" fails with CodeNetwork when SMD can't be reached.
func TestDiscoverStatic_V1NetworkError(t *testing.T) {
	t.Parallel()

	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchamiWithRuntime(t, "discover", "static", "-d", discoveryPayload, "--discovery-version", "1",
		"--ignore-config", "--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}

// TestDiscoverStatic_OverwriteNetworkError verifies that "discover static
// --overwrite --discovery-version 1" fails with CodeNetwork when SMD can't be
// reached.
func TestDiscoverStatic_OverwriteNetworkError(t *testing.T) {
	t.Parallel()

	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchamiWithRuntime(t, "discover", "static", "-d", discoveryPayload, "--overwrite",
		"--discovery-version", "1",
		"--ignore-config", "--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}
