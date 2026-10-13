// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// discover_static_test.go exercises "ochami discover static", which reads a
// discovery payload and populates SMD by POSTing components, redfish endpoints,
// and groups, against an httptest.Server standing in for SMD: the default path,
// the --overwrite 409-then-fallback loops (redfish PUT, ethernet-interface
// PATCH, group PATCH), the discovery-version v1 ethernet-interface path, stdin
// input, and the deprecated discovery format. Error-arm cases live in
// discover_static_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// discoveryPayload is a minimal valid static-discovery payload with one BMC and
// one node (member of one group and with one ethernet interface).
const discoveryPayload = `{
  "bmcs": [
    {"xname": "x1000c1s7b0", "mac": "de:ca:fc:0f:ee:ee", "ip": "172.16.0.101"}
  ],
  "nodes": [
    {
      "name": "node01",
      "nid": 1,
      "xname": "x1000c1s7b0n0",
      "bmc": "x1000c1s7b0",
      "groups": ["compute"],
      "interfaces": [
        {"mac_addr": "de:ad:be:ee:ee:f1", "ip_addrs": [{"name": "internal", "ip_addr": "172.16.0.1"}]}
      ]
    }
  ]
}`

// TestDiscoverStatic_Success verifies "discover static -d <payload>" populates SMD,
// issuing POST requests for the discovered structures, and exits successfully
// when the server accepts them.
func TestDiscoverStatic_Success(t *testing.T) {

	sawPost := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			sawPost = true
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "discover", "static", "-d", discoveryPayload,
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !sawPost {
		t.Error("expected at least one POST to SMD, got none")
	}
}

// TestDiscoverStatic_Overwrite verifies that "discover static --overwrite"
// succeeds against a service that accepts every request.
func TestDiscoverStatic_Overwrite(t *testing.T) {

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "discover", "static", "-d", discoveryPayload, "--overwrite",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if res.exitCode != cli.CodeSuccess {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeSuccess, cli.CodeName(cli.CodeSuccess))
	}
}

// smdOverwriteRecorder records which overwrite fallbacks smdOverwriteServer
// served.
type smdOverwriteRecorder struct {
	mu         sync.Mutex
	rfePut     bool
	ifacePatch bool
	groupPatch bool
}

// smdOverwriteServer returns an httptest.Server that emulates SMD's overwrite
// semantics: it returns 409 Conflict for the first POST to redfish, ethernet,
// and group endpoints (so the command falls back to PUT/PATCH), and 200 for the
// subsequent PUT/PATCH. Components always succeed. It records which fallback
// verbs were seen so tests can assert the fallback path was exercised.
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
				w.Write([]byte(`{"type":"about:blank","detail":"exists"}`))
				return
			}
			if r.Method == http.MethodPut {
				rec.mu.Lock()
				rec.rfePut = true
				rec.mu.Unlock()
			}
			w.Write([]byte(`[]`))
		case strings.Contains(path, "EthernetInterfaces"):
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"detail":"exists"}`))
				return
			}
			if r.Method == http.MethodPatch {
				rec.mu.Lock()
				rec.ifacePatch = true
				rec.mu.Unlock()
			}
			w.Write([]byte(`[]`))
		case strings.Contains(path, "groups"):
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte(`{"detail":"exists"}`))
				return
			}
			if r.Method == http.MethodPatch {
				rec.mu.Lock()
				rec.groupPatch = true
				rec.mu.Unlock()
			}
			w.Write([]byte(`[]`))
		default:
			// Components and everything else succeed.
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
		}
	}))
}

// TestDiscoverStatic_OverwriteFallback verifies that, with --overwrite,
// "discover static" falls back to a PUT for a Redfish endpoint and a PATCH for
// a group when SMD answers their POSTs with 409 Conflict.
func TestDiscoverStatic_OverwriteFallback(t *testing.T) {
	rec := &smdOverwriteRecorder{}
	srv := smdOverwriteServer(t, rec)
	defer srv.Close()

	res := runOchamiWithRuntime(t, "discover", "static", "-d", discoveryPayload, "--overwrite",
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

// TestDiscoverStatic_V1Overwrite verifies that, with --overwrite and
// --discovery-version 1, "discover static" falls back to a PATCH for an
// ethernet interface when SMD answers its POST with 409 Conflict.
func TestDiscoverStatic_V1Overwrite(t *testing.T) {
	rec := &smdOverwriteRecorder{}
	srv := smdOverwriteServer(t, rec)
	defer srv.Close()

	res := runOchamiWithRuntime(t, "discover", "static", "-d", discoveryPayload, "--overwrite",
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

// TestDiscoverStatic_V1 verifies the discovery-version v1 path (non-overwrite)
// POSTs ethernet interfaces.
func TestDiscoverStatic_V1(t *testing.T) {
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
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "discover", "static", "-d", discoveryPayload,
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

// TestDiscoverStatic_Stdin verifies the command reads the discovery payload from
// stdin when -d is not passed.
func TestDiscoverStatic_Stdin(t *testing.T) {
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
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithInputAndRuntime(t, discoveryPayload,
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

// TestDiscoverStatic_DeprecatedFormat verifies that "discover static" accepts
// the deprecated discovery format (detected by the bmc_mac node key) and sends
// its data to SMD.
func TestDiscoverStatic_DeprecatedFormat(t *testing.T) {
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
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "discover", "static", "-d", deprecatedPayload,
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
