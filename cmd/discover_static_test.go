// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// discover_static_test.go exercises the branchy paths of "ochami discover
// static" that the happy-path tests in discover_test.go do not reach: the
// --overwrite 409-then-fallback loops (redfish PUT, ethernet-interface PATCH,
// group PATCH), the discovery-version v1 ethernet-interface path, the
// deprecated discovery format, stdin input, and payload errors.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	discover_static "github.com/openchami/ochami/cmd/discover/static"
	"github.com/openchami/ochami/internal/cli"
)

// TestDiscoverStaticFlagStateIsLocal verifies one command invocation cannot
// change the default discovery version of a subsequently constructed command.
func TestDiscoverStaticFlagStateIsLocal(t *testing.T) {
	first := discover_static.NewCmd()
	if err := first.Flags().Set("discovery-version", "1"); err != nil {
		t.Fatalf("set first discovery version: %v", err)
	}

	second := discover_static.NewCmd()
	if got := second.Flags().Lookup("discovery-version").Value.String(); got != "2" {
		t.Errorf("second command discovery version = %q, want default 2", got)
	}
}

// smdOverwriteServer returns an httptest.Server that emulates SMD's overwrite
// semantics: it returns 409 Conflict for the first POST to redfish, ethernet,
// and group endpoints (so the command falls back to PUT/PATCH), and 200 for the
// subsequent PUT/PATCH. Components always succeed. It records which fallback
// verbs were seen so tests can assert the fallback path was exercised.
type smdOverwriteRecorder struct {
	mu         sync.Mutex
	rfePut     bool
	ifacePatch bool
	groupPatch bool
}

