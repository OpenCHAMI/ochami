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

// TestSMDRFEGet_Filters verifies the query builder emits the expected filter
// query parameters for each flag.
func TestSMDRFEGet_Filters(t *testing.T) {
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

			args := append([]string{"--ignore-config", "smd", "rfe", "get", "--uri", srv.URL, "--token", "t"}, tc.args...)
			res := runOchamiWithRuntime(t, args...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if got := gotQuery.Get(tc.wantKey); got != tc.wantVal {
				t.Errorf("query %s = %q, want %q", tc.wantKey, got, tc.wantVal)
			}
		})
	}
}

// TestSMDRFEGet_Formats verifies the output-format variants.
func TestSMDRFEGet_Formats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"RedfishEndpoints":[{"ID":"x3000c1s7b56"}]}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchamiWithRuntime(t, "smd", "rfe", "get", "--uri", srv.URL, "--token", "t", "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
		if !strings.Contains(res.stdout, "x3000c1s7b56") {
			t.Errorf("format %s: stdout = %q, want it to contain the endpoint id", f, res.stdout)
		}
	}
}

// TestSMDRFEGet_HTTPError verifies an unsuccessful HTTP response resolves to
// CodeHTTP.
func TestSMDRFEGet_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "rfe", "get", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDRFEGet_NetworkError verifies pointing at a closed port resolves to
// CodeNetwork.
func TestSMDRFEGet_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res := runOchamiWithRuntime(t, "smd", "rfe", "get", "--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestSMDRFEAdd_ByFlagsWithOptional verifies "add" with the optional
// domain/hostname/username/password flags issues a POST.
func TestSMDRFEAdd_ByFlagsWithOptional(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "rfe", "add", "--uri", srv.URL, "--token", "t",
		"--domain", "example.com", "--hostname", "bmc56", "--username", "root", "--password", "pw",
		"x3000c1s7b56", "bmc-node56", "172.16.0.156", "de:ca:fc:0f:fe:ee")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
}

// TestSMDRFEAdd_ByData verifies "add -d <payload>" issues a POST.
func TestSMDRFEAdd_ByData(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "rfe", "add", "--uri", srv.URL, "--token", "t",
		"-d", `{"RedfishEndpoints":[{"ID":"x3000c1s7b56","Name":"bmc","IPAddress":"172.16.0.156","MACAddr":"de:ca:fc:0f:fe:ee"}]}`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
}

// TestSMDRFEAdd_WrongArgs verifies that fewer than 4 args without -d is a usage
// error.
func TestSMDRFEAdd_WrongArgs(t *testing.T) {
	res := runOchamiWithRuntime(t, "smd", "rfe", "add", "--uri", "http://127.0.0.1:1", "--token", "t",
		"x3000c1s7b56", "bmc-node56")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestSMDRFEAdd_HTTPError verifies a failing POST resolves to CodeHTTP.
func TestSMDRFEAdd_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "rfe", "add", "--uri", srv.URL, "--token", "t",
		"x3000c1s7b56", "bmc-node56", "172.16.0.156", "de:ca:fc:0f:fe:ee")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDRFEDelete_ByXnames verifies "delete --no-confirm <xname>..." issues a
// DELETE per endpoint.
func TestSMDRFEDelete_ByXnames(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "rfe", "delete", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "x3000c1s7b56", "x3000c1s7b57")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 2 {
		t.Errorf("DELETE count = %d, want 2", deletes)
	}
}

// TestSMDRFEDelete_ByData verifies IDs in a payload drive DELETE requests.
func TestSMDRFEDelete_ByData(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "rfe", "delete", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "-d", `{"RedfishEndpoints":[{"ID":"x3000c1s7b56"}]}`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 1 {
		t.Errorf("DELETE count = %d, want 1", deletes)
	}
}

// TestSMDRFEDelete_AllConfirm verifies "delete --all" prompts and, on "y", issues
// a DELETE to the collection endpoint.
func TestSMDRFEDelete_AllConfirm(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			gotMethod = r.Method
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInputAndRuntime(t, "y\n",
		"--ignore-config", "smd", "rfe", "delete", "--uri", srv.URL, "--token", "t", "--all")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}

// TestSMDRFEDelete_Abort verifies answering "n" aborts without a request.
func TestSMDRFEDelete_Abort(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInputAndRuntime(t, "n\n",
		"--ignore-config", "smd", "rfe", "delete", "--uri", srv.URL, "--token", "t", "x3000c1s7b56")
	if res.err != nil {
		t.Fatalf("unexpected error on abort: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0", deletes)
	}
}

// TestSMDRFEDelete_NoSelector verifies delete with neither -d, --all, nor args is
// a usage error.
func TestSMDRFEDelete_NoSelector(t *testing.T) {
	res := runOchamiWithRuntime(t, "smd", "rfe", "delete", "--uri", "http://127.0.0.1:1",
		"--token", "t", "--no-confirm")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestSMDRFEDelete_ByXnamesHTTPError verifies a failing per-item DELETE resolves
// to CodeHTTP via the aggregate.
func TestSMDRFEDelete_ByXnamesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "rfe", "delete", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "x3000c1s7b56")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}
