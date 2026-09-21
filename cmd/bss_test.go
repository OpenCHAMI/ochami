// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
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
	"net/url"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestBSSBootParamsGet_All verifies "bss boot params get" issues GET
// /bootparameters and prints the response body.
func TestBSSBootParamsGet_All(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[{"hosts":["x0c0s0b0n0"]}]`))
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
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`[]`))
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
}

// TestBSSBootParamsAdd_ViaFlags verifies "bss boot params add" issues POST
// /bootparameters with the kernel and macs encoded in the body.
func TestBSSBootParamsAdd_ViaFlags(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
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
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"state":"ok"}`))
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "dumpstate", "--ignore-config", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/dumpstate" {
		t.Errorf("path = %q, want /dumpstate", gotPath)
	}
}

// TestBSSHostsGet_Success verifies "bss hosts get" issues GET /hosts.
func TestBSSHostsGet_Success(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "hosts", "get", "--ignore-config", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/hosts" {
		t.Errorf("path = %q, want /hosts", gotPath)
	}
}

// TestBSSServiceStatus_Success verifies "bss service status" issues GET /service/status.
func TestBSSServiceStatus_Success(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "service", "status", "--ignore-config", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/service/status" {
		t.Errorf("path = %q, want /service/status", gotPath)
	}
}

// TestBSSBootScriptGet_Success verifies "bss boot script get" issues GET /bootscript
// with the selector encoded in the query string.
func TestBSSBootScriptGet_Success(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		_, _ = w.Write([]byte(`#!ipxe`))
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
	if !strings.Contains(gotQuery, "mac=de%3Aad%3Abe%3Aef%3A00%3A00") {
		t.Errorf("query = %q, want it to contain the URL-encoded mac", gotQuery)
	}
}

// TestBSSHistoryGet_Success verifies "bss history" issues GET /endpoint-history.
func TestBSSHistoryGet_Success(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "history", "--ignore-config", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/endpoint-history" {
		t.Errorf("path = %q, want /endpoint-history", gotPath)
	}
}

// TestBSSStatusDeprecated verifies the deprecated top-level "bss status" command
// still issues a GET under /service.
func TestBSSStatusDeprecated(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "status", "--ignore-config", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.HasPrefix(gotPath, "/service") {
		t.Errorf("path = %q, want a /service path", gotPath)
	}
}

// TestBSSBootImageSet_Success verifies "bss boot image set" fetches existing boot
// parameters (GET /bootparameters) and writes the updated root back
// (PUT /bootparameters).
func TestBSSBootImageSet_Success(t *testing.T) {
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method == http.MethodGet {
			// Return one boot-params entry matching the requested mac so the
			// command has something to edit and PUT back.
			_, _ = w.Write([]byte(`[{"macs":["de:ad:be:ef:00:00"],"params":"console=tty0"}]`))
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

// TestBSSBootParamsGet_Query verifies the query builder emits name/mac/nid query
// parameters for the corresponding flags.
func TestBSSBootParamsGet_Query(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantKey string
		wantVal string
	}{
		{"xname", []string{"--xname", "x0c0s0b0n0"}, "name", "x0c0s0b0n0"},
		{"mac", []string{"--mac", "de:ad:be:ef:00:00"}, "mac", "de:ad:be:ef:00:00"},
		{"nid", []string{"--nid", "42"}, "nid", "42"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotQuery url.Values
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotQuery = r.URL.Query()
				_, _ = w.Write([]byte(`[]`))
			}))
			defer srv.Close()

			args := append([]string{"bss", "boot", "params", "get",
				"--ignore-config", "--uri", srv.URL, "--token", "t"}, tc.args...)
			res := runOchami(t, args...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if got := gotQuery.Get(tc.wantKey); got != tc.wantVal {
				t.Errorf("query %s = %q, want %q", tc.wantKey, got, tc.wantVal)
			}
		})
	}
}

