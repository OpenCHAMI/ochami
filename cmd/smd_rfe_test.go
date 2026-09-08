// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_rfe_test.go covers the branch families of the "smd rfe" verbs (get, add,
// delete): the get filter/query builder arms, output-format variants, HTTP and
// network error mapping, add flag/payload variants, and the delete
// --all/args/-d selection with confirmation and per-item aggregation.

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestSMDRFEGetFilters verifies the query builder emits the expected filter
// query parameters for each flag.
func TestSMDRFEGetFilters(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantKey string
		wantVal string
	}{
		{"xname", []string{"--xname", "x3000c1s7b56"}, "id", "x3000c1s7b56"},
		{"mac", []string{"--mac", "de:ca:fc:0f:fe:ee"}, "macaddr", "de:ca:fc:0f:fe:ee"},
		{"ip", []string{"--ip", "172.16.0.156"}, "ipaddress", "172.16.0.156"},
		{"fqdn", []string{"--fqdn", "bmc.example.com"}, "fqdn", "bmc.example.com"},
		{"type", []string{"--type", "NodeBMC"}, "type", "NodeBMC"},
		{"uuid", []string{"--uuid", "abcd"}, "uuid", "abcd"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotQuery url.Values
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotQuery = r.URL.Query()
				_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			args := append([]string{"smd", "rfe", "get", "--ignore-config", "--uri", srv.URL, "--token", "t"}, tc.args...)
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

// TestSMDRFEGetFormats verifies the output-format variants.
func TestSMDRFEGetFormats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"RedfishEndpoints":[{"ID":"x3000c1s7b56"}]}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchami(t, "smd", "rfe", "get", "--ignore-config", "--uri", srv.URL, "--token", "t", "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
		if !strings.Contains(res.stdout, "x3000c1s7b56") {
			t.Errorf("format %s: stdout = %q, want it to contain the endpoint id", f, res.stdout)
		}
	}
}

// TestSMDRFEGetHTTPError verifies an unsuccessful HTTP response resolves to
// CodeHTTP.
func TestSMDRFEGetHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "rfe", "get", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDRFEGetNetworkError verifies pointing at a closed port resolves to
// CodeNetwork.
func TestSMDRFEGetNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res := runOchami(t, "smd", "rfe", "get", "--ignore-config", "--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestSMDRFEAddByFlagsWithOptional verifies "add" with the optional
// domain/hostname/username/password flags issues a POST.
func TestSMDRFEAddByFlagsWithOptional(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "rfe", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--domain", "example.com", "--hostname", "bmc56", "--username", "root", "--password", "pw",
		"x3000c1s7b56", "bmc-node56", "172.16.0.156", "de:ca:fc:0f:fe:ee")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
}

// TestSMDRFEAddByData verifies "add -d <payload>" issues a POST.
func TestSMDRFEAddByData(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "rfe", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"-d", `{"RedfishEndpoints":[{"ID":"x3000c1s7b56","Name":"bmc","IPAddress":"172.16.0.156","MACAddr":"de:ca:fc:0f:fe:ee"}]}`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
}

// TestSMDRFEAddWrongArgs verifies that fewer than 4 args without -d is a usage
// error.
func TestSMDRFEAddWrongArgs(t *testing.T) {
	res := runOchami(t, "smd", "rfe", "add", "--ignore-config", "--uri", "http://127.0.0.1:1", "--token", "t",
		"x3000c1s7b56", "bmc-node56")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestSMDRFEAddHTTPError verifies a failing POST resolves to CodeHTTP.
func TestSMDRFEAddHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "rfe", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"x3000c1s7b56", "bmc-node56", "172.16.0.156", "de:ca:fc:0f:fe:ee")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDRFEDeleteByXnames verifies "delete --no-confirm <xname>..." issues a
// DELETE per endpoint.
func TestSMDRFEDeleteByXnames(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "rfe", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "x3000c1s7b56", "x3000c1s7b57")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 2 {
		t.Errorf("DELETE count = %d, want 2", deletes)
	}
}

// TestSMDRFEDeleteByData verifies IDs in a payload drive DELETE requests.
func TestSMDRFEDeleteByData(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "rfe", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "-d", `{"RedfishEndpoints":[{"ID":"x3000c1s7b56"}]}`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 1 {
		t.Errorf("DELETE count = %d, want 1", deletes)
	}
}

// TestSMDRFEDeleteAllConfirm verifies "delete --all" prompts and, on "y", issues
// a DELETE to the collection endpoint.
func TestSMDRFEDeleteAllConfirm(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			gotMethod = r.Method
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "y\n",
		"smd", "rfe", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t", "--all")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}

// TestSMDRFEDeleteAbort verifies answering "n" aborts without a request.
func TestSMDRFEDeleteAbort(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "n\n",
		"smd", "rfe", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t", "x3000c1s7b56")
	if res.err != nil {
		t.Fatalf("unexpected error on abort: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0", deletes)
	}
}

// TestSMDRFEDeleteNoSelector verifies delete with neither -d, --all, nor args is
// a usage error.
func TestSMDRFEDeleteNoSelector(t *testing.T) {
	res := runOchami(t, "smd", "rfe", "delete", "--ignore-config", "--uri", "http://127.0.0.1:1",
		"--token", "t", "--no-confirm")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestSMDRFEDeleteByXnamesHTTPError verifies a failing per-item DELETE resolves
// to CodeHTTP via the aggregate.
func TestSMDRFEDeleteByXnamesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "rfe", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "x3000c1s7b56")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}
