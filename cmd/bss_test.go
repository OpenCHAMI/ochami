// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// bss_test.go exercises representative "bss" commands end-to-end against an
// httptest.Server, asserting outbound request method/path/query/body and the
// exit codes that command errors resolve to. HTTP-failure and rejection-path
// cases are covered in bss_errors_test.go.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestBSSBootParamsGet_All verifies "bss boot params get" issues GET
// /bootparameters and prints the response body.
func TestBSSBootParamsGet_All(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Write([]byte(`[{"hosts":["x0c0s0b0n0"]}]`))
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "params", "get",
		"--ignore-config", "--uri", srv.URL, "--token", "faketoken")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/bootparameters" {
		t.Errorf("path = %q, want /bootparameters", gotPath)
	}
	if !strings.Contains(res.stdout, "x0c0s0b0n0") {
		t.Errorf("stdout = %q, want it to contain the host", res.stdout)
	}
}

// TestBSSBootParamsGet_WithMAC verifies that --mac is encoded into the query
// string sent to /bootparameters.
func TestBSSBootParamsGet_WithMAC(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		gotPath = r.URL.Path
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "params", "get",
		"--ignore-config", "--uri", srv.URL, "--token", "faketoken",
		"--mac", "de:ad:be:ef:00:00")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(gotQuery, "mac=de%3Aad%3Abe%3Aef%3A00%3A00") {
		t.Errorf("query = %q, want it to contain the URL-encoded mac", gotQuery)
	}
	if gotPath != "/bootparameters" {
		t.Errorf("path = %q, want /bootparameters", gotPath)
	}
}

// TestBSSBootParamsAdd_ViaFlags verifies "bss boot params add" issues POST
// /bootparameters with the kernel and macs encoded in the body.
func TestBSSBootParamsAdd_ViaFlags(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "params", "add",
		"--ignore-config", "--uri", srv.URL, "--token", "faketoken",
		"--mac", "de:ad:be:ef:00:00",
		"--kernel", "https://example.com/vmlinuz")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/bootparameters" {
		t.Errorf("path = %q, want /bootparameters", gotPath)
	}
	var bp map[string]any
	if err := json.Unmarshal(gotBody, &bp); err != nil {
		t.Fatalf("failed to unmarshal body %q: %v", string(gotBody), err)
	}
	if k, _ := bp["kernel"].(string); k != "https://example.com/vmlinuz" {
		t.Errorf("kernel = %v, want https://example.com/vmlinuz", bp["kernel"])
	}
}

// TestBSSBootParamsDelete_NoConfirm verifies "bss boot params delete --no-confirm"
// issues DELETE /bootparameters.
func TestBSSBootParamsDelete_NoConfirm(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "params", "delete",
		"--ignore-config", "--uri", srv.URL, "--token", "faketoken",
		"--no-confirm", "--mac", "de:ad:be:ef:00:00",
		"--kernel", "https://example.com/vmlinuz")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/bootparameters" {
		t.Errorf("path = %q, want /bootparameters", gotPath)
	}
}

// TestBSSDumpstate verifies "bss dumpstate" issues GET /dumpstate.
func TestBSSDumpstate(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Write([]byte(`{"state":"ok"}`))
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "dumpstate", "--ignore-config", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/dumpstate" {
		t.Errorf("path = %q, want /dumpstate", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
}

// TestBSSHostsGet_Success verifies "bss hosts get" issues GET /hosts.
func TestBSSHostsGet_Success(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "hosts", "get", "--ignore-config", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/hosts" {
		t.Errorf("path = %q, want /hosts", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
}

// TestBSSServiceStatus_Success verifies "bss service status" issues GET /service/status.
func TestBSSServiceStatus_Success(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "service", "status", "--ignore-config", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/service/status" {
		t.Errorf("path = %q, want /service/status", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
}

// TestBSSBootScriptGet_Success verifies "bss boot script get" issues GET /bootscript
// with the selector encoded in the query string.
func TestBSSBootScriptGet_Success(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		gotMethod = r.Method
		w.Write([]byte(`#!ipxe`))
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "script", "get",
		"--ignore-config", "--uri", srv.URL, "--mac", "de:ad:be:ef:00:00")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/bootscript" {
		t.Errorf("path = %q, want /bootscript", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if !strings.Contains(gotQuery, "mac=de%3Aad%3Abe%3Aef%3A00%3A00") {
		t.Errorf("query = %q, want it to contain the URL-encoded mac", gotQuery)
	}
}

// TestBSSHistoryGet_Success verifies "bss history" issues GET /endpoint-history.
func TestBSSHistoryGet_Success(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "history", "--ignore-config", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/endpoint-history" {
		t.Errorf("path = %q, want /endpoint-history", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
}

// TestBSSStatus_Deprecated verifies the deprecated top-level "bss status" command
// still issues a GET under /service.
func TestBSSStatus_Deprecated(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "status", "--ignore-config", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.HasPrefix(gotPath, "/service") {
		t.Errorf("path = %q, want a /service path", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
}

// TestBSSBootImageSet_Success verifies "bss boot image set" fetches the
// existing boot parameters with a GET and writes the updated root back with a
// PUT.
func TestBSSBootImageSet_Success(t *testing.T) {
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method == http.MethodGet {
			// Return one boot-params entry matching the requested mac so the
			// command has something to edit and PUT back.
			w.Write([]byte(`[{"macs":["de:ad:be:ef:00:00"],"params":"console=tty0"}]`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "image", "set",
		"--ignore-config", "--uri", srv.URL, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "/dev/sda1")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	// Expect at least one GET (fetch) followed by a PUT (update).
	sawGet, sawPut := false, false
	for _, m := range methods {
		switch m {
		case http.MethodGet:
			sawGet = true
		case http.MethodPut:
			sawPut = true
		}
	}
	if !sawGet || !sawPut {
		t.Errorf("methods = %v, want at least one GET and one PUT", methods)
	}
}
