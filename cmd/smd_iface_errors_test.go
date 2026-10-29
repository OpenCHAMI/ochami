// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestSMDIfaceGet_ByIPWithoutID verifies "--by-ip" without "--id" is a usage
// error.
func TestSMDIfaceGet_ByIPWithoutID(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "iface", "get", "--uri", "http://127.0.0.1:1",
		"--token", "t", "--by-ip")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestSMDIfaceGet_HTTPError verifies an unsuccessful HTTP response resolves to
// CodeHTTP.
func TestSMDIfaceGet_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "iface", "get", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDIfaceGet_NetworkError verifies pointing at a closed port resolves to
// CodeNetwork.
func TestSMDIfaceGet_NetworkError(t *testing.T) {
	t.Parallel()

	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "iface", "get", "--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}

// TestSMDIfaceAdd_InvalidIP verifies an invalid IP in the net,ip pair is a usage
// error.
func TestSMDIfaceAdd_InvalidIP(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "iface", "add", "--ignore-config", "--uri", "http://127.0.0.1:1", "--token", "t",
		"x3000c1s7b55n0", "de:ca:fc:0f:fe:ee", "NMN,not-an-ip")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestSMDIfaceAdd_InvalidNetworkPairs verifies malformed pairs and missing
// positional arguments return usage errors without sending a request.
func TestSMDIfaceAdd_InvalidNetworkPairs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"no arguments", nil, "expected -d or >= 3 arguments"},
		{"component only", []string{"node"}, "expected -d or >= 3 arguments"},
		{"no network", []string{"node", "de:ca:fc:0f:fe:ee"}, "expected -d or >= 3 arguments"},
		{"missing comma", []string{"node", "de:ca:fc:0f:fe:ee", "external"}, "invalid network,ip pair"},
		{"empty network", []string{"node", "de:ca:fc:0f:fe:ee", ",10.1.0.55"}, "invalid network,ip pair"},
		{"empty IP", []string{"node", "de:ca:fc:0f:fe:ee", "external,"}, "invalid IP address"},
		{"extra comma", []string{"node", "de:ca:fc:0f:fe:ee", "external,10.1.0.55,internal"}, "invalid IP address"},
		{"IPv6", []string{"node", "de:ca:fc:0f:fe:ee", "external,2001:db8::1"}, "invalid IP address"},
		{"malformed second pair", []string{"node", "de:ca:fc:0f:fe:ee", "external,10.1.0.55", "internal"}, "invalid network,ip pair"},
		{"invalid second IP", []string{"node", "de:ca:fc:0f:fe:ee", "external,10.1.0.55", "internal,not-an-ip"}, "invalid IP address"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var requests int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.WriteHeader(http.StatusCreated)
			}))
			defer srv.Close()

			args := append([]string{"smd", "--ignore-config", "iface", "add", "--uri", srv.URL, "--token", "t"}, tc.args...)
			res := runOchamiWithRuntime(t, args...)
			if res.err == nil || !strings.Contains(res.err.Error(), tc.want) {
				t.Errorf("error = %v, want %q", res.err, tc.want)
			}
			if res.exitCode != cli.CodeUsage {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
			}
			if requests != 0 {
				t.Errorf("request count = %d, want 0", requests)
			}
		})
	}
}

// TestSMDIfaceAdd_HTTPError verifies a failing POST resolves to CodeHTTP via the
// per-item aggregation.
func TestSMDIfaceAdd_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "iface", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"x3000c1s7b55n0", "de:ca:fc:0f:fe:ee", "NMN,172.16.0.55")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDIfaceDelete_Abort verifies answering "n" aborts without a request.
func TestSMDIfaceDelete_Abort(t *testing.T) {
	t.Parallel()

	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInputAndRuntime(t, "n\n",
		"smd", "--ignore-config", "iface", "delete", "--uri", srv.URL, "--token", "t", "decafc0ffeee")
	if res.exitCode != cli.CodeDeclined {
		t.Fatalf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodeDeclined, cli.CodeName(cli.CodeDeclined))
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0", deletes)
	}
}

// TestSMDIfaceDelete_NoSelector verifies delete with neither -d, --all, nor args
// is a usage error.
func TestSMDIfaceDelete_NoSelector(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "iface", "delete", "--uri", "http://127.0.0.1:1",
		"--token", "t", "--no-confirm")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestSMDIfaceDelete_AllHTTPError verifies a failing "delete --all" resolves to
// CodeHTTP.
func TestSMDIfaceDelete_AllHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "iface", "delete", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "--all")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDIfaceDelete_ByIDsHTTPError verifies a failing per-item DELETE resolves
// to CodeHTTP via the aggregate.
func TestSMDIfaceDelete_ByIDsHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "iface", "delete", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "decafc0ffeee")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDIfaceDelete_AllFalseNoSelector verifies "delete --all=false" with
// neither -d nor arguments is a usage error, as if --all were omitted.
func TestSMDIfaceDelete_AllFalseNoSelector(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "iface", "delete", "--uri", "http://127.0.0.1:1",
		"--token", "t", "--no-confirm", "--all=false")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestSMDIfaceDelete_AllWithSelector verifies "delete --all" with -d or arguments
// is a usage error that sends no request, so a stray selector cannot widen
// deletion to everything.
func TestSMDIfaceDelete_AllWithSelector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"argument", []string{"decafc0ffeee"}, "--all cannot be used with arguments"},
		{"invalid argument", []string{"invalid/id"}, "--all cannot be used with arguments"},
		{"payload", []string{"-d", `[{"ID":"decafc0ffeee"}]`}, "--all cannot be used with -d"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var requests int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			args := append([]string{"smd", "--ignore-config", "iface", "delete", "--uri", srv.URL, "--token", "t",
				"--no-confirm", "--all"}, tc.args...)
			res := runOchamiWithRuntime(t, args...)
			if res.err == nil || !strings.Contains(res.err.Error(), tc.want) {
				t.Errorf("error = %v, want %q", res.err, tc.want)
			}
			if res.exitCode != cli.CodeUsage {
				t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
			}
			if requests != 0 {
				t.Errorf("request count = %d, want 0", requests)
			}
		})
	}
}
