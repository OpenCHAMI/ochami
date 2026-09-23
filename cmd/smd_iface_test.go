// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_iface_test.go covers the success paths of the "smd iface" verbs (get,
// add, delete): the get filter/query builder arms, the --id and --by-ip
// branches, output-format variants, payload input variants, and the delete
// --all/args/-d selection with confirmation. HTTP and network error mapping,
// rejected input, and the declined confirmation are covered in
// smd_iface_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestSMDIfaceGet_Filters verifies the query builder emits the expected filter
// query parameters.
func TestSMDIfaceGet_Filters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantKey string
		wantVal string
	}{
		{"mac", []string{"--mac", "de:ad:be:ef:00:00"}, "MACAddress", "de:ad:be:ef:00:00"},
		{"ip", []string{"--ip", "172.16.0.1"}, "IPAddress", "172.16.0.1"},
		{"net", []string{"--net", "NMN"}, "Network", "NMN"},
		{"comp-id", []string{"--comp-id", "x0c0s0b0n0"}, "ComponentID", "x0c0s0b0n0"},
		{"type", []string{"--type", "Node"}, "Type", "Node"},
		{"older-than", []string{"--older-than", "2020-01-01T00:00:00Z"}, "OlderThan", "2020-01-01T00:00:00Z"},
		{"newer-than", []string{"--newer-than", "2020-01-01T00:00:00Z"}, "NewerThan", "2020-01-01T00:00:00Z"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotQuery url.Values
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotQuery = r.URL.Query()
				w.Write([]byte(`[]`))
			}))
			defer srv.Close()

			args := append([]string{"smd", "--ignore-config", "iface", "get", "--uri", srv.URL, "--token", "t"}, tc.args...)

			t.Parallel()
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

// TestSMDIfaceGet_Formats verifies the output-format variants.
func TestSMDIfaceGet_Formats(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"ComponentID":"x0c0s0b0n0","MACAddress":"de:ad:be:ef:00:00"}]`))
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchamiWithRuntime(t, "smd", "--ignore-config", "iface", "get", "--uri", srv.URL, "--token", "t", "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
		assertFormattedOutput(t, f, res.stdout, "ComponentID", "x0c0s0b0n0")
	}
}

// TestSMDIfaceGet_ByID verifies "get --id" targets the by-ID endpoint. The
// command validates the token, so a real JWT is supplied.
func TestSMDIfaceGet_ByID(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "iface", "get", "--uri", srv.URL,
		"--token", validToken(t), "--id", "decafc0ffeee")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(gotPath, "decafc0ffeee") {
		t.Errorf("path = %q, want it to reference the interface id", gotPath)
	}
}

// TestSMDIfaceGet_ByIDWithByIP verifies "get --id --by-ip" targets the IP-address
// subpath.
func TestSMDIfaceGet_ByIDWithByIP(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "iface", "get", "--uri", srv.URL,
		"--token", validToken(t), "--id", "decafc0ffeee", "--by-ip")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(gotPath, "IPAddresses") {
		t.Errorf("path = %q, want it to reference IPAddresses", gotPath)
	}
}

// TestSMDIfaceAdd_ByFlags verifies "add <comp> <mac> <net,ip>" issues a POST.
func TestSMDIfaceAdd_ByFlags(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "iface", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"x3000c1s7b55n0", "de:ca:fc:0f:fe:ee", "NMN,172.16.0.55")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost || !strings.Contains(gotPath, "EthernetInterfaces") {
		t.Errorf("request = %s %s, want POST under EthernetInterfaces", gotMethod, gotPath)
	}
}

// TestSMDIfaceAdd_ByData verifies "add -d <payload>" issues a POST.
func TestSMDIfaceAdd_ByData(t *testing.T) {
	t.Parallel()

	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "iface", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"-d", `[{"ComponentID":"x0c0s0b0n0","MACAddress":"de:ad:be:ef:00:00"}]`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
}

// TestSMDIfaceDelete_ByIDs verifies "delete --no-confirm <id>..." issues a DELETE
// per interface.
func TestSMDIfaceDelete_ByIDs(t *testing.T) {
	t.Parallel()

	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "iface", "delete", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "decafc0ffeee", "de:ad:be:ee:ee:ef")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 2 {
		t.Errorf("DELETE count = %d, want 2", deletes)
	}
}

// TestSMDIfaceDelete_AllConfirm verifies "delete --all" prompts and, on "y",
// issues a single DELETE to the collection endpoint.
func TestSMDIfaceDelete_AllConfirm(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			gotMethod, gotPath = r.Method, r.URL.Path
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInputAndRuntime(t, "y\n",
		"smd", "--ignore-config", "iface", "delete", "--uri", srv.URL, "--token", "t", "--all")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete || !strings.Contains(gotPath, "EthernetInterfaces") {
		t.Errorf("request = %s %s, want DELETE under EthernetInterfaces", gotMethod, gotPath)
	}
	if !strings.Contains(res.stdout, "ALL ETHERNET INTERFACES") {
		t.Errorf("stdout = %q, want the all-interfaces confirmation prompt", res.stdout)
	}
}

// TestSMDIfaceDelete_ByData verifies IDs in a payload drive DELETE requests.
func TestSMDIfaceDelete_ByData(t *testing.T) {
	t.Parallel()

	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "iface", "delete", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "-d", `[{"ID":"decafc0ffeee"}]`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 1 {
		t.Errorf("DELETE count = %d, want 1", deletes)
	}
}