// TestBSSBootParamsGet_Formats verifies the output-format variants.
func TestBSSBootParamsGet_Formats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"macs":["de:ad:be:ef:00:00"],"params":"console=tty0"}]`))
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchami(t, "bss", "boot", "params", "get",
			"--ignore-config", "--uri", srv.URL, "--token", "t", "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
		if !strings.Contains(res.stdout, "console=tty0") {
			t.Errorf("format %s: stdout = %q, want it to contain params", f, res.stdout)
		}
	}
}

// TestBSSBootParamsGet_AllHTTPError verifies an unsuccessful HTTP response resolves
// to CodeHTTP.
func TestBSSBootParamsGet_AllHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "params", "get",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestBSSBootParamsGet_AllNetworkError verifies pointing at a closed port resolves
// to CodeNetwork.
func TestBSSBootParamsGet_AllNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res := runOchami(t, "bss", "boot", "params", "get",
		"--ignore-config", "--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestBSSBootParamsAdd_DataAndFlags verifies "add -d <payload>" merged with
// component flags issues a POST (payload read first, then flags applied).
func TestBSSBootParamsAdd_DataAndFlags(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "params", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"-d", `{"kernel":"https://example.com/vmlinuz"}`, "--mac", "de:ad:be:ef:00:00")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
}

// TestBSSBootParamsAdd_MalformedPayload verifies malformed inline payload
// resolves to CodePayload.
func TestBSSBootParamsAdd_MalformedPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "params", "add",
		"--ignore-config", "--uri", srv.URL, "--token", "t", "-d", `not json`)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (CodePayload)", res.exitCode, cli.CodePayload)
	}
}

// TestBSSBootParamsUpdate_HTTPError verifies a failing PATCH resolves to
// CodeHTTP.
func TestBSSBootParamsUpdate_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "params", "update",
		"--ignore-config", "--uri", srv.URL, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "https://example.com/vmlinuz")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestBSSBootParamsAdd_AllFlags verifies "add" applies every component selector
// and config-field flag arm.
func TestBSSBootParamsAdd_AllFlags(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "params", "add",
		"--ignore-config", "--uri", srv.URL, "--token", "t",
		"--xname", "x0c0s0b0n0", "--mac", "de:ad:be:ef:00:00", "--nid", "1",
		"--kernel", "https://example.com/vmlinuz", "--initrd", "https://example.com/initrd",
		"--params", "console=tty0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
}

// TestBSSBootParamsDelete_AllFlags verifies "delete --no-confirm" applies every
// selector and config-field flag arm.
func TestBSSBootParamsDelete_AllFlags(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "params", "delete",
		"--ignore-config", "--uri", srv.URL, "--token", "t", "--no-confirm",
		"--xname", "x0c0s0b0n0", "--mac", "de:ad:be:ef:00:00", "--nid", "1",
		"--kernel", "https://example.com/vmlinuz", "--initrd", "https://example.com/initrd",
		"--params", "console=tty0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}

// TestBSSBootParamsSet_AllSelectorFlags verifies "set" applies every component
// selector (--xname/--mac/--nid) and every config field (--kernel/--initrd/
// --params) flag arm.
func TestBSSBootParamsSet_AllSelectorFlags(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "params", "set",
		"--ignore-config", "--uri", srv.URL, "--token", "t",
		"--xname", "x0c0s0b0n0", "--mac", "de:ad:be:ef:00:00", "--nid", "1",
		"--kernel", "https://example.com/vmlinuz", "--initrd", "https://example.com/initrd",
		"--params", "console=tty0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
}

// TestBSSBootParamsUpdate_AllSelectorFlags verifies "update" applies every
// selector and config-field flag arm.
func TestBSSBootParamsUpdate_AllSelectorFlags(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "params", "update",
		"--ignore-config", "--uri", srv.URL, "--token", "t",
		"--xname", "x0c0s0b0n0", "--nid", "1",
		"--kernel", "https://example.com/vmlinuz", "--initrd", "https://example.com/initrd",
		"--params", "console=tty0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}
}

// TestBSSBootParamsSet_DataWithFlagOverride verifies "set -d <payload>" merged
// with flags (which override payload fields) issues a PUT.
func TestBSSBootParamsSet_DataWithFlagOverride(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "params", "set",
		"--ignore-config", "--uri", srv.URL, "--token", "t",
		"-d", `{"macs":["de:ad:be:ef:00:00"],"kernel":"http://old/vmlinuz"}`,
		"--kernel", "https://example.com/new-vmlinuz")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
}

// TestBSSBootParamsDelete_ByFlags verifies "delete --no-confirm" with component
// and config flags issues a DELETE.
func TestBSSBootParamsDelete_ByFlags(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "params", "delete",
		"--ignore-config", "--uri", srv.URL, "--token", "t", "--no-confirm",
		"--xname", "x0c0s0b0n0", "--nid", "1", "--kernel", "https://example.com/vmlinuz",
		"--initrd", "https://example.com/initrd", "--params", "quiet")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}

// TestBSSBootParamsDelete_ByData verifies "delete -d <payload>" issues a DELETE.
func TestBSSBootParamsDelete_ByData(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "params", "delete",
		"--ignore-config", "--uri", srv.URL, "--token", "t", "--no-confirm",
		"-d", `{"macs":["de:ad:be:ef:00:00"]}`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}

// TestBSSBootParamsDelete_MissingSelector verifies delete without -d and without
// a component selector is a usage error.
func TestBSSBootParamsDelete_MissingSelector(t *testing.T) {
	res := runOchami(t, "bss", "boot", "params", "delete",
		"--ignore-config", "--uri", "http://127.0.0.1:1", "--token", "t", "--no-confirm")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestBSSBootParamsDelete_MissingConfig verifies delete with a component selector
// but no config selector is a usage error.
func TestBSSBootParamsDelete_MissingConfig(t *testing.T) {
	res := runOchami(t, "bss", "boot", "params", "delete",
		"--ignore-config", "--uri", "http://127.0.0.1:1", "--token", "t", "--no-confirm",
		"--mac", "de:ad:be:ef:00:00")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestBSSBootParamsDelete_Confirm verifies the interactive confirm path issues
// the DELETE when the user answers "y".
func TestBSSBootParamsDelete_Confirm(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "y\n",
		"bss", "boot", "params", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "https://example.com/vmlinuz")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 1 {
		t.Errorf("DELETE count = %d, want 1", deletes)
	}
}

// TestBSSBootParamsDelete_HTTPError verifies a failing DELETE resolves to
// CodeHTTP.
func TestBSSBootParamsDelete_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "params", "delete",
		"--ignore-config", "--uri", srv.URL, "--token", "t", "--no-confirm",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "https://example.com/vmlinuz")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestBSSBootParamsUpdate_DataWithFlags verifies "update -d <payload>" combined
// with CLI flags (which the command warns are ignored) still issues a PATCH.
func TestBSSBootParamsUpdate_DataWithFlags(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "params", "update", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"-d", `{"macs":["de:ad:be:ef:00:00"],"kernel":"http://k"}`, "--mac", "de:ad:be:ef:00:01")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}
}

// TestBSSBootParamsUpdate_MissingSelector verifies update without -d and without
// a component selector is a usage error.
func TestBSSBootParamsUpdate_MissingSelector(t *testing.T) {
	res := runOchami(t, "bss", "boot", "params", "update", "--ignore-config",
		"--uri", "http://127.0.0.1:1", "--token", "t")
	if res.err == nil || res.exitCode != cli.CodeUsage {
		t.Errorf("err=%v exit=%d, want CodeUsage", res.err, res.exitCode)
	}
}

// TestBSSBootParamsUpdate_MissingConfig verifies update with a component selector
// but no config selector is a usage error.
func TestBSSBootParamsUpdate_MissingConfig(t *testing.T) {
	res := runOchami(t, "bss", "boot", "params", "update", "--ignore-config",
		"--uri", "http://127.0.0.1:1", "--token", "t", "--mac", "de:ad:be:ef:00:00")
	if res.err == nil || res.exitCode != cli.CodeUsage {
		t.Errorf("err=%v exit=%d, want CodeUsage", res.err, res.exitCode)
	}
}

// TestBSSBootParamsSet_InvalidMac verifies an invalid MAC address is a usage
// error for "set".
func TestBSSBootParamsSet_InvalidMac(t *testing.T) {
	res := runOchami(t, "bss", "boot", "params", "set", "--ignore-config",
		"--uri", "http://127.0.0.1:1", "--token", "t", "--mac", "not-a-mac", "--kernel", "http://k")
	if res.err == nil || res.exitCode != cli.CodeUsage {
		t.Errorf("err=%v exit=%d, want CodeUsage", res.err, res.exitCode)
	}
}

// TestBSSBootParamsUpdate_NetworkError verifies a closed port surfaces
// CodeNetwork for update.
func TestBSSBootParamsUpdate_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res := runOchami(t, "bss", "boot", "params", "update", "--ignore-config", "--uri", url, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "http://k")
	if res.err == nil || res.exitCode != cli.CodeNetwork {
		t.Errorf("err=%v exit=%d, want CodeNetwork", res.err, res.exitCode)
	}
}

// TestBSSBootParamsAdd_InvalidMac verifies an invalid MAC is a usage error for
// "add".
func TestBSSBootParamsAdd_InvalidMac(t *testing.T) {
	res := runOchami(t, "bss", "boot", "params", "add", "--ignore-config",
		"--uri", "http://127.0.0.1:1", "--token", "t", "--mac", "not-a-mac", "--kernel", "http://k")
	if res.err == nil || res.exitCode != cli.CodeUsage {
		t.Errorf("err=%v exit=%d, want CodeUsage", res.err, res.exitCode)
	}
}

// TestBSSBootParamsAdd_NetworkError verifies a closed port surfaces CodeNetwork
// for "add".
func TestBSSBootParamsAdd_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res := runOchami(t, "bss", "boot", "params", "add", "--ignore-config", "--uri", url, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "http://k")
	if res.err == nil || res.exitCode != cli.CodeNetwork {
		t.Errorf("err=%v exit=%d, want CodeNetwork", res.err, res.exitCode)
	}
}

// TestBSSBootParamsSet_NetworkError verifies a closed port surfaces CodeNetwork
// for "set".
func TestBSSBootParamsSet_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res := runOchami(t, "bss", "boot", "params", "set", "--ignore-config", "--uri", url, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "http://k")
	if res.err == nil || res.exitCode != cli.CodeNetwork {
		t.Errorf("err=%v exit=%d, want CodeNetwork", res.err, res.exitCode)
	}
}

// TestBSSBootParamsDelete_NetworkError verifies a closed port surfaces
// CodeNetwork for "delete".
func TestBSSBootParamsDelete_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res := runOchami(t, "bss", "boot", "params", "delete", "--ignore-config", "--uri", url, "--token", "t",
		"--no-confirm", "--mac", "de:ad:be:ef:00:00", "--kernel", "http://k")
	if res.err == nil || res.exitCode != cli.CodeNetwork {
		t.Errorf("err=%v exit=%d, want CodeNetwork", res.err, res.exitCode)
	}
}
