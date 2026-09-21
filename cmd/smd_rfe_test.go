// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_rfe_test.go covers the success paths of the "smd rfe" verbs (get, add,
// delete): the get filter/query builder arms, output-format variants, add
// flag/payload variants, and the delete --all/args/-d selection with
// confirmation. HTTP and network error mapping, rejected input, and the
// declined confirmation are covered in smd_rfe_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
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
				w.Write([]byte(`{}`))
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

// TestSMDRFEGet_Formats verifies the output-format variants.
func TestSMDRFEGet_Formats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"RedfishEndpoints":[{"ID":"x3000c1s7b56"}]}`))
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchami(t, "smd", "rfe", "get", "--ignore-config", "--uri", srv.URL, "--token", "t", "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
		assertFormattedOutput(t, f, res.stdout, "ID", "x3000c1s7b56")
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

// TestSMDRFEAdd_ByData verifies "add -d <payload>" issues a POST.
func TestSMDRFEAdd_ByData(t *testing.T) {
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

	res := runOchami(t, "smd", "rfe", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
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

	res := runOchami(t, "smd", "rfe", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
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

	res := runOchamiWithInput(t, "y\n",
		"smd", "rfe", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t", "--all")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "Really delete ALL REDFISH ENDPOINTS?") {
		t.Errorf("output = %q, want the confirmation prompt", res.stdout)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}
