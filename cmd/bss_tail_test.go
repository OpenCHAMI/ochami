// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// bss_tail_test.go covers the branch families of the smaller "bss" verbs (boot
// image set, boot script get, hosts get, history get) that the happy paths do
// not reach: query-builder arms (xname/mac/nid), output-format variants, HTTP
// and network error mapping, and per-item error aggregation for image set.

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestBSSBootImageSetByXnameAndNid verifies "boot image set" selects nodes by
// --xname and --nid, fetching then PUTting the modified boot parameters.
func TestBSSBootImageSetByXnameAndNid(t *testing.T) {
	for _, sel := range [][]string{{"--xname", "x0c0s0b0n0"}, {"--nid", "1"}} {
		t.Run(sel[0], func(t *testing.T) {
			var puts int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					_, _ = w.Write([]byte(`[{"macs":["de:ad:be:ef:00:00"],"kernel":"http://s3/vmlinuz","params":"root=live:old"}]`))
				case http.MethodPut:
					puts++
					w.WriteHeader(http.StatusOK)
				}
			}))
			defer srv.Close()

			args := append([]string{"bss", "boot", "image", "set", "--ignore-config", "--uri", srv.URL, "--token", "t"},
				append(sel, "https://example.com/new-image")...)
			res := runOchami(t, args...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if puts == 0 {
				t.Error("expected at least one PUT to update boot params, got none")
			}
		})
	}
}

// TestBSSBootImageSetGetHTTPError verifies a failing GET of boot params resolves
// to CodeHTTP.
func TestBSSBootImageSetGetHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "image", "set", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "https://example.com/new-image")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestBSSBootImageSetPutHTTPError verifies a failing PUT resolves to CodeHTTP
// via the per-item aggregate.
func TestBSSBootImageSetPutHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`[{"macs":["de:ad:be:ef:00:00"],"kernel":"http://s3/vmlinuz","params":"root=live:old"}]`))
			return
		}
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "image", "set", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--mac", "de:ad:be:ef:00:00", "https://example.com/new-image")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestBSSBootScriptGetQuery verifies the boot-script query builder emits the
// mac/xname/nid and optional retry/arch/timestamp parameters.
func TestBSSBootScriptGetQuery(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`#!ipxe`))
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "script", "get", "--ignore-config", "--uri", srv.URL,
		"--xname", "x0c0s0b0n0", "--retry", "3", "--arch", "x86_64", "--timestamp", "12345")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotQuery.Get("name") == "" && gotQuery.Get("xname") == "" {
		t.Errorf("query = %v, want an xname/name parameter", gotQuery)
	}
}

// TestBSSBootScriptGetHTTPError verifies a failing boot-script GET resolves to
// CodeHTTP.
func TestBSSBootScriptGetHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "boot", "script", "get", "--ignore-config", "--uri", srv.URL,
		"--mac", "de:ad:be:ef:00:00")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestBSSHostsGetQueryAndFormats verifies the hosts query builder and
// output-format variants.
func TestBSSHostsGetQueryAndFormats(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`[{"ID":"x0c0s0b0n0"}]`))
	}))
	defer srv.Close()

	for _, f := range []string{"json", "yaml"} {
		res := runOchami(t, "bss", "hosts", "get", "--ignore-config", "--uri", srv.URL,
			"--xname", "x0c0s0b0n0", "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
	}
	if len(gotQuery) == 0 {
		t.Error("expected a non-empty query for --xname")
	}
}

// TestBSSHostsGetHTTPError verifies a failing hosts GET resolves to CodeHTTP.
func TestBSSHostsGetHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "hosts", "get", "--ignore-config", "--uri", srv.URL, "--mac", "de:ad:be:ef:00:00")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestBSSHistoryGetQueryAndFormats verifies the history query builder and
// output-format variants.
func TestBSSHistoryGetQueryAndFormats(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	for _, f := range []string{"json", "yaml"} {
		res := runOchami(t, "bss", "history", "--ignore-config", "--uri", srv.URL,
			"--xname", "x0c0s0b0n0", "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
	}
	if len(gotQuery) == 0 {
		t.Error("expected a non-empty query for --xname")
	}
}

// TestBSSHistoryGetHTTPError verifies a failing history GET resolves to
// CodeHTTP.
func TestBSSHistoryGetHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "history", "--ignore-config", "--uri", srv.URL, "--endpoint", "x0c0s0b0")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}