func smdOverwriteServer(t *testing.T, rec *smdOverwriteRecorder) *httptest.Server {
	t.Helper()
	postSeen := map[string]bool{}
	var mu sync.Mutex
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path
		switch {
		case strings.Contains(path, "RedfishEndpoints"):
			if r.Method == http.MethodPost {
				mu.Lock()
				postSeen["rfe"] = true
				mu.Unlock()
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"type":"about:blank","detail":"exists"}`)) //nolint:errcheck // test response writes are observed by the client
				return
			}
			if r.Method == http.MethodPut {
				rec.mu.Lock()
				rec.rfePut = true
				rec.mu.Unlock()
			}
			_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
		case strings.Contains(path, "EthernetInterfaces"):
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"detail":"exists"}`)) //nolint:errcheck // test response writes are observed by the client
				return
			}
			if r.Method == http.MethodPatch {
				rec.mu.Lock()
				rec.ifacePatch = true
				rec.mu.Unlock()
			}
			_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
		case strings.Contains(path, "groups"):
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"detail":"exists"}`)) //nolint:errcheck // test response writes are observed by the client
				return
			}
			if r.Method == http.MethodPatch {
				rec.mu.Lock()
				rec.groupPatch = true
				rec.mu.Unlock()
			}
			_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
		default:
			// Components and everything else succeed.
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
		}
	}))
}

// TestDiscoverStaticOverwriteFallback verifies that with --overwrite and a
// server that returns 409 on POST, the command falls back to PUT (redfish) and
// PATCH (group), exercising the 409-fallback loops.
func TestDiscoverStaticOverwriteFallback(t *testing.T) {
	rec := &smdOverwriteRecorder{}
	srv := smdOverwriteServer(t, rec)
	defer srv.Close()

	res := runOchami(t, "discover", "static", "-d", discoveryPayload, "--overwrite",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if !rec.rfePut {
		t.Error("expected a redfish PUT fallback after 409, got none")
	}
	if !rec.groupPatch {
		t.Error("expected a group PATCH fallback after 409, got none")
	}
}

// TestDiscoverStaticV1Overwrite verifies the discovery-version v1 path with
// --overwrite exercises the ethernet-interface POST->409->PATCH fallback loop.
func TestDiscoverStaticV1Overwrite(t *testing.T) {
	rec := &smdOverwriteRecorder{}
	srv := smdOverwriteServer(t, rec)
	defer srv.Close()

	res := runOchami(t, "discover", "static", "-d", discoveryPayload, "--overwrite",
		"--discovery-version", "1",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if !rec.ifacePatch {
		t.Error("expected an ethernet-interface PATCH fallback after 409, got none")
	}
}

// TestDiscoverStaticV1 verifies the discovery-version v1 path (non-overwrite)
// POSTs ethernet interfaces.
func TestDiscoverStaticV1(t *testing.T) {
	var sawIfacePost bool
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "EthernetInterfaces") {
			mu.Lock()
			sawIfacePost = true
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "discover", "static", "-d", discoveryPayload,
		"--discovery-version", "1",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	mu.Lock()
	defer mu.Unlock()
	if !sawIfacePost {
		t.Error("expected an ethernet-interface POST for discovery-version v1, got none")
	}
}

// TestDiscoverStaticStdin verifies the command reads the discovery payload from
// stdin when -d is not passed.
func TestDiscoverStaticStdin(t *testing.T) {
	var sawPost bool
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			mu.Lock()
			sawPost = true
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, discoveryPayload,
		"discover", "static", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	mu.Lock()
	defer mu.Unlock()
	if !sawPost {
		t.Error("expected a POST from stdin-provided payload, got none")
	}
}

// TestDiscoverStaticDeprecatedFormat verifies the deprecated discovery format
// (detected via the bmc_mac node key) is accepted and populates SMD.
func TestDiscoverStaticDeprecatedFormat(t *testing.T) {
	const deprecatedPayload = `{
  "nodes": [
    {
      "name": "node01",
      "nid": 1,
      "xname": "x1000c1s7b0n0",
      "bmc_mac": "de:ca:fc:0f:ee:ee",
      "bmc_ip": "172.16.0.101",
      "group": "compute",
      "interfaces": [
        {"mac_addr": "de:ad:be:ee:ee:f1", "ip_addrs": [{"name": "internal", "ip_addr": "172.16.0.1"}]}
      ]
    }
  ]
}`
	var sawPost bool
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			mu.Lock()
			sawPost = true
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "discover", "static", "-d", deprecatedPayload,
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	mu.Lock()
	defer mu.Unlock()
	if !sawPost {
		t.Error("expected a POST for deprecated-format discovery, got none")
	}
}

// TestDiscoverStaticMalformedPayload verifies malformed inline payload resolves
// to CodePayload.
func TestDiscoverStaticMalformedPayload(t *testing.T) {
	res := runOchami(t, "discover", "static", "-d", `not json`,
		"--ignore-config", "--uri", "http://127.0.0.1:1", "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (CodePayload)", res.exitCode, cli.CodePayload)
	}
}

// TestDiscoverStaticNoConfig verifies that without a resolvable base URI the
// command fails with CodeConfig.
func TestDiscoverStaticNoConfig(t *testing.T) {
	res := runOchami(t, "discover", "static", "-d", discoveryPayload, "--ignore-config", "--token", "t")
	if res.err == nil {
		t.Fatal("expected a config error, got nil")
	}
	if res.exitCode != cli.CodeConfig {
		t.Errorf("exit code = %d, want %d (CodeConfig)", res.exitCode, cli.CodeConfig)
	}
}

// TestDiscoverStaticOverwriteHTTPError verifies that with --overwrite, a
// non-409 HTTP error on the redfish POST resolves to the CodeHTTP aggregate.
func TestDiscoverStaticOverwriteHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "RedfishEndpoints") && r.Method == http.MethodPost {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "discover", "static", "-d", discoveryPayload, "--overwrite",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestDiscoverStaticOverwritePutFails verifies that with --overwrite, when the
// redfish POST returns 409 but the fallback PUT also fails, the command reports
// the CodeHTTP aggregate.
func TestDiscoverStaticOverwritePutFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "RedfishEndpoints") {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"detail":"exists"}`)) //nolint:errcheck // test response writes are observed by the client
				return
			}
			// PUT fallback also fails.
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "discover", "static", "-d", discoveryPayload, "--overwrite",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestDiscoverStaticV1OverwritePatchFails verifies that with --overwrite and
// discovery-version v1, when the ethernet-interface POST returns 409 but the
// fallback PATCH also fails, the command reports the CodeHTTP aggregate.
func TestDiscoverStaticV1OverwritePatchFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "EthernetInterfaces") {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"detail":"exists"}`)) //nolint:errcheck // test response writes are observed by the client
				return
			}
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "discover", "static", "-d", discoveryPayload, "--overwrite",
		"--discovery-version", "1",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestDiscoverStaticOverwriteGroupPatchFails verifies that with --overwrite,
// when the group POST returns 409 but the fallback PATCH also fails, the
// command reports the CodeHTTP aggregate.
func TestDiscoverStaticOverwriteGroupPatchFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "groups") {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"detail":"exists"}`)) //nolint:errcheck // test response writes are observed by the client
				return
			}
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "discover", "static", "-d", discoveryPayload, "--overwrite",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestDiscoverStaticComponentHTTPError verifies that a failing component POST
// (non-overwrite) surfaces the CodeHTTP aggregate.
func TestDiscoverStaticComponentHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "State/Components") {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "discover", "static", "-d", discoveryPayload,
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestDiscoverStaticOverwriteComponentError verifies that with --overwrite, a
// failing component PUT surfaces the CodeHTTP aggregate.
func TestDiscoverStaticOverwriteComponentError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "State/Components") {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "discover", "static", "-d", discoveryPayload, "--overwrite",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestDiscoverStaticV1IfaceError verifies the discovery-version v1 non-overwrite
// path surfaces an ethernet-interface POST error as the CodeHTTP aggregate.
func TestDiscoverStaticV1IfaceError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "EthernetInterfaces") {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "discover", "static", "-d", discoveryPayload, "--discovery-version", "1",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestDiscoverStaticGroupError verifies a failing group POST (non-overwrite)
// surfaces the CodeHTTP aggregate.
func TestDiscoverStaticGroupError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "groups") {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "discover", "static", "-d", discoveryPayload,
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestDiscoverStaticNetworkError verifies that a transport-level failure (closed
// server) surfaces as an aggregate error, exercising the non-HTTP ("failed to
// add ... to SMD") error-message arms across all sections.
func TestDiscoverStaticNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res := runOchami(t, "discover", "static", "-d", discoveryPayload,
		"--ignore-config", "--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode == cli.CodeSuccess {
		t.Errorf("exit code = %d, want a non-success code", res.exitCode)
	}
}

// TestDiscoverStaticV1NetworkError does the same for the discovery-version v1
// path so the ethernet-interface non-HTTP error arm is exercised.
func TestDiscoverStaticV1NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res := runOchami(t, "discover", "static", "-d", discoveryPayload, "--discovery-version", "1",
		"--ignore-config", "--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode == cli.CodeSuccess {
		t.Errorf("exit code = %d, want a non-success code", res.exitCode)
	}
}

// TestDiscoverStaticOverwriteNetworkError exercises the overwrite path against a
// closed server so the function-level error arms in the overwrite loops fire.
func TestDiscoverStaticOverwriteNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res := runOchami(t, "discover", "static", "-d", discoveryPayload, "--overwrite",
		"--discovery-version", "1",
		"--ignore-config", "--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode == cli.CodeSuccess {
		t.Errorf("exit code = %d, want a non-success code", res.exitCode)
	}
}
