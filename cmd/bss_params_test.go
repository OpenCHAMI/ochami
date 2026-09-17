// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// bss_params_test.go covers the branch families of the "bss boot params" verbs
// (get, add, set, update, delete) that the happy paths do not reach: the
// query-building arms (--xname/--mac/--nid), output-format variants, HTTP and
// network error mapping, payload input variants (inline, stdin, malformed), and
// the delete flag/payload/confirmation branches.

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestBSSBootParamsGetQuery verifies the query builder emits name/mac/nid query
// parameters for the corresponding flags.
func TestBSSBootParamsGetQuery(t *testing.T) {
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

// TestBSSBootParamsGetFormats verifies the output-format variants.
func TestBSSBootParamsGetFormats(t *testing.T) {
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

// TestBSSBootParamsGetHTTPError verifies an unsuccessful HTTP response resolves
// to CodeHTTP.
func TestBSSBootParamsGetHTTPError(t *testing.T) {
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

// TestBSSBootParamsGetNetworkError verifies pointing at a closed port resolves
// to CodeNetwork.
func TestBSSBootParamsGetNetworkError(t *testing.T) {
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

// TestBSSBootParamsAddDataAndFlags verifies "add -d <payload>" merged with
// component flags issues a POST (payload read first, then flags applied).
func TestBSSBootParamsAddDataAndFlags(t *testing.T) {
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

// TestBSSBootParamsAddMalformedPayload verifies malformed inline payload
// resolves to CodePayload.
func TestBSSBootParamsAddMalformedPayload(t *testing.T) {
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

// TestBSSBootParamsUpdateHTTPError verifies a failing PATCH resolves to
// CodeHTTP.
func TestBSSBootParamsUpdateHTTPError(t *testing.T) {
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

// TestBSSBootParamsAddAllFlags verifies "add" applies every component selector
// and config-field flag arm.
func TestBSSBootParamsAddAllFlags(t *testing.T) {
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

// TestBSSBootParamsDeleteAllFlags verifies "delete --no-confirm" applies every
// selector and config-field flag arm.
func TestBSSBootParamsDeleteAllFlags(t *testing.T) {
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

// TestBSSBootParamsSetAllSelectorFlags verifies "set" applies every component
// selector (--xname/--mac/--nid) and every config field (--kernel/--initrd/
// --params) flag arm.
func TestBSSBootParamsSetAllSelectorFlags(t *testing.T) {
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

// TestBSSBootParamsUpdateAllSelectorFlags verifies "update" applies every
// selector and config-field flag arm.
func TestBSSBootParamsUpdateAllSelectorFlags(t *testing.T) {
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

// TestBSSBootParamsSetDataWithFlagOverride verifies "set -d <payload>" merged
// with flags (which override payload fields) issues a PUT.
func TestBSSBootParamsSetDataWithFlagOverride(t *testing.T) {
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

// TestBSSBootParamsDeleteByFlags verifies "delete --no-confirm" with component
// and config flags issues a DELETE.
func TestBSSBootParamsDeleteByFlags(t *testing.T) {
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

// TestBSSBootParamsDeleteByData verifies "delete -d <payload>" issues a DELETE.
func TestBSSBootParamsDeleteByData(t *testing.T) {
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

// TestBSSBootParamsDeleteMissingSelector verifies delete without -d and without
// a component selector is a usage error.
func TestBSSBootParamsDeleteMissingSelector(t *testing.T) {
	res := runOchami(t, "bss", "boot", "params", "delete",
		"--ignore-config", "--uri", "http://127.0.0.1:1", "--token", "t", "--no-confirm")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestBSSBootParamsDeleteMissingConfig verifies delete with a component selector
// but no config selector is a usage error.
func TestBSSBootParamsDeleteMissingConfig(t *testing.T) {
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

// TestBSSBootParamsDeleteConfirm verifies the interactive confirm path issues
// the DELETE when the user answers "y".
func TestBSSBootParamsDeleteConfirm(t *testing.T) {
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

// TestBSSBootParamsDeleteHTTPError verifies a failing DELETE resolves to
// CodeHTTP.
func TestBSSBootParamsDeleteHTTPError(t *testing.T) {
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

// TestBSSBootParamsUpdateDataWithFlags verifies "update -d <payload>" combined
// with CLI flags (which the command warns are ignored) still issues a PATCH.
func TestBSSBootParamsUpdateDataWithFlags(t *testing.T) {
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

// TestBSSBootParamsUpdateMissingSelector verifies update without -d and without
// a component selector is a usage error.
func TestBSSBootParamsUpdateMissingSelector(t *testing.T) {
	res := runOchami(t, "bss", "boot", "params", "update", "--ignore-config",
		"--uri", "http://127.0.0.1:1", "--token", "t")
	if res.err == nil || res.exitCode != cli.CodeUsage {
		t.Errorf("err=%v exit=%d, want CodeUsage", res.err, res.exitCode)
	}
}

// TestBSSBootParamsUpdateMissingConfig verifies update with a component selector
// but no config selector is a usage error.
func TestBSSBootParamsUpdateMissingConfig(t *testing.T) {
	res := runOchami(t, "bss", "boot", "params", "update", "--ignore-config",
		"--uri", "http://127.0.0.1:1", "--token", "t", "--mac", "de:ad:be:ef:00:00")
	if res.err == nil || res.exitCode != cli.CodeUsage {
		t.Errorf("err=%v exit=%d, want CodeUsage", res.err, res.exitCode)
	}
}

// TestBSSBootParamsSetInvalidMac verifies an invalid MAC address is a usage
// error for "set".
func TestBSSBootParamsSetInvalidMac(t *testing.T) {
	res := runOchami(t, "bss", "boot", "params", "set", "--ignore-config",
		"--uri", "http://127.0.0.1:1", "--token", "t", "--mac", "not-a-mac", "--kernel", "http://k")
	if res.err == nil || res.exitCode != cli.CodeUsage {
		t.Errorf("err=%v exit=%d, want CodeUsage", res.err, res.exitCode)
	}
}

// TestBSSBootParamsUpdateNetworkError verifies a closed port surfaces
// CodeNetwork for update.
func TestBSSBootParamsUpdateNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res := runOchami(t, "bss", "boot", "params", "update", "--ignore-config", "--uri", url, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "http://k")
	if res.err == nil || res.exitCode != cli.CodeNetwork {
		t.Errorf("err=%v exit=%d, want CodeNetwork", res.err, res.exitCode)
	}
}

// TestBSSBootParamsAddInvalidMac verifies an invalid MAC is a usage error for
// "add".
func TestBSSBootParamsAddInvalidMac(t *testing.T) {
	res := runOchami(t, "bss", "boot", "params", "add", "--ignore-config",
		"--uri", "http://127.0.0.1:1", "--token", "t", "--mac", "not-a-mac", "--kernel", "http://k")
	if res.err == nil || res.exitCode != cli.CodeUsage {
		t.Errorf("err=%v exit=%d, want CodeUsage", res.err, res.exitCode)
	}
}

// TestBSSBootParamsAddNetworkError verifies a closed port surfaces CodeNetwork
// for "add".
func TestBSSBootParamsAddNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res := runOchami(t, "bss", "boot", "params", "add", "--ignore-config", "--uri", url, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "http://k")
	if res.err == nil || res.exitCode != cli.CodeNetwork {
		t.Errorf("err=%v exit=%d, want CodeNetwork", res.err, res.exitCode)
	}
}

// TestBSSBootParamsSetNetworkError verifies a closed port surfaces CodeNetwork
// for "set".
func TestBSSBootParamsSetNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res := runOchami(t, "bss", "boot", "params", "set", "--ignore-config", "--uri", url, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "http://k")
	if res.err == nil || res.exitCode != cli.CodeNetwork {
		t.Errorf("err=%v exit=%d, want CodeNetwork", res.err, res.exitCode)
	}
}

// TestBSSBootParamsDeleteNetworkError verifies a closed port surfaces
// CodeNetwork for "delete".
func TestBSSBootParamsDeleteNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res := runOchami(t, "bss", "boot", "params", "delete", "--ignore-config", "--uri", url, "--token", "t",
		"--no-confirm", "--mac", "de:ad:be:ef:00:00", "--kernel", "http://k")
	if res.err == nil || res.exitCode != cli.CodeNetwork {
		t.Errorf("err=%v exit=%d, want CodeNetwork", res.err, res.exitCode)
	}
}
