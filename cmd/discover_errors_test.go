// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// discover_errors_test.go exercises the error-arm branches of "ochami
// discover static": HTTP failures (with and without --overwrite),
// malformed/missing payload and config, 409-then-failing-fallback cases, and
// transport-level (network) failures across the plain, discovery-version v1,
// and --overwrite paths. The discoveryPayload fixture is in discover_test.go.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
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

// TestDiscoverStatic_ExtraArgs verifies that "discover static" rejects
// positional arguments as a usage error and sends no request, whether or not
// -d is passed.
func TestDiscoverStatic_ExtraArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{"with -d", []string{"-d", discoveryPayload, "extra-arg"}},
		{"without -d", []string{"nodes.yaml"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var requests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			args := append([]string{"discover", "static", "--ignore-config", "--uri", srv.URL, "--token", "t"}, tt.args...)
			res := runOchamiWithInputAndRuntime(t, discoveryPayload, args...)
			if res.err == nil || res.exitCode != cli.CodeUsage {
				t.Errorf("err=%v exit=%d, want %d (%s)", res.err, res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
			}
			if n := requests.Load(); n != 0 {
				t.Errorf("requests = %d, want none for positional arguments", n)
			}
		})
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

// TestDiscoverStatic_RedfishEndpointError verifies a failing redfish endpoint
// POST (non-overwrite) surfaces the CodeHTTP aggregate.
func TestDiscoverStatic_RedfishEndpointError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "RedfishEndpoints") {
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

// TestDiscoverStatic_OverwriteRejectsInvalidIDs verifies that with --overwrite,
// where IDs end up in request paths, one invalid ID fails discovery with
// CodeUsage before any request is sent, while the same payload without
// --overwrite only sends POSTs and succeeds.
func TestDiscoverStatic_OverwriteRejectsInvalidIDs(t *testing.T) {
	t.Parallel()

	payload := strings.Replace(discoveryPayload, `["compute"]`, `["compute", ".."]`, 1)

	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "discover", "static", "-d", payload, "--overwrite",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.exitCode != cli.CodeUsage {
		t.Errorf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
	if want := `group label ".." is not a single path segment`; res.err == nil || !strings.Contains(res.err.Error(), want) {
		t.Errorf("error = %v, want it to contain %q", res.err, want)
	}
	if got := requests.Load(); got != 0 {
		t.Errorf("request count = %d, want 0", got)
	}

	res = runOchamiWithRuntime(t, "discover", "static", "-d", payload,
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Errorf("without --overwrite: error = %v, want nil", res.err)
	}
}

// TestDiscoverStatic_IfaceIDsCheckedOnlyWhenInPath verifies that an interface is
// checked by the ID adapted from its MAC address, and only in the one mode that
// puts that ID in a request path: --overwrite with discovery version 1. Version
// 2 sends interfaces inside redfish endpoints, and without --overwrite every
// request is a POST with its IDs in the body, so neither rejects the payload.
func TestDiscoverStatic_IfaceIDsCheckedOnlyWhenInPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		mac          string
		args         []string
		wantCode     int
		wantMsg      string
		wantOnlyPOST bool
	}{
		{name: "v1 overwrite with blank adapted ID", mac: "::", args: []string{"--overwrite", "--discovery-version", "1"}, wantCode: cli.CodeUsage, wantMsg: "ethernet interface ID cannot be blank"},
		{name: "v1 overwrite with path-like ID", mac: "a/b", args: []string{"--overwrite", "--discovery-version", "1"}, wantCode: cli.CodeUsage, wantMsg: `ethernet interface ID "a/b" is not a single path segment`},
		{name: "v1 without overwrite", mac: "::", args: []string{"--discovery-version", "1"}, wantCode: cli.CodeSuccess, wantOnlyPOST: true},
		{name: "v2 overwrite", mac: "::", args: []string{"--overwrite", "--discovery-version", "2"}, wantCode: cli.CodeSuccess},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var (
				mu       sync.Mutex
				requests []string
			)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				requests = append(requests, r.Method+" "+r.URL.Path)
				mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[]`))
			}))
			defer srv.Close()

			payload := strings.Replace(discoveryPayload, "de:ad:be:ee:ee:f1", tc.mac, 1)
			args := append([]string{"discover", "static", "-d", payload, "--ignore-config", "--uri", srv.URL, "--token", "t"}, tc.args...)
			res := runOchamiWithRuntime(t, args...)
			if res.exitCode != tc.wantCode {
				t.Errorf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, tc.wantCode, cli.CodeName(tc.wantCode))
			}

			mu.Lock()
			defer mu.Unlock()
			if tc.wantMsg != "" {
				if res.err == nil || !strings.Contains(res.err.Error(), tc.wantMsg) {
					t.Errorf("error = %v, want it to contain %q", res.err, tc.wantMsg)
				}
				if len(requests) != 0 {
					t.Errorf("requests = %v, want none", requests)
				}
				return
			}
			if len(requests) == 0 {
				t.Error("no requests sent, want discovery to proceed")
			}
			for _, req := range requests {
				if strings.Contains(req, "EthernetInterfaces/") {
					t.Errorf("request %q put an interface ID in its path", req)
				}
				if tc.wantOnlyPOST && !strings.HasPrefix(req, http.MethodPost+" ") {
					t.Errorf("request %q is not a POST", req)
				}
			}
		})
	}
}
