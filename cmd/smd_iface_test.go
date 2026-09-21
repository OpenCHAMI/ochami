// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_iface_test.go covers the branch families of the "smd iface" verbs (get,
// add, delete) that the happy paths do not reach: the get filter/query builder
// arms, the --id and --by-ip branches, output-format variants, HTTP and network
// error mapping, payload input variants, the delete --all/args/-d selection,
// the confirmation prompt, and per-item error aggregation.

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestSMDIfaceGet_Filters verifies the query builder emits the expected filter
// query parameters.
func TestSMDIfaceGet_Filters(t *testing.T) {
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
				_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			args := append([]string{"smd", "iface", "get", "--ignore-config", "--uri", srv.URL, "--token", "t"}, tc.args...)
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

// TestSMDIfaceGet_Formats verifies the output-format variants.
func TestSMDIfaceGet_Formats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"ComponentID":"x0c0s0b0n0","MACAddress":"de:ad:be:ef:00:00"}]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchami(t, "smd", "iface", "get", "--ignore-config", "--uri", srv.URL, "--token", "t", "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
		if !strings.Contains(res.stdout, "x0c0s0b0n0") {
			t.Errorf("format %s: stdout = %q, want it to contain component id", f, res.stdout)
		}
	}
}

// TestSMDIfaceGet_ByID verifies "get --id" targets the by-ID endpoint. The
// command validates the token, so a real JWT is supplied.
func TestSMDIfaceGet_ByID(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "iface", "get", "--ignore-config", "--uri", srv.URL,
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
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "iface", "get", "--ignore-config", "--uri", srv.URL,
		"--token", validToken(t), "--id", "decafc0ffeee", "--by-ip")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(gotPath, "IPAddresses") {
		t.Errorf("path = %q, want it to reference IPAddresses", gotPath)
	}
}

// TestSMDIfaceGet_ByIPWithoutID verifies "--by-ip" without "--id" is a usage
// error.
func TestSMDIfaceGet_ByIPWithoutID(t *testing.T) {
	res := runOchami(t, "smd", "iface", "get", "--ignore-config", "--uri", "http://127.0.0.1:1",
		"--token", "t", "--by-ip")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestSMDIfaceGet_HTTPError verifies an unsuccessful HTTP response resolves to
// CodeHTTP.
func TestSMDIfaceGet_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "iface", "get", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDIfaceGet_NetworkError verifies pointing at a closed port resolves to
// CodeNetwork.
func TestSMDIfaceGet_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res := runOchami(t, "smd", "iface", "get", "--ignore-config", "--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestSMDIfaceAdd_ByFlags verifies "add <comp> <mac> <net,ip>" issues a POST.
func TestSMDIfaceAdd_ByFlags(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "iface", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
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
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "iface", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"-d", `[{"ComponentID":"x0c0s0b0n0","MACAddress":"de:ad:be:ef:00:00"}]`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
}

// TestSMDIfaceAdd_InvalidIP verifies an invalid IP in the net,ip pair is a usage
// error.
func TestSMDIfaceAdd_InvalidIP(t *testing.T) {
	res := runOchami(t, "smd", "iface", "add", "--ignore-config", "--uri", "http://127.0.0.1:1", "--token", "t",
		"x3000c1s7b55n0", "de:ca:fc:0f:fe:ee", "NMN,not-an-ip")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestSMDIfaceAdd_HTTPError verifies a failing POST resolves to CodeHTTP via the
// per-item aggregation.
func TestSMDIfaceAdd_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "iface", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"x3000c1s7b55n0", "de:ca:fc:0f:fe:ee", "NMN,172.16.0.55")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDIfaceDelete_ByIDs verifies "delete --no-confirm <id>..." issues a DELETE
// per interface.
func TestSMDIfaceDelete_ByIDs(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "iface", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
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
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			gotMethod, gotPath = r.Method, r.URL.Path
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "y\n",
		"smd", "iface", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t", "--all")
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

// TestSMDIfaceDelete_Abort verifies answering "n" aborts without a request.
func TestSMDIfaceDelete_Abort(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "n\n",
		"smd", "iface", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t", "decafc0ffeee")
	if res.err != nil {
		t.Fatalf("unexpected error on abort: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0", deletes)
	}
}

// TestSMDIfaceDelete_ByData verifies IDs in a payload drive DELETE requests.
func TestSMDIfaceDelete_ByData(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "iface", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "-d", `[{"ID":"decafc0ffeee"}]`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 1 {
		t.Errorf("DELETE count = %d, want 1", deletes)
	}
}

// TestSMDIfaceDelete_NoSelector verifies delete with neither -d, --all, nor args
// is a usage error.
func TestSMDIfaceDelete_NoSelector(t *testing.T) {
	res := runOchami(t, "smd", "iface", "delete", "--ignore-config", "--uri", "http://127.0.0.1:1",
		"--token", "t", "--no-confirm")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestSMDIfaceDelete_AllHTTPError verifies a failing "delete --all" resolves to
// CodeHTTP.
func TestSMDIfaceDelete_AllHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "iface", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "--all")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDIfaceDelete_ByIDsHTTPError verifies a failing per-item DELETE resolves
// to CodeHTTP via the aggregate.
func TestSMDIfaceDelete_ByIDsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "iface", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "decafc0ffeee")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}
